// cardea 코어 엔트리포인트.
//
// ADR-0008: 가로채기 메커니즘은 표준 프로토콜(LLM API) 프록시가 기본값이다.
// 로컬 전용(127.0.0.1)으로 바인딩하고, 클라우드 비의존 원칙(NFR-DP-01)을 지킨다.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/cardea-ai/cardea/internal/gateway"
)

func main() {
	addr := os.Getenv("CARDEA_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8787"
	}

	srv, err := gateway.New()
	if err != nil {
		// NFR-AV-02: 판정 엔진이 뜨지 못하면 fail-closed — 트래픽을 받지 않고 종료한다.
		log.Fatalf("cardea: 게이트웨이 초기화 실패(fail-closed로 종료): %v", err)
	}

	log.Printf("cardea: 로컬 게이트웨이 시작 (%s)", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("cardea: 서버 종료: %v", err)
	}
}
