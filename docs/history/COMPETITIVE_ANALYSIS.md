# docs/COMPETITIVE_ANALYSIS.md 변경 이력

이 문서는 `docs/COMPETITIVE_ANALYSIS.md`에 일어난 변경을 날짜순(최신 먼저)으로 기록한다.
본문에는 날짜 태그를 달지 않고, 변경 이력은 이 파일에서만 관리한다.

## 2026-10-25 (5차) — NFR-DQ-02 벤치마크 1차 스파이크

- `eval/nfr-dq/`에 체크섬 식별번호(KR_RRN/KR_CARD_NUMBER/KR_BIZ_REG_NO) 탐지
  검증용 파이썬 스파이크를 작성하고 실행했다. 6장 후속 조치에 결과를 기록.
- 결과: 합성 평가셋 46건(양성 34/음성 12)에서 재현율 1.0000, 정밀도 0.9714 —
  NFR-DQ-01 목표(99%/95%) 충족.
- 실제로 발견한 문제 2건: Amex 카드 패턴 누락 버그(수정함), 체크섬 단독 탐지의
  거짓양성 구조적 한계(운송장 번호가 우연히 RRN 체크섬 통과 — 문맥 가중치 보강
  필요, 미해결로 남김).
- 한계: 경쟁사 제품을 직접 실행한 비교가 아니라 Cardea 설계 자체의 목표 충족
  여부만 확인한 것 — "동등 조건 비교"는 여전히 미해결 과제.

## 2026-10-25 (4차) — "한국형" 표현 제거

- Cardea 자신의 포지셔닝·OSS 조합 설명에서 "한국형"이라는 표현을 모두 제거하고 "한국
  민감정보" 또는 "로케일별"로 바꿨다(3-1장 OSS 표, 4장 결론). 경쟁사(Airgap Studio,
  GenAI Shield)의 자체 마케팅 문구를 그대로 인용한 1장 비교표는 사실 인용이라 그대로
  뒀다.
- 사용자 지시: "글로벌 표준으로 개인정보·민감정보를 외부로 유출하지 않는 것"이 이
  프로젝트의 목표라는 원칙을 문구 수준까지 반영.

## 2026-10-25 (3차) — OSS 적극 채택 재검토

- "안정적인 OSS가 있으면 적극 사용해 프로젝트 부피를 줄인다"는 원칙으로 FR/AG/NFR 전 영역을
  재검토해 "3-1. OSS 적극 채택 재검토" 절을 추가했다.
- 채택 결정: Microsoft Presidio(탐지), OpenBao(토큰 금고 크립토 백엔드), gitleaks(자가
  시크릿 탐지), cosign+syft/grype(서명·SBOM), 기존 Bifrost core(MCP 1차 인증/allowlist).
  NVIDIA NeMo Guardrails는 설계 참고용으로만 쓰고 임베드하지 않음.
- 채택 보류: 소규모 토큰 금고 OSS(openpii-vault, clawwarden, npm pii-vault 등) — 성숙도·
  유지보수 이력 불명확.
- 4장 결론과 6장 후속 조치를 이 재검토를 반영해 갱신했다. `docs/requirements.md`에
  ADR-0007(신설 제안)로 공식화했다.

## 2026-10-25 (2차) — 해외 PII 토큰화 경쟁사 분석

- Skyflow, Protecto, Limina(구 Private AI), Nightfall AI, NoPII 5개 해외 경쟁사를 조사해
  5장으로 추가했다.
- 핵심 발견: Skyflow가 "에이전트 메모리/컨텍스트 그래프에서 PII를 뺀다"를 이미 공개
  메시지로 쓰고 있어, 기존에 Cardea만의 차별점으로 여겼던 주장이 무효화됐다. Protecto의
  CBAC(역할+작업+컨텍스트 기반 동적 재식별)가 Cardea FR-AC-03(break-glass)보다 설계상
  앞서 있다는 격차도 확인했다.
- 이 분석이 `docs/POSITIONING.md`의 포지셔닝을 "오픈코어·로컬 우선·클라우드 비의존 +
  통합 정책 엔진 + 세션 오염 추적"으로 좁히는 근거가 됐다.

## 2026-10-24 — 최초 작성

- 해외 범용 AI 게이트웨이(LiteLLM, Portkey, NeuralTrust TrustGate, Kong AI Gateway,
  Cloudflare AI Gateway)와 국내 한국형 PII/DLP 경쟁사(Airgap Studio, GenAI Shield,
  Corepin, 엘세븐시큐리티)를 비교했다(1장).
- 명세 대비 평가(2장), 1차 Build vs Buy 판단(3장), 결론(4장), 후속 조치 제안(현재 6장의
  초판)을 작성했다.
