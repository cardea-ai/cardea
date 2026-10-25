package detect

import (
	"context"
	"testing"
)

func TestFR_PI_01_ResidentNumberChecksum(t *testing.T) {
	cases := []struct {
		name  string
		digit string
		want  bool
	}{
		{"valid", "9012311234563", true},
		{"bad_checksum", "9012311234564", false},
		{"too_short", "901231123456", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidateRRN(c.digit); got != c.want {
				t.Errorf("ValidateRRN(%q) = %v, want %v", c.digit, got, c.want)
			}
		})
	}
}

func TestFR_PI_08_CardNumberLuhn(t *testing.T) {
	cases := []struct {
		name  string
		digit string
		want  bool
	}{
		{"visa_valid", "4111111111111111", true},
		{"mastercard_valid", "5500005555555559", true},
		{"amex_valid", "340000000000009", true},
		{"invalid", "1234567812345678", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ValidateLuhn(c.digit); got != c.want {
				t.Errorf("ValidateLuhn(%q) = %v, want %v", c.digit, got, c.want)
			}
		})
	}
}

func TestFR_PI_11_NormalizeHangulDigitsAndFullwidth(t *testing.T) {
	n := Normalizer{}
	got, err := n.Normalize(context.Background(), "구공일이삼일-1234563")
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	want := "901231-1234563"
	if got != want {
		t.Errorf("Normalize hangul digits = %q, want %q", got, want)
	}

	got, err = n.Normalize(context.Background(), "９０１２３１－１２３４５６３")
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if got != want {
		t.Errorf("Normalize fullwidth = %q, want %q", got, want)
	}
}

func TestFR_PI_01_ChecksumRecognizerDetectsValidRRNOnly(t *testing.T) {
	r := ChecksumRecognizer{}
	n := Normalizer{}
	ctx := context.Background()

	normalized, _ := n.Normalize(ctx, "주민등록번호는 901231-1234563 입니다.")
	findings, err := r.Detect(ctx, normalized)
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if len(findings) != 1 || findings[0].EntityType != "KR_RRN" {
		t.Fatalf("expected 1 KR_RRN finding, got %+v", findings)
	}

	// 체크섬이 틀린 경우에는 탐지되지 않아야 한다 (거짓양성 방지).
	normalized, _ = n.Normalize(ctx, "임시 코드 901231-1234564 는 유효하지 않습니다")
	findings, err = r.Detect(ctx, normalized)
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings for invalid checksum, got %+v", findings)
	}
}

func TestFR_PI_08_ChecksumRecognizerDetectsAmexGrouping(t *testing.T) {
	r := ChecksumRecognizer{}
	ctx := context.Background()
	findings, err := r.Detect(ctx, "아멕스 카드 3400-000000-00009 사용 거절")
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if len(findings) != 1 || findings[0].EntityType != "KR_CARD_NUMBER" {
		t.Fatalf("expected 1 KR_CARD_NUMBER finding for Amex grouping, got %+v", findings)
	}
}
