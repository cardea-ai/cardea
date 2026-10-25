// Package detect는 FR-PI-01/08(체크섬 있는 식별번호·카드번호 탐지)과
// FR-PI-11(정규화 선행)의 1차 Go 구현이다.
//
// 이 로직은 eval/nfr-dq/의 파이썬 스파이크에서 먼저 검증했다 — 합성
// 평가셋 46건에서 재현율 1.0000/정밀도 0.9714로 NFR-DQ-01 목표(99%/95%)를
// 충족했다(자세한 내용은 eval/nfr-dq/README.md). 여기서는 그 정규화·체크섬
// 로직을 Go로 포팅한다. Microsoft Presidio(ADR-0007)는 별도 프로세스로
// 붙일 "탐지기" 슬롯의 또 다른 백엔드이고, 이 Go 구현은 체크섬 기반
// 엔티티처럼 가벼운 경로를 위해 코어에 직접 둔다.
package detect

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"github.com/cardea-ai/cardea/internal/plugin"
)

var hangulDigit = map[rune]rune{
	'영': '0', '공': '0', '일': '1', '이': '2', '삼': '3', '사': '4',
	'오': '5', '육': '6', '칠': '7', '팔': '8', '구': '9',
}

// Normalizer는 FR-PI-11 정규화를 구현한다: 전각 숫자 반각화, 한글 숫자
// 치환, 구분자는 탐지기 쪽 정규식에서 허용하므로 여기서는 제거하지 않는다
// (원문 레이아웃을 과도하게 바꾸지 않기 위함).
type Normalizer struct{}

var _ plugin.Normalizer = Normalizer{}

func (Normalizer) Normalize(_ context.Context, text string) (string, error) {
	// 1) 전각 숫자(U+FF10~U+FF19) → 반각 숫자
	runes := []rune(text)
	for i, r := range runes {
		switch {
		case r >= 0xFF10 && r <= 0xFF19:
			runes[i] = '0' + (r - 0xFF10)
		case r == 0xFF0D: // 전각 하이픈-마이너스(－) → 반각 하이픈(-)
			runes[i] = '-'
		}
	}
	out := string(runes)

	// 2) 한글 숫자 단어 2글자 이상 연속 구간만 치환 (과교정 방지)
	var b strings.Builder
	runRunes := []rune(out)
	i := 0
	for i < len(runRunes) {
		j := i
		for j < len(runRunes) {
			if _, ok := hangulDigit[runRunes[j]]; !ok {
				break
			}
			j++
		}
		if j-i >= 2 {
			for k := i; k < j; k++ {
				b.WriteRune(hangulDigit[runRunes[k]])
			}
		} else {
			for k := i; k < j; k++ {
				b.WriteRune(runRunes[k])
			}
		}
		if j == i {
			b.WriteRune(runRunes[i])
			i++
		} else {
			i = j
		}
	}
	return b.String(), nil
}

func stripSeparators(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case unicode.IsDigit(r):
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_' || r == '·' || r == '.' || r == '*':
			// 구분자는 버린다
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// ValidateRRN은 주민등록번호(13자리) 체크섬을 검증한다.
func ValidateRRN(digits13 string) bool {
	if len(digits13) != 13 {
		return false
	}
	weights := []int{2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5}
	total := 0
	for i, w := range weights {
		if digits13[i] < '0' || digits13[i] > '9' {
			return false
		}
		total += int(digits13[i]-'0') * w
	}
	check := (11 - (total % 11)) % 10
	return digits13[12]-'0' == byte(check)
}

// ValidateLuhn은 카드번호 Luhn 체크섬을 검증한다.
func ValidateLuhn(digits string) bool {
	if len(digits) < 12 {
		return false
	}
	total := 0
	alt := false
	for i := len(digits) - 1; i >= 0; i-- {
		if digits[i] < '0' || digits[i] > '9' {
			return false
		}
		d := int(digits[i] - '0')
		if alt {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		total += d
		alt = !alt
	}
	return total%10 == 0
}

// ValidateBizRegNo는 사업자등록번호(10자리) 체크섬을 검증한다.
func ValidateBizRegNo(digits10 string) bool {
	if len(digits10) != 10 {
		return false
	}
	weights := []int{1, 3, 7, 1, 3, 7, 1, 3, 5}
	total := 0
	for i, w := range weights {
		if digits10[i] < '0' || digits10[i] > '9' {
			return false
		}
		total += int(digits10[i]-'0') * w
	}
	total += (int(digits10[8]-'0') * 5) / 10
	check := (10 - (total % 10)) % 10
	return digits10[9]-'0' == byte(check)
}

var (
	rrnCandidate   = regexp.MustCompile(`\b\d{6}[\s\-_·.*]?\d{7}\b`)
	cardCandidate4 = regexp.MustCompile(`\b\d{4}[\s\-_·.]?\d{4}[\s\-_·.]?\d{4}[\s\-_·.]?\d{1,4}\b`)
	cardCandidateA = regexp.MustCompile(`\b\d{4}[\s\-_·.]?\d{6}[\s\-_·.]?\d{5}\b`)
	bizCandidate   = regexp.MustCompile(`\b\d{3}[\s\-_·.]?\d{2}[\s\-_·.]?\d{5}\b`)
)

// ChecksumRecognizer는 체크섬 있는 식별번호 3종(KR_RRN/KR_CARD_NUMBER/
// KR_BIZ_REG_NO)을 탐지한다 (FR-PI-01, FR-PI-08).
type ChecksumRecognizer struct{}

var _ plugin.Detector = ChecksumRecognizer{}

func (ChecksumRecognizer) Name() string { return "ChecksumRecognizer" }

func (ChecksumRecognizer) Detect(_ context.Context, normalized string) ([]plugin.Finding, error) {
	var findings []plugin.Finding

	for _, loc := range rrnCandidate.FindAllStringIndex(normalized, -1) {
		match := normalized[loc[0]:loc[1]]
		digits := stripSeparators(match)
		if len(digits) == 13 && ValidateRRN(digits) {
			findings = append(findings, plugin.Finding{
				EntityType: "KR_RRN", Start: loc[0], End: loc[1], Score: 0.95, Text: match,
			})
		}
	}
	for _, pat := range []*regexp.Regexp{cardCandidate4, cardCandidateA} {
		for _, loc := range pat.FindAllStringIndex(normalized, -1) {
			match := normalized[loc[0]:loc[1]]
			digits := stripSeparators(match)
			if len(digits) >= 12 && len(digits) <= 19 && ValidateLuhn(digits) {
				findings = append(findings, plugin.Finding{
					EntityType: "KR_CARD_NUMBER", Start: loc[0], End: loc[1], Score: 0.95, Text: match,
				})
			}
		}
	}
	for _, loc := range bizCandidate.FindAllStringIndex(normalized, -1) {
		match := normalized[loc[0]:loc[1]]
		digits := stripSeparators(match)
		if len(digits) == 10 && ValidateBizRegNo(digits) {
			findings = append(findings, plugin.Finding{
				EntityType: "KR_BIZ_REG_NO", Start: loc[0], End: loc[1], Score: 0.9, Text: match,
			})
		}
	}
	return findings, nil
}
