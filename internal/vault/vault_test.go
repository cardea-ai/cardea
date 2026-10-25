package vault

import (
	"context"
	"testing"
	"time"
)

func TestFR_PI_04_PutResolveRoundTrip(t *testing.T) {
	v, err := NewMemoryVault()
	if err != nil {
		t.Fatalf("NewMemoryVault error: %v", err)
	}
	ctx := context.Background()

	token, err := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}

	got, err := v.Resolve(ctx, "scope-a", token)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "901231-1234563" {
		t.Errorf("Resolve = %q, want original plaintext", got)
	}
}

func TestFR_PI_04_TokenFormatHasNoSequentialSuffix(t *testing.T) {
	v, _ := NewMemoryVault()
	ctx := context.Background()

	t1, _ := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)
	t2, _ := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "880515-2345671", time.Minute)
	if t1 == t2 {
		t.Fatalf("두 다른 원문이 같은 토큰을 받음: %q", t1)
	}
	// 토큰 접미사가 "_1", "_2" 같은 순번이면 안 된다 (메타데이터 유출).
	if len(t1) < 10 || len(t2) < 10 {
		t.Fatalf("토큰 형식이 예상보다 짧음: %q %q", t1, t2)
	}
}

func TestFR_PI_04_ScopeIsolation(t *testing.T) {
	v, _ := NewMemoryVault()
	ctx := context.Background()

	token, _ := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)

	if _, err := v.Resolve(ctx, "scope-b", token); err != ErrExpiredOrMissing {
		t.Errorf("다른 스코프에서 토큰이 풀림 — scope isolation 깨짐 (err=%v)", err)
	}
}

func TestFR_PI_04_ExpiredTokenFailsClosed(t *testing.T) {
	v, _ := NewMemoryVault()
	ctx := context.Background()

	token, _ := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", -time.Second)

	if _, err := v.Resolve(ctx, "scope-a", token); err != ErrExpiredOrMissing {
		t.Errorf("만료된 토큰이 풀림 (err=%v)", err)
	}
}
