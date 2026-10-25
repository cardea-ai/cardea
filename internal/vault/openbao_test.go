package vault

import (
	"context"
	"os"
	"testing"
	"time"
)

// 실제 OpenBao 서버가 있어야 도는 통합 테스트다. CI/로컬에 OpenBao가 없으면
// 건너뛴다 — fail 대신 skip으로 처리해 "OpenBao 없는 환경"과 "OpenBao
// 연동이 실제로 깨진 상황"을 구분한다.
//
// 로컬 재현: docker run --rm -p 8300:8200 \
//   -e BAO_DEV_ROOT_TOKEN_ID=cardea-dev-root \
//   -e BAO_DEV_LISTEN_ADDRESS=0.0.0.0:8200 openbao/openbao:latest server -dev
func testOpenBaoConfig(t *testing.T) OpenBaoConfig {
	addr := os.Getenv("CARDEA_OPENBAO_TEST_ADDR")
	if addr == "" {
		t.Skip("CARDEA_OPENBAO_TEST_ADDR 미설정 — OpenBao 통합 테스트 건너뜀")
	}
	return OpenBaoConfig{
		Address: addr,
		Token:   os.Getenv("CARDEA_OPENBAO_TEST_TOKEN"),
		KeyName: "cardea-vault-key-test",
	}
}

func TestFR_PI_04_OpenBaoPutResolveRoundTrip(t *testing.T) {
	cfg := testOpenBaoConfig(t)
	ctx := context.Background()

	v, err := NewOpenBaoVault(ctx, cfg)
	if err != nil {
		t.Fatalf("NewOpenBaoVault error: %v", err)
	}

	token, err := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if token == "" {
		t.Fatal("Put이 빈 토큰을 반환함")
	}

	got, err := v.Resolve(ctx, "scope-a", token)
	if err != nil {
		t.Fatalf("Resolve error: %v", err)
	}
	if got != "901231-1234563" {
		t.Errorf("Resolve = %q, want original plaintext", got)
	}
}

func TestFR_PI_04_OpenBaoScopeIsolation(t *testing.T) {
	cfg := testOpenBaoConfig(t)
	ctx := context.Background()

	v, err := NewOpenBaoVault(ctx, cfg)
	if err != nil {
		t.Fatalf("NewOpenBaoVault error: %v", err)
	}

	token, err := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if _, err := v.Resolve(ctx, "scope-b", token); err != ErrExpiredOrMissing {
		t.Errorf("다른 스코프에서 토큰이 풀림 (err=%v)", err)
	}
}

func TestFR_PI_04_OpenBaoExpiredTokenFailsClosed(t *testing.T) {
	cfg := testOpenBaoConfig(t)
	ctx := context.Background()

	v, err := NewOpenBaoVault(ctx, cfg)
	if err != nil {
		t.Fatalf("NewOpenBaoVault error: %v", err)
	}

	token, err := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", -time.Second)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}
	if _, err := v.Resolve(ctx, "scope-a", token); err != ErrExpiredOrMissing {
		t.Errorf("만료된 토큰이 풀림 (err=%v)", err)
	}
}

// 실제 암호화 키가 Cardea 프로세스로 안 내려온다는 걸 확인한다 — 로컬에
// 남는 건 OpenBao가 돌려준 ciphertext 문자열뿐이다(ADR-0007 완화책).
func TestFR_PI_04_OpenBaoLocalEntryHoldsOnlyCiphertextNotKey(t *testing.T) {
	cfg := testOpenBaoConfig(t)
	ctx := context.Background()

	v, err := NewOpenBaoVault(ctx, cfg)
	if err != nil {
		t.Fatalf("NewOpenBaoVault error: %v", err)
	}
	token, err := v.Put(ctx, "scope-a", "KR_RRN", "rule-1", "v0", "901231-1234563", time.Minute)
	if err != nil {
		t.Fatalf("Put error: %v", err)
	}

	v.mu.Lock()
	entry, ok := v.entries[v.key("scope-a", token)]
	v.mu.Unlock()
	if !ok {
		t.Fatal("로컬 엔트리를 찾을 수 없음")
	}
	if entry.Ciphertext == "901231-1234563" {
		t.Fatal("로컬에 평문이 그대로 저장됨 — 암호화가 안 된 것")
	}
	if len(entry.Ciphertext) < 10 {
		t.Fatalf("ciphertext가 너무 짧음(실제 OpenBao Transit 포맷이 아닌 듯): %q", entry.Ciphertext)
	}
}
