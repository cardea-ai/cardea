// Package forward가 ADR-0001의 Forwarder 인터페이스를 Bifrost core
// (github.com/maximhq/bifrost/core, Apache-2.0)로 구현한다.
//
// 범위는 2026-10-25에 재확인한 ADR-0007/8장 결론을 따른다: Bifrost 공식
// 자료(README "Key Features" vs "Enterprise Deployments")가 MCP gateway를
// OSS와 엔터프라이즈로 서로 다르게 서술해 경계가 불명확하므로, 이 구현은
// Bifrost를 ADR-0001 원래 범위 — LLM 프로바이더 라우팅·폴백·스트리밍 —
// 로만 쓴다. MCP 1차 인증/allowlist는 Cardea 코어가 별도로 구현한다
// (여기서는 다루지 않는다).
package forward

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	bifrost "github.com/maximhq/bifrost/core"
	"github.com/maximhq/bifrost/core/schemas"
)

// envAccount는 schemas.Account를 환경변수 기반으로 구현한다. 1차 골격 단계의
// 자리값이다 — 실제 운영에서는 FR-RB(RBAC)·AG-01(에이전트 신원)과 연동되는
// 키 관리로 교체해야 한다. 키를 코드에 하드코딩하지 않고, 환경변수 이름도
// "CARDEA_" 접두어로만 읽어 일반 OPENAI_API_KEY 등을 실수로 집어가지 않는다.
type envAccount struct {
	keys map[schemas.ModelProvider]string // provider -> API key
}

var _ schemas.Account = (*envAccount)(nil)

// providerEnvVars는 지원하는 프로바이더와 그 API 키를 읽을 환경변수 이름이다.
var providerEnvVars = map[schemas.ModelProvider]string{
	schemas.OpenAI: "CARDEA_OPENAI_API_KEY",
}

func newEnvAccount() *envAccount {
	keys := make(map[schemas.ModelProvider]string)
	for provider, envVar := range providerEnvVars {
		if v := os.Getenv(envVar); v != "" {
			keys[provider] = v
		}
	}
	return &envAccount{keys: keys}
}

func (a *envAccount) GetConfiguredProviders() ([]schemas.ModelProvider, error) {
	providers := make([]schemas.ModelProvider, 0, len(a.keys))
	for p := range a.keys {
		providers = append(providers, p)
	}
	return providers, nil
}

func (a *envAccount) GetKeysForProvider(_ context.Context, providerKey schemas.ModelProvider) ([]schemas.Key, error) {
	v, ok := a.keys[providerKey]
	if !ok {
		return nil, fmt.Errorf("forward: provider %q에 대한 키가 설정되지 않음(%s 환경변수 필요)",
			providerKey, providerEnvVars[providerKey])
	}
	return []schemas.Key{{
		ID:     "env-key",
		Name:   "env-key",
		Value:  schemas.SecretVar{Val: v},
		Weight: 1,
	}}, nil
}

func (a *envAccount) GetConfigForProvider(_ schemas.ModelProvider) (*schemas.ProviderConfig, error) {
	return &schemas.ProviderConfig{}, nil
}

// BifrostForwarder는 Forwarder를 Bifrost core Go SDK로 구현한다.
type BifrostForwarder struct {
	bf *bifrost.Bifrost
}

var _ Forwarder = (*BifrostForwarder)(nil)

// NewBifrostForwarder는 Bifrost를 초기화한다. 프로바이더 키가 하나도
// 설정돼 있지 않아도 초기화 자체는 성공한다 — 실제 요청이 들어올 때
// "이 프로바이더에 키가 없다"는 에러를 그 요청에서만 돌려준다(NFR-AV-02
// fail-closed와 충돌하지 않는다: 키가 없다는 건 "거부"가 아니라 "설정
// 누락" 에러다).
func NewBifrostForwarder(ctx context.Context) (*BifrostForwarder, error) {
	bf, err := bifrost.Init(ctx, schemas.BifrostConfig{
		Account: newEnvAccount(),
		Logger:  bifrost.NewNoOpLogger(), // Cardea 자체 로깅을 쓴다 — 원문이 Bifrost 로그로 새지 않게
	})
	if err != nil {
		return nil, fmt.Errorf("forward: Bifrost 초기화 실패: %w", err)
	}
	return &BifrostForwarder{bf: bf}, nil
}

// chatRequestBody는 Forward가 기대하는 최소 입력 포맷이다. 실제 OpenAI/
// Anthropic 프로토콜 변환(ADR-0006 SDK 어댑터 계층)은 아직 이 골격에 없고,
// 당장은 "provider/model + 평문 메시지 목록"만 받는다.
type chatRequestBody struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
}

func (f *BifrostForwarder) Forward(ctx context.Context, req Request) (Response, error) {
	var body chatRequestBody
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return Response{}, fmt.Errorf("forward: 요청 본문 파싱 실패: %w", err)
	}
	if body.Provider == "" {
		body.Provider = "openai" // 1차 골격 기본값
	}

	messages := make([]schemas.ChatMessage, 0, len(body.Messages))
	for _, m := range body.Messages {
		content := m.Content
		messages = append(messages, schemas.ChatMessage{
			Role:    schemas.ChatMessageRole(m.Role),
			Content: &schemas.ChatMessageContent{ContentStr: &content},
		})
	}

	bifrostCtx, cancel := schemas.NewBifrostContextWithCancel(ctx)
	defer cancel()

	resp, bifrostErr := f.bf.ChatCompletionRequest(bifrostCtx, &schemas.BifrostChatRequest{
		Provider: schemas.ModelProvider(strings.ToLower(body.Provider)),
		Model:    body.Model,
		Input:    messages,
	})
	if bifrostErr != nil {
		status := bifrostErr.EffectiveHTTPStatus()
		if status == 0 {
			status = 502
		}
		return Response{StatusCode: status}, fmt.Errorf("forward: Bifrost 오류: %s", bifrostErr.GetErrorString())
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return Response{}, fmt.Errorf("forward: 응답 직렬화 실패: %w", err)
	}
	return Response{StatusCode: 200, Body: out}, nil
}
