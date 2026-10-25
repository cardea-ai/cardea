// Package gateway는 GW-01(듀얼 프로토콜 엔드포인트)의 최소 골격이다.
//
// 요청 처리 순서는 CLAUDE.md 0장의 원칙을 그대로 따른다: 정규화 → 탐지 →
// (분류/정책) → 처리(마스킹) → 전달. 전달(Forwarder)은 환경변수로 백엔드를
// 고른다 — 기본값은 StubForwarder(502, Bifrost 비연동 상태를 숨기지
// 않음)이고, CARDEA_ENABLE_BIFROST=1이면 Bifrost core(ADR-0001)를 실제로
// 초기화해서 쓴다. 마스킹(Processor)은 Forward 호출보다 앞에 있어야
// 한다는 순서는 백엔드가 뭐든 동일하게 강제된다.
package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/cardea-ai/cardea/internal/detect"
	"github.com/cardea-ai/cardea/internal/forward"
	"github.com/cardea-ai/cardea/internal/plugin"
	"github.com/cardea-ai/cardea/internal/policy"
	"github.com/cardea-ai/cardea/internal/presidio"
	"github.com/cardea-ai/cardea/internal/processor"
	"github.com/cardea-ai/cardea/internal/vault"
)

// Server는 OpenAI(Chat Completions)와 Anthropic(Messages) 엔드포인트를
// 함께 제공한다(GW-01).
type Server struct {
	mux        *http.ServeMux
	normalizer plugin.Normalizer
	detectors  []plugin.Detector
	policy     plugin.PolicyEngine
	processor  plugin.Processor
	forwarder  forward.Forwarder
}

// newVault는 환경변수로 토큰 금고 백엔드를 고른다. CARDEA_OPENBAO_ADDR가
// 비어 있으면 MemoryVault, 설정돼 있으면 OpenBao에 연결한다. OpenBao
// 연결이 설정됐는데 실패하면 에러를 그대로 올려 fail-closed 기동 중단으로
// 이어지게 한다 — 조용히 MemoryVault로 떨어지지 않는다.
func newVault() (vault.Vault, error) {
	addr := os.Getenv("CARDEA_OPENBAO_ADDR")
	if addr == "" {
		return vault.NewMemoryVault()
	}
	keyName := os.Getenv("CARDEA_OPENBAO_KEY")
	if keyName == "" {
		keyName = "cardea-vault-key"
	}
	return vault.NewOpenBaoVault(context.Background(), vault.OpenBaoConfig{
		Address: addr,
		Token:   os.Getenv("CARDEA_OPENBAO_TOKEN"),
		KeyName: keyName,
	})
}

// newForwarder는 환경변수로 전달 엔진 백엔드를 고른다.
// CARDEA_ENABLE_BIFROST=1이면 Bifrost core를 초기화해서 쓰고, 아니면
// StubForwarder로 떨어진다(502로 미구현 상태를 그대로 보여준다 — 조용히
// 가짜 성공을 만들지 않는다). Bifrost 초기화가 명시적으로 켜졌는데
// 실패하면 fail-closed로 기동을 멈춘다.
func newForwarder(ctx context.Context) (forward.Forwarder, error) {
	if os.Getenv("CARDEA_ENABLE_BIFROST") != "1" {
		return forward.StubForwarder{}, nil
	}
	return forward.NewBifrostForwarder(ctx)
}

// newDetectors는 체크섬 리코그나이저(항상 켜짐)에, 환경변수가 설정돼
// 있으면 Presidio REST 클라이언트(ADR-0007, 체크섬 없는 엔티티용)를
// 추가한다. CARDEA_PRESIDIO_ADDR가 설정됐는데 Presidio가 응답하지 않으면
// fail-closed로 기동을 막는다 — 조용히 체크섬 탐지기만 쓰는 쪽으로
// 내려가지 않는다(켜겠다고 했으면 실제로 떠 있어야 한다).
func newDetectors(ctx context.Context) ([]plugin.Detector, error) {
	detectors := []plugin.Detector{detect.ChecksumRecognizer{}}

	addr := os.Getenv("CARDEA_PRESIDIO_ADDR")
	if addr == "" {
		return detectors, nil
	}
	lang := os.Getenv("CARDEA_PRESIDIO_LANGUAGE")
	if lang == "" {
		lang = "en" // 공식 Docker 이미지 기본 NLP 엔진 — 한국어는 아직 미지원(presidio 패키지 주석 참고)
	}
	client := presidio.NewClient(addr, lang)
	if err := client.Healthy(ctx); err != nil {
		return nil, fmt.Errorf("gateway: Presidio 연동이 켜졌는데 응답 없음(fail-closed): %w", err)
	}
	return append(detectors, client), nil
}

// New는 기본 플러그인 체인(정규화기 1종, 체크섬 탐지기 1종, 기본 정책,
// 토큰 금고 기반 Processor, Bifrost 임베딩 전 StubForwarder)으로 서버를
// 구성한다.
//
// 토큰 금고는 환경변수로 백엔드를 고른다: CARDEA_OPENBAO_ADDR가 설정돼
// 있으면 OpenBao(ADR-0007)를 쓰고, 아니면 MemoryVault로 떨어진다(로컬
// 개발/단일 사용자 무료 티어 기본값 — 프로세스 재시작 시 매핑이 사라지는
// 걸 감수한다). OpenBao 연결이 설정됐는데 실패하면 fail-closed로 기동을
// 멈춘다 — MemoryVault로 조용히 되돌아가지 않는다.
//
// 정책 엔진 컴파일이나 토큰 금고 초기화가 실패하면 에러를 반환한다 —
// 호출부(cmd/cardea)가 fail-closed로 기동을 중단해야 한다.
func New() (*Server, error) {
	eng, err := policy.New(context.Background())
	if err != nil {
		return nil, err
	}

	v, err := newVault()
	if err != nil {
		return nil, err
	}

	fw, err := newForwarder(context.Background())
	if err != nil {
		return nil, err
	}

	detectors, err := newDetectors(context.Background())
	if err != nil {
		return nil, err
	}

	s := &Server{
		mux:        http.NewServeMux(),
		normalizer: detect.Normalizer{},
		detectors:  detectors,
		policy:     eng,
		processor:  &processor.TokenProcessor{Vault: v},
		forwarder:  fw,
	}
	s.mux.HandleFunc("/v1/chat/completions", s.handle("openai"))
	s.mux.HandleFunc("/v1/messages", s.handle("anthropic"))
	s.mux.HandleFunc("/healthz", s.healthz)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// newRequestScope는 요청 단위 스코프 id를 발급한다. FR-MEM 세션 개념이
// 들어오기 전까지의 자리값 — 지금은 매 요청이 독립 스코프라 참조 일관성은
// 한 요청 본문 내부에서만 보장된다(동일 요청 안에서 같은 원문이 여러 번
// 나오면 같은 토큰을 받는다는 뜻). 세션이 들어오면 이 함수를
// "세션 쿠키/헤더에서 스코프를 읽는" 걸로 교체한다.
func newRequestScope() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "req_" + hex.EncodeToString(b)
}

func (s *Server) handle(protocol string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := processor.WithScope(r.Context(), newRequestScope())

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "요청 본문을 읽을 수 없음", http.StatusBadRequest)
			return
		}

		// FR-PI-11: 판정 전 정규화를 먼저 거친다.
		normalized, err := s.normalizer.Normalize(ctx, string(body))
		if err != nil {
			http.Error(w, "정규화 실패", http.StatusInternalServerError)
			return
		}

		var findings []plugin.Finding
		for _, det := range s.detectors {
			found, err := det.Detect(ctx, normalized)
			if err != nil {
				// 탐지기 오류도 fail-closed — 판정 없이 통과시키지 않는다.
				log.Printf("gateway: 탐지기 %s 오류(fail-closed): %v", det.Name(), err)
				http.Error(w, "탐지 실패", http.StatusServiceUnavailable)
				return
			}
			findings = append(findings, found...)
		}

		grade := plugin.GradePublic
		if len(findings) > 0 {
			grade = plugin.GradeConfidential
		}

		decision, err := s.policy.Evaluate(ctx, plugin.PolicyInput{Grade: grade, Findings: findings})
		if err != nil {
			log.Printf("gateway: 정책 평가 오류(fail-closed): %v", err)
			http.Error(w, "정책 평가 실패", http.StatusServiceUnavailable)
			return
		}

		// 감사 로그: 원문 없이 타입·건수·판정만 기록한다(CLAUDE.md 4장).
		log.Printf(
			"gateway: protocol=%s grade=%s findings=%d decision=%s",
			protocol, grade, len(findings), decision,
		)

		if decision == plugin.DecisionDeny {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "정책에 의해 거부됨 (fail-closed)",
			})
			return
		}

		// FR-PI-04: Forward 호출 전에 반드시 마스킹을 거친다 — partial
		// 판정이어도 평문이 Forwarder까지 그대로 넘어가지 않는다. Detect는
		// 정규화된 텍스트 기준 인덱스를 돌려주므로, 마스킹도 정규화된
		// 텍스트에 적용한다(원문 본문의 인덱스와 섞어 쓰면 깨진다).
		outbound := normalized
		if decision == plugin.DecisionPartial {
			outbound, err = s.processor.Process(ctx, decision, findings, normalized)
			if err != nil {
				log.Printf("gateway: 마스킹 처리 오류(fail-closed): %v", err)
				http.Error(w, "마스킹 처리 실패", http.StatusServiceUnavailable)
				return
			}
		}

		_, err = s.forwarder.Forward(ctx, forward.Request{Protocol: protocol, Body: []byte(outbound)})
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}
	}
}
