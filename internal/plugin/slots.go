// Package plugin은 ADR-0002가 정의한 6종 플러그인 슬롯(정규화기 → 탐지기 →
// 분류기 → 정책 → 처리기 → 연동)의 인터페이스를 정의한다.
//
// 코어는 이 인터페이스에만 의존하고, 구체 구현(Go 내장이든 WASM/Rego/gRPC
// 플러그인이든)은 실행 시점에 레지스트리로 조립한다. 로케일별 탐지(한국
// 민감정보 유형 포함)도 "기본 플러그인"일 뿐, 코어에 하드코딩하지 않는다.
package plugin

import "context"

// Grade는 요청·데이터에 붙는 민감도 등급이다 (공개·내부·기밀·극비).
type Grade int

const (
	GradePublic Grade = iota
	GradeInternal
	GradeConfidential
	GradeTopSecret
)

func (g Grade) String() string {
	switch g {
	case GradePublic:
		return "공개"
	case GradeInternal:
		return "내부"
	case GradeConfidential:
		return "기밀"
	case GradeTopSecret:
		return "극비"
	default:
		return "알수없음"
	}
}

// Finding은 탐지기가 찾은 민감정보 후보 하나를 나타낸다.
//
// 원문(Text)은 디버깅 편의를 위해 메모리에만 잠깐 들고 있을 수 있지만,
// 호출부는 이 구조체를 로그·감사 기록으로 직렬화할 때 Text 필드를 반드시
// 제외해야 한다 (CLAUDE.md 4장 "원문을 저장하지 않는다").
type Finding struct {
	EntityType string // 예: "KR_RRN", "KR_CARD_NUMBER"
	Start      int
	End        int
	Score      float64 // 0.0~1.0, 체크섬 검증 통과 시 보통 >= 0.9
	Text       string  // 원문 매치 — 저장 금지, 토큰 발급 직전까지만 사용
}

// Normalizer는 판정 전 정규화를 수행한다 (FR-PI-11): 공백/구분자 제거,
// 한글 숫자, 전각 문자, 디코딩. 탐지기는 정규화를 거치지 않은 원문에
// 직접 정규식을 돌리면 안 된다.
type Normalizer interface {
	Normalize(ctx context.Context, text string) (string, error)
}

// Detector는 정규화된 텍스트에서 민감정보 후보를 찾는다.
type Detector interface {
	Name() string
	Detect(ctx context.Context, normalized string) ([]Finding, error)
}

// Classifier는 탐지된 Finding들을 종합해 요청 전체의 민감도 등급을 매긴다.
// 여러 등급이 섞이면 가장 높은 등급을 따르고, 기밀+개인식별정보 결합은
// 극비로 올린다 (결합 규칙, CLAUDE.md 4장).
type Classifier interface {
	Classify(ctx context.Context, findings []Finding) (Grade, error)
}

// Decision은 정책 평가 결과다.
type Decision string

const (
	DecisionAllow   Decision = "allow"
	DecisionPartial Decision = "partial"
	DecisionDeny    Decision = "deny"
)

// PolicyInput은 정책 엔진에 넘기는 판정 입력이다.
type PolicyInput struct {
	Grade    Grade
	Findings []Finding
	// Context는 세션 오염(AG-04) 등 요청 시점 신호를 담는다. 값은 문자열로만
	// 넘겨 원문이 섞여 들어가는 걸 구조적으로 막는다.
	Context map[string]string
}

// PolicyEngine은 판정을 내린다. 평가 자체가 실패하면 반드시 DecisionDeny를
// 반환해야 한다 (NFR-AV-02 fail-closed) — 호출부가 에러를 무시하고 allow로
// 넘어가는 경로를 만들지 않는다.
type PolicyEngine interface {
	Evaluate(ctx context.Context, in PolicyInput) (Decision, error)
}

// Processor는 Decision에 따라 요청/응답을 처리한다 (마스킹, 토큰화 등).
type Processor interface {
	Process(ctx context.Context, decision Decision, findings []Finding, text string) (string, error)
}

// Integration은 감사 로그·SIEM 등 외부 연동을 담당한다.
type Integration interface {
	Record(ctx context.Context, event AuditEvent) error
}

// AuditEvent는 감사 로그 한 건이다. 기본은 L0(메타데이터+해시)만 남기고
// 원문은 담지 않는다 (CLAUDE.md 4장).
type AuditEvent struct {
	RuleID     string
	EntityType string
	Grade      Grade
	Decision   Decision
}
