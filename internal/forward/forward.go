// Package forward는 ADR-0001의 Forwarder 인터페이스를 정의한다.
//
// Cardea 쪽 코드는 Bifrost core(또는 어떤 전달 엔진이든) 내부 구조에 직접
// 의존하지 않는다 — 이 인터페이스 뒤에서만 접근한다. 모든 판정(분류·마스킹·
// 정책·메모리·감사)은 Forward 호출 전에 끝나 있어야 한다.
//
// 이 패키지는 아직 Bifrost core를 임베드하지 않는다 — "최소 골격" 단계라
// 실제 모델 호출 없이 StubForwarder로 그 자리를 비워 둔다. Bifrost 임베딩은
// 별도 작업으로 TODO.md에 남겨 둔다(가짜로 호출에 성공한 것처럼 꾸미지
// 않는다).
package forward

import (
	"context"
	"errors"
)

// Request는 코어 표준 모델로 정규화된 요청이다(ADR-0006 SDK 어댑터 계층의
// 입력). 최소 골격에서는 더 자세한 필드를 아직 정의하지 않는다.
type Request struct {
	Protocol string // "openai" | "anthropic"
	Model    string
	Body     []byte // 마스킹 처리를 이미 거친 본문
}

// Response는 Forward 결과다.
type Response struct {
	StatusCode int
	Body       []byte
}

// Forwarder는 판정이 끝난 요청을 실제 모델 프로바이더로 보낸다.
type Forwarder interface {
	Forward(ctx context.Context, req Request) (Response, error)
}

// ErrNotImplemented는 Bifrost core 임베딩 전까지 StubForwarder가 반환하는
// 에러다. 호출부는 이걸 받으면 502로 응답해야 한다 — 임의로 더미 응답을
// 만들어 내지 않는다.
var ErrNotImplemented = errors.New("forward: Bifrost core 임베딩 전 — 전달 엔진 미구현(ADR-0001)")

// StubForwarder는 Bifrost core를 아직 임베드하지 않은 상태의 자리표시자다.
type StubForwarder struct{}

var _ Forwarder = StubForwarder{}

func (StubForwarder) Forward(_ context.Context, _ Request) (Response, error) {
	return Response{}, ErrNotImplemented
}
