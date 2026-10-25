// Package policy는 ADR-0002의 "정책" 슬롯을 Rego(OPA)로 구현한다.
//
// NFR-AV-02 fail-closed: 정책 평가 자체가 실패(모듈 컴파일 오류, 쿼리 오류,
// 타임아웃 등)하면 반드시 DecisionDeny를 반환한다 — "일단 허용하고 나중에
// 막자"는 패턴은 쓰지 않는다. 이 패키지는 아직 FR-AC-03 CBAC 전체를
// 구현하지 않는다 — 최소 골격 단계에서는 "민감도 등급이 허용 범위를
// 넘는 요청을 마스킹/처리 없이 그대로 전달하지 않는다"는 가장 기초적인
// 규칙만 평가한다. CBAC의 4종 판정 입력(역할·작업·등급·컨텍스트)은 이
// 구조 위에 정책 모듈을 교체/확장해서 올린다.
package policy

import (
	"context"
	"fmt"

	"github.com/open-policy-agent/opa/v1/rego"

	"github.com/cardea-ai/cardea/internal/plugin"
)

// defaultModule은 최소 골격의 기본 정책이다: 극비 등급은 항상 deny,
// findings가 있는데 아직 처리기(Processor)를 거치지 않은 상태면 partial로
// 내려 호출부가 마스킹을 강제하게 한다.
const defaultModule = `
package cardea.gateway

default decision := "deny"

decision := "allow" if {
	input.grade <= 1
	input.findings_count == 0
}

decision := "partial" if {
	input.grade <= 2
	input.findings_count > 0
}
`

// Engine은 plugin.PolicyEngine을 구현한다.
type Engine struct {
	query rego.PreparedEvalQuery
}

// New는 기본 정책 모듈을 컴파일한다. 컴파일 실패는 즉시 에러로 반환하고,
// 호출부(cmd/cardea)는 이를 fail-closed 기동 실패로 처리한다.
func New(ctx context.Context) (*Engine, error) {
	r := rego.New(
		rego.Query("data.cardea.gateway.decision"),
		rego.Module("cardea_default.rego", defaultModule),
	)
	q, err := r.PrepareForEval(ctx)
	if err != nil {
		return nil, fmt.Errorf("policy: 기본 모듈 컴파일 실패: %w", err)
	}
	return &Engine{query: q}, nil
}

var _ plugin.PolicyEngine = (*Engine)(nil)

func (e *Engine) Evaluate(ctx context.Context, in plugin.PolicyInput) (plugin.Decision, error) {
	input := map[string]any{
		"grade":          int(in.Grade),
		"findings_count": len(in.Findings),
	}
	rs, err := e.query.Eval(ctx, rego.EvalInput(input))
	if err != nil {
		// fail-closed: 평가 자체가 깨지면 무조건 deny.
		return plugin.DecisionDeny, fmt.Errorf("policy: 평가 실패(fail-closed로 deny): %w", err)
	}
	if len(rs) == 0 || len(rs[0].Expressions) == 0 {
		return plugin.DecisionDeny, fmt.Errorf("policy: 평가 결과 없음(fail-closed로 deny)")
	}
	decision, ok := rs[0].Expressions[0].Value.(string)
	if !ok {
		return plugin.DecisionDeny, fmt.Errorf("policy: 평가 결과 타입 오류(fail-closed로 deny)")
	}
	switch plugin.Decision(decision) {
	case plugin.DecisionAllow, plugin.DecisionPartial, plugin.DecisionDeny:
		return plugin.Decision(decision), nil
	default:
		return plugin.DecisionDeny, fmt.Errorf("policy: 알 수 없는 decision %q(fail-closed로 deny)", decision)
	}
}
