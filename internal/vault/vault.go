// Package vault는 가역 마스킹 토큰의 원문-토큰 매핑을 보관하는 암호화 저장소다
// (CLAUDE.md 4장 "가역 마스킹 토큰은 난수+금고 방식만 쓴다").
//
// 이 구현은 ADR-0007이 선택한 OpenBao를 아직 연동하지 않은 1차 골격이다 —
// 프로세스 메모리 안에서 AES-256-GCM으로 암호화해 들고 있는다
// (`MemoryVault`). OpenBao 클라이언트로 교체할 때 이 인터페이스(Vault)만
// 구현하면 되도록 분리해 뒀다. 절대 하지 않는 것: 원문-토큰 매핑을
// 결정론적 해시로 만들지 않는다(좁은 포맷 값은 해시만으로도 역산 가능).
package vault

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// Entry는 토큰 금고의 한 행이다. 필드 구성은
// open-core-spec-authoring 스킬의 가역 마스킹 설계를 그대로 따른다.
type Entry struct {
	Token       string
	Type        string
	RuleID      string
	RuleVersion string
	Scope       string
	CreatedAt   time.Time
	ExpiresAt   time.Time

	ciphertext []byte
	nonce      []byte
}

// ErrExpiredOrMissing은 Resolve가 만료되거나 없는 토큰에 대해 반환한다.
// 호출부는 이 에러를 받으면 토큰을 불투명한 문자열로 그대로 둬야 한다
// (원문도 에러도 아닌 값을 꾸며내지 않는다) — fail-closed.
var ErrExpiredOrMissing = errors.New("vault: 토큰이 없거나 만료됨")

// Vault는 원문 <-> 토큰 매핑을 보관한다. OpenBao 클라이언트를 포함해 모든
// 구현체가 지켜야 할 계약: Put은 매 호출마다 새 CSPRNG 토큰을 발급하고,
// Resolve는 스코프가 다르면 절대 값을 돌려주지 않는다.
type Vault interface {
	// Put은 plaintext를 암호화해 저장하고 새 토큰을 발급한다.
	Put(ctx context.Context, scope, entityType, ruleID, ruleVersion, plaintext string, ttl time.Duration) (token string, err error)
	// Resolve는 scope가 일치하고 아직 만료되지 않은 토큰만 원문으로 되돌린다.
	Resolve(ctx context.Context, scope, token string) (plaintext string, err error)
}

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// newSuffix는 CSPRNG 기반 base62 8자 문자열을 만든다. 순번(RRN_1, RRN_2, ...)을
// 쓰지 않는 이유: 순번 자체가 "이 유형이 몇 개 나왔는지"라는 민감한 메타데이터를
// 흘린다.
func newSuffix() (string, error) {
	const n = 8
	b := make([]byte, n)
	max := big.NewInt(int64(len(base62Alphabet)))
	for i := 0; i < n; i++ {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", fmt.Errorf("vault: 토큰 접미사 생성 실패: %w", err)
		}
		b[i] = base62Alphabet[idx.Int64()]
	}
	return string(b), nil
}

// MemoryVault는 OpenBao 연동 전까지 쓰는 프로세스 메모리 구현체다.
// 프로세스가 재시작되면 모든 매핑이 사라진다 — 이건 결함이 아니라 의도다
// (영구 저장이 필요하면 그게 바로 OpenBao를 붙여야 하는 이유).
type MemoryVault struct {
	mu      sync.Mutex
	entries map[string]Entry // key = scope + "\x00" + token
	gcm     cipher.AEAD
}

var _ Vault = (*MemoryVault)(nil)

// NewMemoryVault는 프로세스 수명 동안만 쓰는 랜덤 AES-256 키로 GCM을
// 초기화한다.
func NewMemoryVault() (*MemoryVault, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("vault: 키 생성 실패: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("vault: cipher 생성 실패: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("vault: GCM 생성 실패: %w", err)
	}
	return &MemoryVault{entries: make(map[string]Entry), gcm: gcm}, nil
}

func (v *MemoryVault) key(scope, token string) string { return scope + "\x00" + token }

func (v *MemoryVault) Put(_ context.Context, scope, entityType, ruleID, ruleVersion, plaintext string, ttl time.Duration) (string, error) {
	suffix, err := newSuffix()
	if err != nil {
		return "", err
	}
	token := fmt.Sprintf("⟦%s_%s⟧", entityType, suffix)

	nonce := make([]byte, v.gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("vault: nonce 생성 실패: %w", err)
	}
	ciphertext := v.gcm.Seal(nil, nonce, []byte(plaintext), nil)

	now := time.Now()
	entry := Entry{
		Token:       token,
		Type:        entityType,
		RuleID:      ruleID,
		RuleVersion: ruleVersion,
		Scope:       scope,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
		ciphertext:  ciphertext,
		nonce:       nonce,
	}

	v.mu.Lock()
	v.entries[v.key(scope, token)] = entry
	v.mu.Unlock()
	return token, nil
}

func (v *MemoryVault) Resolve(_ context.Context, scope, token string) (string, error) {
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
	plaintext, err := v.gcm.Open(nil, entry.nonce, entry.ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("vault: 복호화 실패: %w", err)
	}
	return string(plaintext), nil
}
