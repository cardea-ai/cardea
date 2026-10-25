// OpenBaoVault는 ADR-0007이 선택한 토큰 금고 크립토 백엔드(OpenBao Transit
// 엔진)로 구현한 Vault다. MemoryVault와 똑같은 인터페이스를 만족하므로
// `internal/gateway`는 둘을 자유롭게 바꿔 끼울 수 있다 — "암호화 백엔드를
// Cardea 자체 매핑/스코프 로직과 분리해 둔다"는 ADR-0007 완화책을 그대로
// 지킨다: 실제 암호화 키는 OpenBao 안에만 있고, Cardea 프로세스는 매핑
// 메타데이터(scope, type, 만료 시각)와 OpenBao가 돌려준 ciphertext 문자열만
// 들고 있는다 — 키 자체는 절대 Cardea 프로세스 메모리로 내려오지 않는다.
package vault

import (
	"context"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	openbao "github.com/openbao/openbao/api/v2"
)

// OpenBaoEntry는 OpenBaoVault가 로컬에 들고 있는 메타데이터다. 암호화된
// 원문(OpenBao Transit이 돌려준 ciphertext)만 저장하고, 복호화 키는 절대
// 여기 없다 — 매 Resolve마다 OpenBao에 다시 물어본다.
type openBaoEntry struct {
	Ciphertext  string
	Type        string
	RuleID      string
	RuleVersion string
	Scope       string
	CreatedAt   time.Time
	ExpiresAt   time.Time
}

// OpenBaoVault는 Vault를 구현한다.
type OpenBaoVault struct {
	client  *openbao.Client
	keyName string // Transit 암호화 키 이름

	mu      sync.Mutex
	entries map[string]openBaoEntry // key = scope + "\x00" + token
}

var _ Vault = (*OpenBaoVault)(nil)

// OpenBaoConfig는 연결 정보다. Token은 환경변수(BAO_TOKEN)로 넘기는 걸
// 권장한다 — 코드에 하드코딩하지 않는다.
type OpenBaoConfig struct {
	Address string // 예: "http://127.0.0.1:8300"
	Token   string
	KeyName string // Transit 키 이름, 예: "cardea-vault-key"
}

// NewOpenBaoVault는 OpenBao에 연결하고, Transit 키가 없으면 새로 만든다.
// 연결 실패나 키 생성 실패는 즉시 에러로 반환한다 — fail-closed로 기동을
// 막는다(NFR-AV-02).
func NewOpenBaoVault(ctx context.Context, cfg OpenBaoConfig) (*OpenBaoVault, error) {
	config := openbao.DefaultConfig()
	config.Address = cfg.Address
	client, err := openbao.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("openbao: 클라이언트 생성 실패: %w", err)
	}
	client.SetToken(cfg.Token)

	v := &OpenBaoVault{client: client, keyName: cfg.KeyName, entries: make(map[string]openBaoEntry)}

	// Transit 엔진이 마운트돼 있고 키가 존재하는지 확인 — 없으면 생성.
	// 둘 다 실패하면 (이미 있어서 실패한 경우를 포함해) 최종적으로 encrypt
	// 호출이 성공하는지로 실제 가용성을 재확인한다.
	_, _ = client.Logical().Write("sys/mounts/transit", map[string]any{"type": "transit"})
	if _, err := client.Logical().Write("transit/keys/"+cfg.KeyName, nil); err != nil {
		// 이미 존재하는 키라면 Write가 에러 없이 멱등하게 처리되는 버전도
		// 있고 아닌 버전도 있어, 여기서는 치명적으로 보지 않는다 — 실제
		// encrypt 호출로 아래에서 다시 검증한다.
		_ = err
	}
	if _, err := v.transitEncrypt(ctx, "liveness-check"); err != nil {
		return nil, fmt.Errorf("openbao: Transit 키 %q 가용성 확인 실패(fail-closed): %w", cfg.KeyName, err)
	}
	return v, nil
}

func (v *OpenBaoVault) transitEncrypt(ctx context.Context, plaintext string) (string, error) {
	secret, err := v.client.Logical().WriteWithContext(ctx, "transit/encrypt/"+v.keyName, map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString([]byte(plaintext)),
	})
	if err != nil {
		return "", err
	}
	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("openbao: encrypt 응답에 ciphertext 없음")
	}
	return ciphertext, nil
}

func (v *OpenBaoVault) transitDecrypt(ctx context.Context, ciphertext string) (string, error) {
	secret, err := v.client.Logical().WriteWithContext(ctx, "transit/decrypt/"+v.keyName, map[string]any{
		"ciphertext": ciphertext,
	})
	if err != nil {
		return "", err
	}
	b64, ok := secret.Data["plaintext"].(string)
	if !ok {
		return "", fmt.Errorf("openbao: decrypt 응답에 plaintext 없음")
	}
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", fmt.Errorf("openbao: plaintext base64 디코딩 실패: %w", err)
	}
	return string(raw), nil
}

func (v *OpenBaoVault) key(scope, token string) string { return scope + "\x00" + token }

func (v *OpenBaoVault) Put(ctx context.Context, scope, entityType, ruleID, ruleVersion, plaintext string, ttl time.Duration) (string, error) {
	suffix, err := newSuffix()
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("⟦%s_%s⟧", entityType, suffix)

	ciphertext, err := v.transitEncrypt(ctx, plaintext)
	if err != nil {
		return "", fmt.Errorf("openbao: 암호화 실패(fail-closed): %w", err)
	}

	now := time.Now()
	v.mu.Lock()
	v.entries[v.key(scope, token)] = openBaoEntry{
		Ciphertext: ciphertext, Type: entityType, RuleID: ruleID, RuleVersion: ruleVersion,
		Scope: scope, CreatedAt: now, ExpiresAt: now.Add(ttl),
	}
	v.mu.Unlock()
	return token, nil
}

func (v *OpenBaoVault) Resolve(ctx context.Context, scope, token string) (string, error) {
	v.mu.Lock()
	entry, ok := v.entries[v.key(scope, token)]
	v.mu.Unlock()
	if !ok {
		return "", ErrExpiredOrMissing
	}
	if time.Now().After(entry.ExpiresAt) {
		v.mu.Lock()
		delete(v.entries, v.key(scope, token))
		v.mu.Unlock()
		return "", ErrExpiredOrMissing
	}
	plaintext, err := v.transitDecrypt(ctx, entry.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("openbao: 복호화 실패: %w", err)
	}
	return plaintext, nil
}
