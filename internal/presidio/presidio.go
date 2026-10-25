// Package presidio는 Microsoft Presidio(ADR-0007)를 별도 프로세스로 붙여
// 체크섬 없는 엔티티(이름·이메일·주소 등)를 탐지한다 — `internal/detect`의
// 체크섬 리코그나이저(KR_RRN 등)가 다루지 못하는 영역이다.
//
// 수정(2026-10-25, Presidio 임베딩 중 실제로 확인함): ADR-0007·8장에는
// "별도 프로세스(gRPC)"라고 적어 뒀지만, Presidio Analyzer의 실제 공식
// Docker 이미지(mcr.microsoft.com/presidio-analyzer)는 gRPC가 아니라
// REST(Flask, `/analyze`)만 제공한다. gRPC는 Cardea 자체 ML 분류기(7장
// 기술스택)를 위한 선택이었고, Presidio라는 특정 OSS 제품의 실제 인터페이스는
// REST다 — 이 패키지는 REST로 구현하고, 문서는 이 차이를 반영해 고쳤다
// (`docs/history/requirements.md` 참고).
//
// 추가로 확인한 제약: 공식 Docker 이미지는 영어(en) NLP 엔진만 기본 탑재돼
// 있다. 한국어 요청은 "No matching recognizers were found" 에러로 거부된다
// — 한국어 지원은 별도 NLP 엔진(한국어 spaCy/다국어 transformer) 구성이
// 필요하고, 이번 작업 범위 밖이다(아래 Client.SupportedLanguages 참고).
package presidio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cardea-ai/cardea/internal/plugin"
)

// Client는 Presidio Analyzer REST API(`POST /analyze`)를 호출하는
// plugin.Detector 구현체다.
type Client struct {
	BaseURL  string // 예: "http://127.0.0.1:8301"
	Language string // 예: "en" — 공식 이미지는 기본적으로 "en"만 지원
	HTTP     *http.Client
}

var _ plugin.Detector = (*Client)(nil)

// NewClient는 기본 5초 타임아웃의 HTTP 클라이언트로 Client를 만든다.
func NewClient(baseURL, language string) *Client {
	return &Client{
		BaseURL:  baseURL,
		Language: language,
		HTTP:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) Name() string { return "PresidioRESTClient" }

type analyzeRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

type analyzeResult struct {
	EntityType string  `json:"entity_type"`
	Start      int     `json:"start"`
	End        int     `json:"end"`
	Score      float64 `json:"score"`
}

// Detect는 정규화된 텍스트를 Presidio에 그대로 보낸다 — Presidio 자체도
// 내부적으로 NLP 토큰화를 거치므로, FR-PI-11 정규화(전각/한글숫자 등)를
// 먼저 거친 입력을 주는 게 맞다.
func (c *Client) Detect(ctx context.Context, normalized string) ([]plugin.Finding, error) {
	reqBody, err := json.Marshal(analyzeRequest{Text: normalized, Language: c.Language})
	if err != nil {
		return nil, fmt.Errorf("presidio: 요청 직렬화 실패: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/analyze", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("presidio: 요청 생성 실패: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		// fail-closed: Presidio에 못 붙으면 "탐지 결과 없음"이 아니라
		// 에러로 올려서 호출부(게이트웨이)가 거부하게 한다.
		return nil, fmt.Errorf("presidio: 요청 실패(fail-closed): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("presidio: 비정상 응답 상태코드 %d", resp.StatusCode)
	}

	var results []analyzeResult
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return nil, fmt.Errorf("presidio: 응답 파싱 실패: %w", err)
	}

	findings := make([]plugin.Finding, 0, len(results))
	for _, r := range results {
		if r.Start < 0 || r.End > len(normalized) || r.Start >= r.End {
			continue // Presidio가 돌려준 인덱스가 입력 범위를 벗어나면 보수적으로 건너뛴다
		}
		findings = append(findings, plugin.Finding{
			EntityType: r.EntityType,
			Start:      r.Start,
			End:        r.End,
			Score:      r.Score,
			Text:       normalized[r.Start:r.End],
		})
	}
	return findings, nil
}

// Healthy는 Presidio 프로세스가 떠 있는지 확인한다. 게이트웨이 기동 시
// liveness-check로 쓴다 — 켜졌는데 못 붙으면 fail-closed로 기동을 막는다.
func (c *Client) Healthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("presidio: health check 실패: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("presidio: health check 비정상 상태코드 %d", resp.StatusCode)
	}
	return nil
}
