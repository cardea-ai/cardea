package forward

import (
	"context"
	"os"
	"strings"
	"testing"
)

// Bifrost core Go SDK가 실제로 임베드돼 있는지 확인한다. 이 환경에는
// 프로바이더 API 키가 없으므로(CLAUDE.md: 비밀값을 코드/테스트에 넣지
// 않는다) 실제 모델 호출까지는 검증하지 못한다 — 대신 "키가 없는
// 프로바이더로 요청하면 Bifrost가 설정 누락을 구조화된 에러로 돌려준다"는
// 경로를 검증한다. 이건 StubForwarder의 "미구현" 에러와는 다른 종류의
// 실패라는 걸 구분해서 본다.
func TestFR_GW_BifrostForwarderInitializesRealSDK(t *testing.T) {
	ctx := context.Background()
	f, err := NewBifrostForwarder(ctx)
	if err != nil {
		t.Fatalf("NewBifrostForwarder error: %v", err)
	}
	if f.bf == nil {
		t.Fatal("내부 *bifrost.Bifrost 인스턴스가 nil")
	}
}

func TestFR_GW_BifrostForwarderReturnsConfigErrorWithoutAPIKey(t *testing.T) {
	// 이 테스트가 실수로 실제 키로 도는 걸 막는다 — 키가 있으면 스킵한다
	// (진짜 네트워크 호출은 별도의, 명시적으로 옵트인하는 통합 테스트로
	// 분리해야 한다).
	if os.Getenv("CARDEA_OPENAI_API_KEY") != "" {
		t.Skip("CARDEA_OPENAI_API_KEY가 설정돼 있음 — 이 테스트는 키 없는 경로만 검증함")
	}

	ctx := context.Background()
	f, err := NewBifrostForwarder(ctx)
	if err != nil {
		t.Fatalf("NewBifrostForwarder error: %v", err)
	}

	_, err = f.Forward(ctx, Request{
		Protocol: "openai",
		Body:     []byte(`{"provider":"openai","model":"gpt-4o-mini","messages":[{"role":"user","content":"hi"}]}`),
	})
	if err == nil {
		t.Fatal("키 없이도 Forward가 성공함 — 실제로는 설정 누락 에러가 와야 함")
	}
	if !strings.Contains(err.Error(), "forward: Bifrost 오류") {
		t.Fatalf("예상한 Bifrost 오류 포맷이 아님: %v", err)
	}
}
