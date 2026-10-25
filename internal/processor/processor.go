// Package processor는 FR-PI-04 가역 마스킹을 구현한다 (ADR-0002의 "처리기"
// 슬롯). Mask는 탐지된 민감정보를 ⟦TYPE_난수⟧ 토큰으로 바꿔 모델로 보낼
// 본문을 만들고, Restore는 모델 응답에 섞여 돌아온 토큰을 원문으로 되돌린다.
//
// 같은 스코프(세션/요청) 안에서 같은 원문은 같은 토큰을 받는다(참조 일관성)
// — 스코프를 넘는 전역 고정 토큰은 쓰지 않는다(재식별 위험,
// `sensitive-data-tokenization` 설계 참고).
package processor

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"time"

	"github.com/cardea-ai/cardea/internal/plugin"
	"github.com/cardea-ai/cardea/internal/vault"
)

// DefaultTTL은 토큰 금고 항목의 기본 수명이다. 세션 종료 이벤트가 아직
// 없는 골격 단계의 자리값이고, FR-MEM 세션 개념이 들어오면 세션 종료에
// 맞춰 즉시 만료시키는 쪽으로 교체해야 한다.
const DefaultTTL = 10 * time.Minute

// tokenPattern은 Restore가 복원 대상 토큰을 찾는 정규식이다.
var tokenPattern = regexp.MustCompile(`⟦([A-Z0-9_]+)_([0-9A-Za-z]{8})⟧`)

// TokenProcessor는 plugin.Processor를 구현한다.
type TokenProcessor struct {
	Vault vault.Vault
}

var _ plugin.Processor = (*TokenProcessor)(nil)

// Process는 decision이 partial일 때만 마스킹을 적용한다. allow는 findings가
// 없다는 전제이므로 원문을 그대로 돌려주고, deny는 호출하면 안 되는
// 경로다(게이트웨이가 Forward 이전에 이미 거부했어야 한다) — 방어적으로
// 에러를 반환한다.
func (p *TokenProcessor) Process(ctx context.Context, decision plugin.Decision, findings []plugin.Finding, text string) (string, error) {
	switch decision {
	case plugin.DecisionAllow:
		return text, nil
	case plugin.DecisionDeny:
		return "", fmt.Errorf("processor: deny 판정에서는 호출되면 안 됨(게이트웨이 순서 버그)")
	case plugin.DecisionPartial:
		return p.Mask(ctx, scopeFromContext(ctx), findings, text)
	default:
		return "", fmt.Errorf("processor: 알 수 없는 decision %q", decision)
	}
}

type scopeKeyType struct{}

var scopeKey = scopeKeyType{}

// WithScope는 요청 스코프(세션/요청 단위 식별자)를 컨텍스트에 심는다.
// 같은 스코프 안에서만 토큰이 같은 원문으로 복원된다.
func WithScope(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, scopeKey, scope)
}

func scopeFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(scopeKey).(string); ok && v != "" {
		return v
	}
	return "unscoped"
}

// Mask는 findings를 Start 역순으로 치환해(인덱스가 뒤에서부터 바뀌어야
// 앞쪽 인덱스가 안 깨짐) 토큰으로 바꾼다. 같은 원문 문자열이 한 호출 안에
// 여러 번 나오면 같은 토큰을 재사용한다(참조 일관성).
func (p *TokenProcessor) Mask(ctx context.Context, scope string, findings []plugin.Finding, text string) (string, error) {
	sorted := make([]plugin.Finding, len(findings))
	copy(sorted, findings)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start > sorted[j].Start })

	issued := make(map[string]string) // 원문 → 토큰, 이번 호출 내 참조 일관성용
	out := text
	for _, f := range sorted {
		if f.Start < 0 || f.End > len(out) || f.Start >= f.End {
			continue // 정규화 단계에서 인덱스가 틀어진 경우 보수적으로 건너뛴다
		}
		plaintext := out[f.Start:f.End]
		token, ok := issued[plaintext]
		if !ok {
			var err error
			token, err = p.Vault.Put(ctx, scope, f.EntityType, "checksum-recognizer", "v0", plaintext, DefaultTTL)
			if err != nil {
				return "", fmt.Errorf("processor: 토큰 발급 실패: %w", err)
			}
			issued[plaintext] = token
		}
		out = out[:f.Start] + token + out[f.End:]
	}
	return out, nil
}

// Restore는 응답 본문에 섞인 토큰을 원문으로 되돌린다. 만료되거나 없는
// 토큰은 에러로 막지 않고 불투명한 토큰 문자열 그대로 둔다 — fail-closed
// (원문도 아니고 에러로 응답 전체를 막지도 않는다. 토큰 금고 설계 문서 참고).
func (p *TokenProcessor) Restore(ctx context.Context, scope string, text string) string {
	return tokenPattern.ReplaceAllStringFunc(text, func(tok string) string {
		plaintext, err := p.Vault.Resolve(ctx, scope, tok)
		if err != nil {
			return tok // 복원 불가 — 토큰 그대로 둔다, 원문을 꾸며내지 않는다
		}
		return plaintext
	})
}
