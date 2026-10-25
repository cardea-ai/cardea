# docs/requirements.md 변경 이력

이 문서는 `docs/requirements.md`에 일어난 변경을 날짜순(최신 먼저)으로 기록한다. 본문에는
날짜 태그를 달지 않고, 변경 이력은 이 파일에서만 관리한다.

## 2026-10-25 (12차) — gitleaks CI 연동 + NFR-DQ-02 Go 벤치마크 확장

- NFR-SS-05 자가 탐지: gitleaks를 실제 설치해 저장소를 스캔(깨끗함). CI
  워크플로(`.github/workflows/gitleaks.yml`), 설정(`.gitleaks.toml`), 로컬
  실행 스크립트(`scripts/self-detect.sh`) 신설.
- NFR-DQ-02 Go 벤치마크 확장: `internal/detect/benchmark_test.go`에 고정
  시드 생성기로 470건(1차 파이썬 스파이크의 10배 규모) 평가셋을 만들어
  실제 배포 코드(`detect.ChecksumRecognizer`)로 재확인. 재현율 1.0000,
  정밀도 0.9945로 NFR-DQ-01 유지. 운송장번호류 디스트랙터의 우연한 체크섬
  통과(2/20)가 큰 표본에서도 똑같이 재현돼, 문맥 가중치 보강 필요성이
  더 분명해졌다. 이 벤치마크는 go test로 CI에 상시 편입된다(1회성 아님).

## 2026-10-25 (11차) — Presidio 실제 연동: REST(gRPC 아님) 확인

- Presidio Analyzer 공식 Docker 이미지(`mcr.microsoft.com/presidio-analyzer`)로
  실제 연동했다. ADR-0007·7장에 적은 "gRPC"는 틀렸다 — 실제 인터페이스는
  REST(Flask `/analyze`)뿐이다. 8장 안정성 검토 표를 정정했다.
  `internal/presidio` REST 클라이언트 신설, 실제 컨테이너 대상 통합 테스트
  3건 PASS. 기본 이미지는 영어 NLP 엔진만 탑재돼 한국어 요청은
  "No matching recognizers"로 거부된다는 것도 확인 — 한국어 지원은 별도
  작업으로 남겼다.

## 2026-10-25 (10차) — Bifrost MCP 경계 재확인: ADR-0007 전제 수정

- Bifrost core 임베딩(TODO.md 골격 다음 단계 3번) 착수 전 공식 자료를 다시 확인했다.
  GitHub README "Key Features"는 MCP를 "Advanced Features"에 넣지만 바로 아래
  "Enterprise Deployments" 섹션은 "guardrails, MCP gateway"를 엔터프라이즈 전용으로
  명시한다 — 공식 자료 자체의 내부 모순을 확인(이전엔 "자료마다 다르다"로만 적었는데,
  더 정확히는 "같은 벤더의 같은 공식 자료 안에서도 모순된다"). ADR-0007의 "MCP 게이트웨이
  1차 인증/allowlist는 Bifrost가 이미 제공" 전제를 폐기하고, Cardea 코어가 MCP 1차
  인증/allowlist를 직접 구현하는 쪽으로 8장 표·ADR-0007 본문을 수정했다. Bifrost core
  Go SDK(`github.com/maximhq/bifrost/core`)는 ADR-0001 원래 범위(LLM 프로바이더
  라우팅·폴백·스트리밍)로만 쓴다.

## 2026-10-25 (9차) — OpenBao 실제 연동 검증

- ADR-0007이 제안한 OpenBao(토큰 금고 크립토 백엔드)를 실제 docker 컨테이너로
  띄워 Transit 엔진 암복호화를 curl과 Go 클라이언트(`github.com/openbao/openbao/api/v2`,
  HashiCorp 클라이언트 아님)로 검증했다. `internal/vault/openbao.go` 구현, 통합
  테스트 4건 실서버 대상 PASS. ADR-0007 자체 문구는 바꾸지 않았다 — "제안"
  상태는 사용자 승인 전까지 유지하되, 실제로 동작한다는 근거가 쌓였다는 점만
  기록한다.

## 2026-10-25 (8차) — Getting Started UX를 FR/AG 수용 기준에 반영

- "10장 신설 — Getting Started 온보딩 UX를 수용 기준에 반영"을 추가했다.
  `docs/GETTING_STARTED.md`에서 발견한 6개 사용성 문제 중 실제 요구사항 성격인
  4건을 기존 ID(AG-12, FR-PI-05, AG-04/AG-09, FR-RP-02/FR-EXT-02)의 수용 기준으로
  확장했다 — `cardea quickstart` 원스텝 경로, 연결 내역 확인/되돌리기, 미리보기의
  대화형 전용 기본값, 에이전트 거버넌스 2단 노출, 서명 검증 실패 메시지 3요소.
  설치 플랫폼 분기는 수용 기준이 아니라 순수 문서/패키징 편의라 ID를 만들지
  않았다.

## 2026-10-25 (7차) — FR-AC-03 CBAC 기반 재식별 설계

- 3장에 "FR-AC-03 (CBAC 기반 동적 재식별 설계)" 절을 신설했다. 기존 "정적 역할 확인
  후 승인 절차"를 "역할+작업(task)+컨텍스트를 매 요청마다 평가해 재식별 범위를
  동적으로 좁히는" 구조로 다시 썼다 — 판정 입력 4종(신원·역할, 작업, 데이터 등급,
  컨텍스트), OPA/Rego 기반 3단계 출력(allow/partial/deny), deny 시에만 break-glass
  진입, 감사 로그 스키마를 명시했다. 유료(엔터프라이즈) 티어로 명확히 하고 무료
  티어와의 경계도 적었다.
- `docs/EDITIONS.md` 4절, `docs/POSITIONING.md` "하지 말아야 할 것"을 같은 기준으로
  갱신했다 — 단, 설계만 됐을 뿐 구현·벤치마크 전까지는 CBAC를 세일즈 메시지로 쓰지
  않는다는 제약은 유지했다.

## 2026-10-25 (6차) — 최종 방향 결정: Build 지속(조건부) 확정

- 1장에 "최종 방향 결정 (확정)" 절을 추가했다. 작업 1(경쟁력 2차 분석)·작업 2
  (Getting Started 가이드) 검토 후 사용자가 Build 지속(조건부)으로 최종 확정했다.
- 조건 4개: (1) 자체 개발 범위는 ADR-0007 OSS 조합의 접합부로 한정, (2) 가로채기
  메커니즘은 ADR-0008의 표준 프로토콜 프록시로 확정(훅은 보조), (3) 포지셔닝은
  "한국형" 표현을 뺀 로케일 무관 글로벌 표준으로 확정, (4) 사업화 전 선행 조건
  (NFR-DQ 벤치마크, FR-AC-03 CBAC급 보완, Getting Started 온보딩 흐름의 수용 기준
  반영) 유지.
- `docs/POSITIONING.md` 상태를 "초안"에서 "확정"으로 바꿨다(본문 변경 없음 — 이미
  반영된 좁힌 포지셔닝이 그대로 최종안).

## 2026-10-25 (5차) — ADR-0008 (게이트웨이 확정 + 대시보드 기술 선택)

- 8장에 ADR-0008을 추가했다(제안이 아니라 확정). 사용자가 로컬 게이트웨이(표준 프로토콜
  프록시) vs AI 도구별 훅 연동을 비교 검토한 뒤 게이트웨이로 확정했고, 같은 자리에서
  로컬 대시보드 기술도 Go 서버 렌더링(html/template) + htmx + Alpine.js + Tailwind로
  확정했다.
- 프록시 vs 훅 판단 근거: 표준 프로토콜(LLM API, MCP)은 벤더 무관·안정적이라 한 번
  구현하면 전체 클라이언트를 커버하지만, 도구별 훅 API는 N개 도구 = N개의 개별
  유지보수 표면이라 Nightfall이 겪은 "4개 분리된 표면" 문제를 재현한다. 훅은 유출
  방지의 대체재가 아니라 로컬 전용 파괴적 행동 차단의 보조 장치로만 쓴다.
- 대시보드 기술 판단 근거: 지표·토글·탭 위주 화면에는 SPA가 과하고 npm/Vite 빌드
  체인이 기여 장벽을 높인다. `go:embed`로 단일 바이너리 배포 원칙을 지키고, 서버
  상태를 템플릿·SSE로 그대로 내려 상태 복제 버그를 없앤다. 더 복잡한 화면(PL-01,
  PL-05)은 cardea-enterprise 콘솔에서 별도로 다룬다.

## 2026-10-25 (4차) — "한국형" 표현 제거, 글로벌 표준 포지셔닝 명문화

- 1장 보완의 제품 한 줄 정의와 "한국형 민감정보 탐지 정확도" 서술에서 "한국형"이라는
  표현을 제거했다. 사용자 지시: "글로벌 표준으로 개인정보·민감정보를 외부로 유출하지
  않는 것이 이 프로젝트의 목표"라는 원칙을 문구 수준까지 일관되게 반영.
  - 제품 정의: "한국형 보안 AI 게이트웨이" → "개인정보·민감정보가 외부로 유출되지
    않도록 보호하는 글로벌 표준 보안 AI 게이트웨이".
  - "한국형 탐지·N2SF/ISMS-P 증적" → "한국 민감정보 탐지·N2SF/ISMS-P 증적"(한국은
    지역명으로만 남기고, "한국형"을 Cardea 자체 기능의 브랜드 명칭으로 쓰지 않음).
  - ADR-0007 요약과 1장 보완 전반을 같은 기준으로 다듬었다.
- 경쟁사(Corepin, GenAI Shield, Airgap Studio 등)를 지칭할 때 쓰던 "한국형 PII" 표현도
  Cardea 자신의 포지셔닝 문맥에서는 "한국 민감정보"로 바꿨다. 경쟁사의 자체 마케팅
  문구를 그대로 인용한 부분(`docs/COMPETITIVE_ANALYSIS.md`)은 사실 인용이라 건드리지
  않았다.

## 2026-10-25 (3차) — 기술 스택 안정성 검토

- 8장에 "기술 스택 안정성 검토" 보완을 추가했다. ADR-0001(Bifrost core)·ADR-0007 대상
  OSS(Presidio, OpenBao, gitleaks)와 7장 기술스택의 TinyGo에 대해 유지보수 상태·성숙도
  등급·장기 리스크·완화책을 정리했다.
- 9장 미결 사항에 Bifrost 엔터프라이즈 경계 공식 문서 재확인, OpenBao의 OpenSSF 성숙도
  단계(Sandbox→Incubating) 추적, TinyGo 컴파일 스파이크 선행 검증을 추가했다.

## 2026-10-25 (2차) — ADR-0007 (OSS 조립 결정)

- 8장에 ADR-0007(신설 제안)을 추가했다. 탐지(Presidio)·가드레일 설계 참고(NeMo
  Guardrails)·토큰 금고 크립토(OpenBao)·자가 탐지(gitleaks)·서명/SBOM(cosign+syft/grype)·
  MCP 1차 allowlist(기존 Bifrost core)를 OSS로 조립하는 결정이다. 근거:
  `docs/COMPETITIVE_ANALYSIS.md`의 OSS 적극 채택 재검토.

## 2026-10-25 (1차) — 코드베이스 모듈 경로 변경

- `go.mod` 모듈 경로를 `github.com/aiport-dev/aiport`에서 `github.com/cardea-ai/cardea`로
  바꿨다(제품명 Cardea와 일치, 기존 경로는 오타로 추정되어 사용자 확인 후 변경).

## 2026-10-24 — 초안 작성

- 요구사항 ID 체계(FR/AG/PL/NFR), 추적 규칙, 용어, 원 요구사항 추적표를 작성했다.
- 1장 보완(포지셔닝과 핵심 차별점), 2장 보완(리포트 항목), 3장 보완(FR-TK 효율성 지표,
  FR-PI-04 가역 마스킹 토큰 설계, FR-GW 게이트웨이 코어 신설, 당근 라우터 교훈을 반영한
  기존 ID 갱신), 4장 PL-06 지표 대시보드, 5장 NFR-OB-02, 6장 티어 매트릭스, 8장
  ADR-0006(SDK 기반 어댑터) 신설 제안, 9장 미결 사항을 작성했다.
