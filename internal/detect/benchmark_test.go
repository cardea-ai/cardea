// NFR-DQ-02 확장 벤치마크 — 실제 배포되는 Go 탐지기(detect.ChecksumRecognizer)를
// 대상으로, eval/nfr-dq/의 46건짜리 1차 파이썬 스파이크보다 큰 규모(수백 건)로
// 재현율/정밀도를 재확인한다. 고정 시드로 결정론적으로 생성해 CI에서 매번
// 같은 결과가 나오게 한다 — 평가셋을 수백 건으로 늘려 NFR-DQ-01 기준(재현율
// 99%/정밀도 95%)이 실제 Go 코드에서도 유지되는지 본다(1차 스파이크는
// 파이썬 구현을 검증했을 뿐, 이게 Go로 포팅된 뒤에도 같은 수치가 나오는지는
// 별도로 확인해야 한다).
package detect

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
)

// mustValidChecksum은 body(체크섬 제외 자릿수)에 0~9를 하나씩 붙여 보며
// validate(검증 함수)를 통과하는 조합을 찾는다. 체크섬 공식 자체를 다시
// 구현하지 않고 실제 검증 함수를 그대로 써서 "정답 생성"과 "판정 로직"이
// 늘 같은 기준을 쓰게 한다.
func mustValidChecksum(body string, validate func(string) bool) (string, bool) {
	for d := 0; d <= 9; d++ {
		candidate := fmt.Sprintf("%s%d", body, d)
		if validate(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func randomDigits(rng *rand.Rand, n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('0' + rng.Intn(10))
	}
	return string(b)
}

// invalidChecksumVariant는 유효한 digits 문자열의 마지막 자리를 바꿔
// 체크섬이 깨진(= validate가 false인) 문자열을 돌려준다.
func invalidChecksumVariant(valid string, validate func(string) bool) string {
	last := valid[len(valid)-1] - '0'
	for delta := 1; delta <= 9; delta++ {
		candidate := valid[:len(valid)-1] + fmt.Sprintf("%d", (int(last)+delta)%10)
		if !validate(candidate) {
			return candidate
		}
	}
	return valid // 이론상 도달 안 함
}

type notation struct {
	name string
	// render는 "분리 전 자릿수 그룹"을 받아 구분자를 넣어 하나의 문자열로 만든다.
	render func(groups []string) string
}

var commonNotations = []notation{
	{"hyphen", func(g []string) string { return joinWith(g, "-") }},
	{"nosep", func(g []string) string { return joinWith(g, "") }},
	{"spaced", func(g []string) string { return joinWith(g, " ") }},
	{"dot", func(g []string) string { return joinWith(g, ".") }},
	{"underscore", func(g []string) string { return joinWith(g, "_") }},
	{"asterisk", func(g []string) string { return joinWith(g, "*") }},
	{"middot", func(g []string) string { return joinWith(g, "·") }},
	{"fullwidth", func(g []string) string { return toFullwidth(joinWith(g, "-")) }},
}

func joinWith(groups []string, sep string) string {
	out := groups[0]
	for _, g := range groups[1:] {
		out += sep + g
	}
	return out
}

func toFullwidth(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			out = append(out, rune(0xFF10+(r-'0')))
		case r == '-':
			out = append(out, 0xFF0D)
		default:
			out = append(out, r)
		}
	}
	return string(out)
}

type evalCase struct {
	text        string
	wantEntity  string // 양성이면 기대 엔티티, 음성이면 ""
	description string
}

// buildExpandedEvalSet은 결정론적 시드로 RRN/카드번호/사업자등록번호 양성·
// 음성 케이스를 수백 건 만든다.
func buildExpandedEvalSet() []evalCase {
	rng := rand.New(rand.NewSource(20261025)) // 고정 시드 — 재현 가능해야 함
	var cases []evalCase

	// --- KR_RRN: 20개 값 × 8가지 표기 + 한글숫자 1종 = 180건 양성 ---
	for i := 0; i < 20; i++ {
		body := randomDigits(rng, 12)
		valid, ok := mustValidChecksum(body, ValidateRRN)
		if !ok {
			continue
		}
		groups := []string{valid[:6], valid[6:]}
		for _, n := range commonNotations {
			cases = append(cases, evalCase{text: n.render(groups), wantEntity: "KR_RRN",
				description: fmt.Sprintf("rrn-%d-%s", i, n.name)})
		}
		// 한글 숫자 표기(앞 6자리만) — Normalizer가 처리해야 하는 경로
		hangulPrefix := toHangulDigits(valid[:6])
		cases = append(cases, evalCase{text: hangulPrefix + "-" + valid[6:], wantEntity: "KR_RRN",
			description: fmt.Sprintf("rrn-%d-hangul", i)})

		// 음성: 체크섬이 깨진 같은 모양
		bad := invalidChecksumVariant(valid, ValidateRRN)
		cases = append(cases, evalCase{text: bad[:6] + "-" + bad[6:], wantEntity: "",
			description: fmt.Sprintf("rrn-%d-bad-checksum", i)})
	}

	// --- KR_CARD_NUMBER: 15개 값 × (16자리 Luhn, 15자리 Amex 모양) × 4가지 표기 ---
	cardNotations := commonNotations[:4] // hyphen/nosep/spaced/dot만 — 카드번호엔 밑줄/중점/전각 안 씀
	for i := 0; i < 15; i++ {
		body16 := randomDigits(rng, 15)
		valid16, ok := mustValidChecksum(body16, ValidateLuhn)
		if ok {
			groups := []string{valid16[0:4], valid16[4:8], valid16[8:12], valid16[12:16]}
			for _, n := range cardNotations {
				cases = append(cases, evalCase{text: n.render(groups), wantEntity: "KR_CARD_NUMBER",
					description: fmt.Sprintf("card16-%d-%s", i, n.name)})
			}
			bad := invalidChecksumVariant(valid16, ValidateLuhn)
			cases = append(cases, evalCase{text: bad[0:4] + "-" + bad[4:8] + "-" + bad[8:12] + "-" + bad[12:16],
				wantEntity: "", description: fmt.Sprintf("card16-%d-bad-luhn", i)})
		}

		body15 := randomDigits(rng, 14)
		valid15, ok := mustValidChecksum(body15, ValidateLuhn)
		if ok {
			groups := []string{valid15[0:4], valid15[4:10], valid15[10:15]}
			for _, n := range cardNotations {
				cases = append(cases, evalCase{text: n.render(groups), wantEntity: "KR_CARD_NUMBER",
					description: fmt.Sprintf("card15-%d-%s", i, n.name)})
			}
		}
	}

	// --- KR_BIZ_REG_NO: 15개 값 × 4가지 표기 ---
	for i := 0; i < 15; i++ {
		body := randomDigits(rng, 9)
		valid, ok := mustValidChecksum(body, ValidateBizRegNo)
		if !ok {
			continue
		}
		groups := []string{valid[0:3], valid[3:5], valid[5:10]}
		for _, n := range cardNotations {
			cases = append(cases, evalCase{text: n.render(groups), wantEntity: "KR_BIZ_REG_NO",
				description: fmt.Sprintf("biz-%d-%s", i, n.name)})
		}
		bad := invalidChecksumVariant(valid, ValidateBizRegNo)
		cases = append(cases, evalCase{text: bad[0:3] + "-" + bad[3:5] + "-" + bad[5:10],
			wantEntity: "", description: fmt.Sprintf("biz-%d-bad-checksum", i)})
	}

	// --- 순수 음성(디스트랙터): 전화번호·날짜·운송장번호류, 60건 ---
	for i := 0; i < 20; i++ {
		phone := fmt.Sprintf("010-%04d-%04d", rng.Intn(10000), rng.Intn(10000))
		cases = append(cases, evalCase{text: "연락처는 " + phone + " 입니다.", wantEntity: "",
			description: fmt.Sprintf("distractor-phone-%d", i)})
		tracking := fmt.Sprintf("%06d-%07d", rng.Intn(1000000), rng.Intn(10000000))
		cases = append(cases, evalCase{text: "운송장 번호 " + tracking, wantEntity: "",
			description: fmt.Sprintf("distractor-tracking-%d", i)})
		plain := fmt.Sprintf("회의는 %d월 %d일 %d시에 시작합니다.", rng.Intn(12)+1, rng.Intn(28)+1, rng.Intn(24))
		cases = append(cases, evalCase{text: plain, wantEntity: "",
			description: fmt.Sprintf("distractor-plain-%d", i)})
	}

	return cases
}

var hangulDigitWords = []string{"공", "일", "이", "삼", "사", "오", "육", "칠", "팔", "구"}

func toHangulDigits(digits string) string {
	out := ""
	for _, d := range digits {
		out += hangulDigitWords[d-'0']
	}
	return out
}

// TestNFR_DQ_02_ExpandedGoChecksumRecognizerBenchmark은 실제 Go 탐지기를
// 수백 건 규모 평가셋으로 돌려 NFR-DQ-01 기준(재현율 99%/정밀도 95%)을
// 재확인한다.
func TestNFR_DQ_02_ExpandedGoChecksumRecognizerBenchmark(t *testing.T) {
	cases := buildExpandedEvalSet()
	if len(cases) < 300 {
		t.Fatalf("평가셋이 너무 작음(%d건) — 생성 로직 확인 필요(NFR-DQ-02는 수백 건 규모를 요구)", len(cases))
	}

	n := Normalizer{}
	r := ChecksumRecognizer{}
	ctx := context.Background()

	var tp, fn, fp, tn int
	var failures []string

	for _, c := range cases {
		normalized, err := n.Normalize(ctx, c.text)
		if err != nil {
			t.Fatalf("Normalize error on %q: %v", c.description, err)
		}
		findings, err := r.Detect(ctx, normalized)
		if err != nil {
			t.Fatalf("Detect error on %q: %v", c.description, err)
		}
		detected := make(map[string]bool)
		for _, f := range findings {
			detected[f.EntityType] = true
		}

		if c.wantEntity != "" {
			if detected[c.wantEntity] {
				tp++
			} else {
				fn++
				failures = append(failures, fmt.Sprintf("[FN] %s: %q", c.description, c.text))
			}
		} else {
			if len(detected) > 0 {
				fp++
				failures = append(failures, fmt.Sprintf("[FP] %s: %q -> %v", c.description, c.text, findings))
			} else {
				tn++
			}
		}
	}

	recall := float64(tp) / float64(tp+fn)
	precision := float64(tp) / float64(tp+fp)

	t.Logf("총 %d건 (양성 %d / 음성 %d) — TP=%d FN=%d FP=%d TN=%d", len(cases), tp+fn, fp+tn, tp, fn, fp, tn)
	t.Logf("재현율=%.4f (목표 >= 0.99), 정밀도=%.4f (목표 >= 0.95)", recall, precision)

	const recallTarget = 0.99
	const precisionTarget = 0.95

	if recall < recallTarget {
		t.Errorf("재현율 %.4f 가 NFR-DQ-01 목표 %.2f 미달", recall, recallTarget)
	}
	if precision < precisionTarget {
		t.Errorf("정밀도 %.4f 가 NFR-DQ-01 목표 %.2f 미달", precision, precisionTarget)
	}
	for _, f := range failures {
		t.Log(f)
	}
}
