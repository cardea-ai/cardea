package presidio

import (
	"context"
	"os"
	"testing"
)

// 실제 Presidio Analyzer 서버가 있어야 도는 통합 테스트다. 로컬 재현:
// docker run --rm -p 8301:3000 mcr.microsoft.com/presidio-analyzer:latest
func testClient(t *testing.T) *Client {
	addr := os.Getenv("CARDEA_PRESIDIO_TEST_ADDR")
	if addr == "" {
		t.Skip("CARDEA_PRESIDIO_TEST_ADDR 미설정 — Presidio 통합 테스트 건너뜀")
	}
	return NewClient(addr, "en")
}

func TestFR_PI_02_PresidioDetectsPersonAndEmail(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	if err := c.Healthy(ctx); err != nil {
		t.Fatalf("Healthy() error: %v", err)
	}

	findings, err := c.Detect(ctx, "My name is John Smith and my email is john@example.com")
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}

	var gotPerson, gotEmail bool
	for _, f := range findings {
		switch f.EntityType {
		case "PERSON":
			gotPerson = true
		case "EMAIL_ADDRESS":
			gotEmail = true
		}
	}
	if !gotPerson {
		t.Errorf("PERSON을 탐지하지 못함: %+v", findings)
	}
	if !gotEmail {
		t.Errorf("EMAIL_ADDRESS를 탐지하지 못함: %+v", findings)
	}
}

func TestFR_PI_02_PresidioFindingIndicesMatchOriginalText(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()

	text := "My name is John Smith and my email is john@example.com"
	findings, err := c.Detect(ctx, text)
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	for _, f := range findings {
		if text[f.Start:f.End] != f.Text {
			t.Errorf("인덱스가 원문과 안 맞음: finding=%+v, text[%d:%d]=%q",
				f, f.Start, f.End, text[f.Start:f.End])
		}
	}
}

// 한국어는 공식 Docker 이미지 기본 NLP 엔진에 없다는 걸 확인한다 — 코드가
// 한국어를 지원한다고 거짓으로 주장하지 않기 위한 네거티브 테스트다.
func TestFR_PI_02_PresidioKoreanNotSupportedByDefaultImage(t *testing.T) {
	addr := os.Getenv("CARDEA_PRESIDIO_TEST_ADDR")
	if addr == "" {
		t.Skip("CARDEA_PRESIDIO_TEST_ADDR 미설정 — Presidio 통합 테스트 건너뜀")
	}
	c := NewClient(addr, "ko")
	ctx := context.Background()

	_, err := c.Detect(ctx, "제 이름은 홍길동입니다")
	if err == nil {
		t.Error("기본 이미지가 한국어를 지원하는 것처럼 성공함 — 실제로는 거부돼야 함(no matching recognizers)")
	}
}
