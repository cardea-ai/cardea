"""
NFR-DQ-02 벤치마크 러너.

eval_set.json의 각 케이스를 recognizers.detect()에 돌려, 엔티티 단위로
재현율(recall)·정밀도(precision)·F1을 계산하고 NFR-DQ-01 기준(재현율 99%/
정밀도 95%)과 비교해 보고한다.

판정 규칙(이 스파이크의 단순화):
  - positive case(expect != null): detect() 결과 중 expect 타입이 하나라도
    있으면 TP, 없으면 FN.
  - negative case(expect == null): detect() 결과가 하나도 없으면 TN, 하나라도
    있으면 FP(몇 개가 잡히든 케이스 단위로 1건의 FP로 집계).
"""
from __future__ import annotations

import json
from pathlib import Path

from recognizers import detect

EVAL_SET_PATH = Path(__file__).parent / "eval_set.json"
NFR_DQ_01_RECALL_TARGET = 0.99
NFR_DQ_01_PRECISION_TARGET = 0.95


def run() -> None:
    data = json.loads(EVAL_SET_PATH.read_text(encoding="utf-8"))
    cases = data["cases"]

    tp = fn = fp = tn = 0
    failures: list[str] = []

    for case in cases:
        text = case["text"]
        expect = case.get("expect")
        results = detect(text)
        detected_types = {r.entity_type for r in results}

        if expect is not None:
            if expect in detected_types:
                tp += 1
            else:
                fn += 1
                failures.append(f"[FN] {case['id']}: '{text}' → 기대 {expect}, 탐지 {detected_types or '없음'}")
        else:
            if detected_types:
                fp += 1
                failures.append(f"[FP] {case['id']}: '{text}' → 탐지 {detected_types} (음성 케이스인데 탐지됨)")
            else:
                tn += 1

    recall = tp / (tp + fn) if (tp + fn) else float("nan")
    precision = tp / (tp + fp) if (tp + fp) else float("nan")
    f1 = (2 * precision * recall / (precision + recall)) if (precision + recall) else float("nan")

    print("=" * 70)
    print("Cardea FR-PI 체크섬 식별번호 탐지 — NFR-DQ-02 벤치마크 결과")
    print("=" * 70)
    print(f"총 케이스: {len(cases)}건 (양성 {tp + fn}건, 음성 {fp + tn}건)")
    print(f"TP={tp}  FN={fn}  FP={fp}  TN={tn}")
    print(f"재현율(Recall):    {recall:.4f}  (NFR-DQ-01 목표: >= {NFR_DQ_01_RECALL_TARGET})")
    print(f"정밀도(Precision): {precision:.4f}  (NFR-DQ-01 목표: >= {NFR_DQ_01_PRECISION_TARGET})")
    print(f"F1:                {f1:.4f}")
    print("-" * 70)
    recall_ok = recall >= NFR_DQ_01_RECALL_TARGET
    precision_ok = precision >= NFR_DQ_01_PRECISION_TARGET
    print(f"재현율 기준 {'충족' if recall_ok else '미충족'}, 정밀도 기준 {'충족' if precision_ok else '미충족'}")
    if failures:
        print("-" * 70)
        print("세부 실패 케이스:")
        for line in failures:
            print(" ", line)
    print("=" * 70)


if __name__ == "__main__":
    run()
