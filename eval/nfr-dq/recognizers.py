"""
Cardea FR-PI 탐지 로직 검증 스파이크 (NFR-DQ-02)

이 모듈은 아직 Go 코어가 없는 상태에서, ADR-0007(Presidio 채택)과 FR-PI-11(정규화
선행)·FR-PI-01/08(체크섬 있는 식별번호 탐지) 설계가 실제로 NFR-DQ-01 기준(재현율
99%/정밀도 95%)을 만족하는지 먼저 검증하기 위한 파이썬 스파이크다. 실제 구현은
Go + Presidio(또는 WASM 플러그인)로 가져가되, 여기서 검증한 정규화·체크섬 로직은
그대로 포팅한다.

Presidio의 PatternRecognizer를 기반 클래스로 쓰되, 체크섬 검증은 validation_result
훅으로 구현한다(Presidio가 공식 지원하는 확장 지점).
"""
from __future__ import annotations

import re
import unicodedata
from typing import Optional

from presidio_analyzer import Pattern, PatternRecognizer, RecognizerResult


# ---------------------------------------------------------------------------
# FR-PI-11: 정규화 (판정 전에 반드시 선행)
# ---------------------------------------------------------------------------

_HANGUL_DIGIT_MAP = {
    "영": "0", "공": "0", "일": "1", "이": "2", "삼": "3", "사": "4",
    "오": "5", "육": "6", "칠": "7", "팔": "8", "구": "9",
}

_FULLWIDTH_DIGIT_MAP = {chr(0xFF10 + i): str(i) for i in range(10)}


def normalize(text: str) -> str:
    """공백/구분자 제거, 전각 문자 반각화, 한글 숫자 치환을 선행한다 (FR-PI-11).

    주의: 원문 레이아웃을 보존해야 하는 하위 처리기(FR-PI-06 형태보존)는 별도
    경로이므로, 이 함수는 "탐지 판정용 정규화 뷰"만 만든다 — 원문을 직접
    덮어쓰지 않는다는 게 실제 구현에서 지켜야 할 불변식이다.
    """
    # 1) 유니코드 정규화(NFKC) — 전각 숫자/문자를 반각으로 통일
    out = unicodedata.normalize("NFKC", text)
    # 2) 한글 숫자 단어 치환 (문맥상 숫자 나열로 보이는 경우만, 과교정 방지를
    #    위해 2글자 이상 연속된 한글 숫자 단어만 치환한다)
    def _replace_hangul_digits(match: re.Match) -> str:
        return "".join(_HANGUL_DIGIT_MAP[ch] for ch in match.group(0))

    hangul_digit_run = "(?:" + "|".join(_HANGUL_DIGIT_MAP.keys()) + "){2,}"
    out = re.sub(hangul_digit_run, _replace_hangul_digits, out)
    return out


def _strip_separators(digits_with_seps: str) -> str:
    """숫자 사이의 공백·하이픈·언더스코어·중간점 등 구분자를 제거한다."""
    return re.sub(r"[\s\-_·.*]", "", digits_with_seps)


# ---------------------------------------------------------------------------
# 체크섬 검증기
# ---------------------------------------------------------------------------

def validate_rrn_checksum(digits13: str) -> bool:
    """주민등록번호(13자리) 체크섬 검증 (공식 가중합 알고리즘)."""
    if len(digits13) != 13 or not digits13.isdigit():
        return False
    weights = [2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5]
    total = sum(int(d) * w for d, w in zip(digits13[:12], weights))
    check = (11 - (total % 11)) % 10
    return check == int(digits13[12])


def validate_luhn(digits: str) -> bool:
    """카드번호 Luhn 체크섬 검증."""
    if not digits.isdigit() or len(digits) < 12:
        return False
    total = 0
    for i, ch in enumerate(reversed(digits)):
        d = int(ch)
        if i % 2 == 1:
            d *= 2
            if d > 9:
                d -= 9
        total += d
    return total % 10 == 0


def validate_biz_reg_no(digits10: str) -> bool:
    """사업자등록번호(10자리) 체크섬 검증 (국세청 공식 알고리즘)."""
    if len(digits10) != 10 or not digits10.isdigit():
        return False
    weights = [1, 3, 7, 1, 3, 7, 1, 3, 5]
    total = sum(int(d) * w for d, w in zip(digits10[:9], weights))
    total += (int(digits10[8]) * 5) // 10
    check = (10 - (total % 10)) % 10
    return check == int(digits10[9])


# ---------------------------------------------------------------------------
# Presidio PatternRecognizer 구현
# ---------------------------------------------------------------------------

class KoreanRRNRecognizer(PatternRecognizer):
    """주민등록번호 — 느슨한 패턴으로 후보를 잡고, 체크섬으로 확정한다."""

    PATTERNS = [
        Pattern(
            name="rrn_candidate",
            # 정규화된(구분자 제거 전) 텍스트에서 구분자가 있어도/없어도 잡는다
            regex=r"\b\d{6}[\s\-_·.*]?\d{7}\b",
            score=0.4,
        )
    ]

    def __init__(self):
        super().__init__(
            supported_entity="KR_RRN",
            patterns=self.PATTERNS,
            name="KoreanRRNRecognizer",
        )

    def validate_result(self, pattern_text: str) -> Optional[bool]:
        digits = _strip_separators(pattern_text)
        return validate_rrn_checksum(digits) if len(digits) == 13 else False


class KoreanCardNumberRecognizer(PatternRecognizer):
    PATTERNS = [
        Pattern(
            name="card_candidate_4x4",
            # Visa/Mastercard/Discover 등 4-4-4-(1~4) 그룹핑
            regex=r"\b\d{4}[\s\-_·.]?\d{4}[\s\-_·.]?\d{4}[\s\-_·.]?\d{1,4}\b",
            score=0.3,
        ),
        Pattern(
            name="card_candidate_amex",
            # Amex 4-6-5 그룹핑 (15자리)
            regex=r"\b\d{4}[\s\-_·.]?\d{6}[\s\-_·.]?\d{5}\b",
            score=0.3,
        ),
    ]

    def __init__(self):
        super().__init__(
            supported_entity="KR_CARD_NUMBER",
            patterns=self.PATTERNS,
            name="KoreanCardNumberRecognizer",
        )

    def validate_result(self, pattern_text: str) -> Optional[bool]:
        digits = _strip_separators(pattern_text)
        return validate_luhn(digits) if 12 <= len(digits) <= 19 else False


class KoreanBizRegNoRecognizer(PatternRecognizer):
    PATTERNS = [
        Pattern(
            name="bizregno_candidate",
            regex=r"\b\d{3}[\s\-_·.]?\d{2}[\s\-_·.]?\d{5}\b",
            score=0.3,
        )
    ]

    def __init__(self):
        super().__init__(
            supported_entity="KR_BIZ_REG_NO",
            patterns=self.PATTERNS,
            name="KoreanBizRegNoRecognizer",
        )

    def validate_result(self, pattern_text: str) -> Optional[bool]:
        digits = _strip_separators(pattern_text)
        return validate_biz_reg_no(digits) if len(digits) == 10 else False


ALL_RECOGNIZERS = [
    KoreanRRNRecognizer(),
    KoreanCardNumberRecognizer(),
    KoreanBizRegNoRecognizer(),
]


def detect(text: str) -> list[RecognizerResult]:
    """정규화(FR-PI-11) 후 모든 리코그나이저를 순서대로 돌려 결과를 합친다.

    presidio_analyzer.AnalyzerEngine을 쓰지 않고 PatternRecognizer를 직접
    호출하는 이유: 기본 AnalyzerEngine은 언어별 NlpEngine(spaCy 모델)이 있어야
    하는데, 체크섬 기반 패턴 인식 자체는 NLP 토큰화가 필요 없다(Presidio도
    컨텍스트 강화에만 NLP 아티팩트를 쓴다). 이 스파이크는 그 경로를 분리해
    무거운 모델 다운로드 없이 핵심 로직만 빠르게 검증한다.
    """
    normalized = normalize(text)
    results: list[RecognizerResult] = []
    for rec in ALL_RECOGNIZERS:
        results.extend(rec.analyze(normalized, entities=[rec.supported_entities[0]]))
    return results
