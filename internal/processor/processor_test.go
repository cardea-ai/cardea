package processor

import (
	"context"
	"strings"
	"testing"

	"github.com/cardea-ai/cardea/internal/plugin"
	"github.com/cardea-ai/cardea/internal/vault"
)

func newTestProcessor(t *testing.T) *TokenProcessor {
	t.Helper()
	v, err := vault.NewMemoryVault()
	if err != nil {
		t.Fatalf("NewMemoryVault error: %v", err)
	}
	return &TokenProcessor{Vault: v}
}

func TestFR_PI_04_MaskReplacesFindingWithOpaqueToken(t *testing.T) {
	p := newTestProcessor(t)
	ctx := WithScope(context.Background(), "test-scope")

	text := "주민번호 901231-1234563 확인"
	findings := []plugin.Finding{{EntityType: "KR_RRN", Start: 5, End: 19, Text: "901231-1234563"}}

	masked, err := p.Mask(ctx, "test-scope", findings, text)
	if err != nil {
		t.Fatalf("Mask error: %v", err)
	}
	if strings.Contains(masked, "901231-1234563") {
		t.Fatalf("마스킹된 본문에 원문이 그대로 남아 있음: %q", masked)
	}
	if !strings.Contains(masked, "⟦KR_RRN_") {
		t.Fatalf("마스킹된 본문에 기대한 토큰 접두사가 없음: %q", masked)
	}
}

func TestFR_PI_04_MaskThenRestoreRoundTrip(t *testing.T) {
	p := newTestProcessor(t)
	scope := "test-scope"
	ctx := WithScope(context.Background(), scope)

	text := "주민번호 901231-1234563 확인"
	findings := []plugin.Finding{{EntityType: "KR_RRN", Start: 5, End: 19, Text: "901231-1234563"}}

	masked, err := p.Mask(ctx, scope, findings, text)
	if err != nil {
		t.Fatalf("Mask error: %v", err)
	}

	restored := p.Restore(ctx, scope, masked)
	if restored != text {
		t.Errorf("Restore = %q, want %q", restored, text)
	}
}

func TestFR_PI_04_RestoreLeavesUnknownTokenOpaque(t *testing.T) {
	p := newTestProcessor(t)
	ctx := WithScope(context.Background(), "test-scope")

	restored := p.Restore(ctx, "test-scope", "값: ⟦KR_RRN_ZZZZZZZZ⟧")
	if restored != "값: ⟦KR_RRN_ZZZZZZZZ⟧" {
		t.Errorf("알 수 없는 토큰이 원문처럼 복원됨: %q", restored)
	}
}

func TestFR_PI_04_ReferentialConsistencyWithinOneCall(t *testing.T) {
	p := newTestProcessor(t)
	ctx := WithScope(context.Background(), "test-scope")

	// 같은 원문이 한 번의 Mask 호출 안에서 두 번 나오면 같은 토큰을 받아야
	// 한다(모델이 "같은 개체"라는 걸 추적할 수 있어야 함).
	text := "901231-1234563 ... 901231-1234563"
	findings := []plugin.Finding{
		{EntityType: "KR_RRN", Start: 0, End: 14},
		{EntityType: "KR_RRN", Start: 19, End: 33},
	}
	masked, err := p.Mask(ctx, "test-scope", findings, text)
	if err != nil {
		t.Fatalf("Mask error: %v", err)
	}
	parts := strings.Split(masked, " ... ")
	if len(parts) != 2 || parts[0] != parts[1] {
		t.Errorf("같은 원문이 다른 토큰을 받음: %q", masked)
	}
}

func TestFR_PI_04_ProcessRejectsDenyDecision(t *testing.T) {
	p := newTestProcessor(t)
	ctx := WithScope(context.Background(), "test-scope")

	if _, err := p.Process(ctx, plugin.DecisionDeny, nil, "x"); err == nil {
		t.Error("deny 판정에서 Process가 에러 없이 통과함")
	}
}
