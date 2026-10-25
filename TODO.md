# TODO — 다음 진행 작업

상태: 2026-10-25. 작업 1~3 모두 완료. 방향은 **Build 지속(조건부)**로 확정됐다
(`docs/requirements.md` 1장 "최종 방향 결정" 참고). 다음 단계는 구현 착수 전
선행 조건 처리다.

## 작업 1 — 경쟁력 2차 분석 (완료, 2026-10-25)

결과는 `docs/COMPETITIVE_ANALYSIS.md` 5~6장, `docs/POSITIONING.md`에 반영했다. 핵심
결론:

- Skyflow·Protecto·Limina·Nightfall AI·NoPII 5개 해외 경쟁사를 조사했다.
- 가설과 다르게 확인된 사실 둘:
  1. "에이전트 메모리에 PII를 안 남긴다"는 Skyflow가 이미 공개 메시지로 쓰고 있다.
     Cardea만의 차별 사례가 아니다. → `docs/POSITIONING.md`의 데모 계획을 "통합
     정책 엔진(사람 채팅+에이전트 툴 호출) + 세션 오염 추적(AG-04)"으로 좁혔다.
  2. Protecto의 CBAC(역할+작업+컨텍스트 기반 동적 재식별)가 Cardea의 FR-AC-03
     (break-glass: 승인 거쳐 예외 허용)보다 설계상 앞서 있다. 사업화 전 명세 보완
     필요(`docs/COMPETITIVE_ANALYSIS.md` 6장 후속 조치 항목 추가).
- 그대로 유지되는 차별점: (a) 오픈코어·로컬 우선·클라우드 비의존 — 다섯 경쟁사 모두
  SaaS/호스팅 볼트가 기본이거나(Skyflow/Protecto/Nightfall) 클라우드 프록시 전용
  (NoPII)이라 "무료 자체 호스팅 바이너리"가 없다. (b) 사람 채팅 + 에이전트 툴 호출을
  하나의 정책 엔진으로 묶는 것 — Nightfall은 공식 문서에서 콘텐츠 검사와 툴 거버넌스를
  4개 분리된 표면으로 나누고 "MCP Gateway는 프롬프트를 스캔하지 않는다"고 명시한다.

### 추가 — OSS 적극 채택 재검토 (Build 범위 축소, ADR-0007)

"안정적인 OSS가 있으면 적극 사용해 프로젝트 부피를 줄인다"는 원칙으로 FR/AG/NFR 전
영역을 다시 훑었다(`docs/COMPETITIVE_ANALYSIS.md` 3-1장, `docs/requirements.md`
ADR-0007). 채택 대상:

- FR-PI 탐지 엔진 → Microsoft Presidio(MIT) — 한국 전용 recognizer 이미 포함, 자체
  NER/정규식 엔진 재구축 불필요.
- 가드레일 설계 참고 → NVIDIA NeMo Guardrails(Apache 2.0)의 5-rail 모델.
- NFR-SS-05 자가 탐지 → gitleaks(MIT) 그대로 사용.
- NFR-SC 서명/SBOM → cosign + syft/grype(모두 Apache 2.0, 이미 방향 일치, 도구명 명시).
- FR-PI-04/FR-TK-05 토큰 금고 크립토 백엔드 → OpenBao(MPL 2.0, Vault의 BSL 이전
  포크, Linux Foundation)의 Transit/Transform 엔진. **HashiCorp Vault 자체는 BSL이라
  배제.**
- AG-02/03 MCP 게이트웨이 1차 인증/allowlist → 이미 채택한 Bifrost core가 제공 —
  재구현 불필요. Cardea는 Bifrost가 안 하는 "툴 호출 콘텐츠 검사"만 추가.
- 채택 안 함: 토큰 금고용 소규모 OSS(openpii-vault, clawwarden 등) — 성숙도/보안
  검증 불명확, 금고처럼 critical한 컴포넌트에는 부적합.

결과: 자체 개발 범위가 "탐지 엔진 자체"에서 "로케일별 탐지 플러그인(한국 민감정보
유형 포함) + 세션 스코프 매핑 로직 + 통합 정책 오케스트레이션"으로 좁혀졌다.

## 작업 2 — 예상 최종 사용 가이드 (Getting Started) 작성 (완료, 2026-10-25)

`docs/GETTING_STARTED.md`에 v0.1 무료 티어 기능 전체(설치~지표 대시보드, 11단계)의
접근 경로를 작성했다. 1차 드래프트에서 발견한 사용성 문제 6개를 모두 흐름에 반영해
개정했다(`cardea quickstart` 원스텝화, 연결 가시성·되돌리기, 미리보기 범위 제한,
에이전트 거버넌스 단계적 노출, 서명 실패 사유 설명, 설치 플랫폼 분기). 핵심 결과:

- 필수 경로(설치→quickstart→첫 마스킹 체감)는 약 2분 30초로 "5분 목표"를 여유 있게
  만족한다. `init`+`connect` 2단계를 `quickstart` 1단계로 합쳐 NoPII의 "SDK
  base_url 한 줄 교체"와 연결 단계 수를 맞췄다 — 남은 유일한 구조적 격차는 설치
  1단계(로컬 바이너리 vs SaaS 가입)뿐이다.
- 구현 시 확인할 검증 항목 3개(설치 자동 감지 실패율, quickstart 질문 최소화 여부,
  에이전트 탭 기본/고급 분리의 실제 효과)를 문서에 남겼다.

## 작업 3 — 최종 방향 결정 (완료, 2026-10-25)

**결론: Build 지속(조건부).** Buy/조합 전환이나 스코프 폐기가 아니다. 네 조건은
`docs/requirements.md` 1장 "최종 방향 결정"에 확정 기록했다:

1. 자체 개발 범위는 ADR-0007 OSS 조합의 접합부로 한정.
2. 가로채기 메커니즘은 ADR-0008(표준 프로토콜 프록시, 훅은 보조)로 확정.
3. 포지셔닝은 "한국형" 표현을 뺀 로케일 무관 글로벌 표준으로 확정 — 핵심 가치
   제안은 "개인정보·민감정보 외부 유출 방지 글로벌 표준 + 통합 정책 엔진(사람
   채팅+에이전트 툴 호출) + 오픈코어 PLG".
4. 사업화 전 선행 조건은 아래 "다음 단계"로 넘긴다.

`docs/POSITIONING.md` 상태를 "초안"에서 "확정"으로 전환했다.

## 다음 단계 — 구현 착수 전 선행 조건 (전부 완료, 2026-10-25)

- [x] NFR-DQ-02 한국어 평가셋으로 Corepin/Airgap Studio와 동등 조건 벤치마크를 돌려
      Cardea 탐지 엔진(Presidio 기반)의 실제 위치를 확인한다. **1차 스파이크 완료
      (2026-10-25, `eval/nfr-dq/`)**: 체크섬 식별번호 탐지(KR_RRN/KR_CARD_NUMBER/
      KR_BIZ_REG_NO)가 합성 평가셋 46건에서 재현율 1.0000·정밀도 0.9714로
      NFR-DQ-01 목표(99%/95%) 충족. Amex 카드 패턴 버그를 고쳤고, 체크섬 단독
      탐지의 거짓양성 구조적 한계(문맥 가중치 필요)를 발견했다. **남은 일**: (a)
      경쟁사 제품과의 실제 동등 조건 비교는 여전히 미해결(합성 데이터로 Cardea
      설계만 검증했다), (b) 이름·주소 등 체크섬 없는 엔티티의 NLP 기반 탐지
      검증, (c) 평가셋을 수백 건 규모로 확장. 상세: `eval/nfr-dq/README.md`.
- [x] FR-AC-03(break-glass) 설계를 Protecto의 CBAC급 — 역할+작업+컨텍스트 기반 동적
      재식별 — 로 보완한다(`docs/COMPETITIVE_ANALYSIS.md` 6장 참고). **완료
      (2026-10-25)**: `docs/requirements.md` 3장에 CBAC 설계(판정 입력 4종, OPA/Rego
      기반 allow/partial/deny, break-glass는 deny 시 2차 경로, 감사 로그 스키마)를
      명세화했다. `docs/EDITIONS.md`·`docs/POSITIONING.md`도 함께 갱신. **남은 일**:
      이건 설계 문서일 뿐 구현이 아니다 — 실제 Go+OPA 구현과 CBAC 판정 지연(latency)
      검증은 구현 착수 단계로 넘긴다. 명세 개정이 별도 PR로 필요하면 그때 처리.
- [x] `docs/GETTING_STARTED.md`의 온보딩 흐름(특히 `cardea quickstart` 원스텝 경로,
      연결 가시성/되돌리기, 에이전트 거버넌스 단계적 노출)을 실제 FR/AG 수용 기준에
      반영한다. **완료(2026-10-25)**: `docs/requirements.md` 10장에 AG-12·FR-PI-05·
      AG-04/AG-09·FR-RP-02/FR-EXT-02의 수용 기준을 확장해 못박았다. 설치 플랫폼
      분기는 요구사항이 아닌 문서/패키징 편의로 분류해 ID를 만들지 않았다.
- [x] ADR-0007·ADR-0008에 따라 go.mod에 실제 의존성(Presidio 연동 방식, OpenBao
      클라이언트, gitleaks 호출 래퍼 등)을 추가하고 최소 골격 구현을 시작한다.
      **완료(2026-10-25, 1차 골격)**: 실제 의존성(`github.com/open-policy-agent/opa`,
      Go 1.26.1)을 추가하고, 6개 패키지로 최소 골격을 짰다 —
      `internal/plugin`(ADR-0002 6종 플러그인 슬롯 인터페이스),
      `internal/detect`(FR-PI-01/08/11 체크섬 탐지·정규화, `eval/nfr-dq/` 파이썬
      스파이크를 Go로 포팅), `internal/policy`(OPA/Rego 기반 정책 평가,
      NFR-AV-02 fail-closed), `internal/forward`(ADR-0001 Forwarder 인터페이스 +
      `StubForwarder` — Bifrost core는 아직 임베드하지 않았고 502로 명시),
      `internal/gateway`(GW-01 듀얼 프로토콜 HTTP 핸들러), `cmd/cardea`(엔트리포인트,
      127.0.0.1 바인딩). `go build`/`go vet`/`go test` 전부 통과, 실제 서버 기동 후
      curl로 end-to-end 확인(정상 요청→allow→502 스텁, RRN 포함 요청→partial→502
      스텁 — 로그에 `grade`/`findings`/`decision`이 올바르게 찍힘, 원문은 안 남음).
      테스트는 CLAUDE.md 2장 규칙대로 `TestFR_PI_01_...`/`TestNFR_AV_02_...`로
      명명. **남은 일**: (a) Bifrost core 실제 임베딩(현재 StubForwarder),
      (b) Processor(마스킹/토큰 발급, FR-PI-04) 미구현 — 정책이 partial이어도
      아직 마스킹 없이 그대로 전달 시도함(단, Forwarder가 막혀 있어 실제 유출은
      없음), (c) Presidio gRPC 연동, OpenBao 클라이언트, gitleaks 래퍼는 아직
      추가 안 함.

## 다음 단계 — 골격 다음 단계 (전부 완료, 2026-10-25)

- [x] `internal/forward`에 Bifrost core를 실제로 임베드해 StubForwarder를 교체한다
      (ADR-0001). Bifrost의 OSS/엔터프라이즈 경계가 자료마다 다르다는 리스크(8장
      "기술 스택 안정성 검토")를 임베딩 전에 공식 라이선스 문서로 먼저 재확인한다.
      **완료(2026-10-25)**: 재확인 결과 공식 자료 자체가 모순됨을 확인(README
      "Key Features"는 MCP를 OSS처럼 적어두고 "Enterprise Deployments"는 "MCP
      gateway"를 엔터프라이즈 전용으로 명시) — 이에 따라 ADR-0007의 "Bifrost가
      MCP allowlist 제공" 전제를 폐기하고 8장/ADR-0007을 수정했다(위 항목들 참고).
      Bifrost 임베딩 자체는 ADR-0001 원래 범위(LLM 프로바이더 라우팅·폴백)로
      좁혀서 진행 — `go get github.com/maximhq/bifrost/core`(Apache-2.0)로 실제
      의존성 추가, `internal/forward/bifrost.go`에 `BifrostForwarder` 구현
      (`bifrost.Init` + `ChatCompletionRequest`, 환경변수 기반 `envAccount`로
      프로바이더 키 주입, Cardea 자체 로깅을 쓰도록 `NewNoOpLogger()`로 원문이
      Bifrost 로그로 새지 않게 함). `internal/gateway.newForwarder()`에서
      `CARDEA_ENABLE_BIFROST=1`일 때만 StubForwarder 대신 쓰도록 선택적으로
      켬(기본값은 여전히 솔직한 502 스텁). 테스트는 실제 Bifrost SDK를
      초기화하고, 키 없는 프로바이더 요청이 구조화된 에러로 돌아오는 경로를
      검증(`TestFR_GW_Bifrost*` 2건 PASS). 서버를 `CARDEA_ENABLE_BIFROST=1`로
      실제 기동해 curl로 end-to-end 확인: RRN 포함 요청 → `decision=partial` →
      마스킹 → 실제 Bifrost SDK 호출 → "`CARDEA_OPENAI_API_KEY` 환경변수 필요"
      구조화 에러로 502(이 환경에 프로바이더 API 키가 없어 실제 모델 호출까지는
      확인 못 함 — 비밀값을 임의로 만들어 넣지 않았다). **남은 일**: (a) 실제
      프로바이더 키가 있는 환경에서 진짜 모델 응답까지 받는 end-to-end 재검증,
      (b) ADR-0006 SDK 어댑터 계층(OpenAI/Anthropic 프로토콜 변환)이 아직 없어
      `chatRequestBody`가 실제 두 프로토콜을 다 못 받는다 — 지금은 "provider/
      model/messages" 평문 포맷 하나만 받는다, (c) MCP 1차 인증/allowlist는
      Bifrost에 기대지 않고 Cardea가 직접 구현해야 한다(위에서 범위를 좁힌
      결과).
- [x] `internal/processor`(가칭)로 FR-PI-04 가역 마스킹(토큰 발급 — CSPRNG 난수 +
      암호화 토큰 금고)을 구현하고, `partial` 판정일 때 Forwarder 호출 전에 반드시
      거치게 한다. 지금은 partial이어도 평문이 그대로 Forward 직전까지 간다(Forwarder
      가 막혀 있어 외부로 안 나가지만, 이 상태로 Bifrost를 먼저 붙이면 안 된다 — 순서
      중요). **완료(2026-10-25)**: `internal/vault`(토큰 금고, AES-256-GCM 메모리
      구현체 `MemoryVault` — OpenBao 연동 전 자리값)와 `internal/processor`
      (`TokenProcessor.Mask`/`Restore`)를 추가했다. 토큰 형식 `⟦TYPE_난수8자⟧`
      (CSPRNG base62, 순번 아님), 스코프 격리(다른 스코프에서 못 풂), 만료 시
      fail-closed(원문도 에러도 아닌 토큰 그대로 유지), 한 호출 내 참조 일관성
      (같은 원문 → 같은 토큰)을 전부 테스트로 검증(`TestFR_PI_04_*`, 9건 전부
      PASS). `internal/gateway`에 실제로 꽂아서 partial 판정 시 Forward 호출
      전에 마스킹이 실행됨을 로그로 확인(`decision=partial` 뒤 Forward가 받는
      본문은 마스킹된 본문 — 단위 테스트로 교체 전후 문자열을 직접 비교해
      원문이 안 남는 것까지 확인). **남은 일**: (a) 세션 스코프가 아직 "요청
      단위"다 — FR-MEM 세션 개념이 들어오면 교체해야 한다, (b) 토큰 금고는
      메모리 구현체라 프로세스 재시작 시 전부 사라진다(OpenBao로 교체 전까지
      의도된 동작), (c) 응답 경로(Restore)는 아직 실제로 안 쓰인다 — Bifrost가
      붙어 실제 모델 응답이 생겨야 호출할 자리가 생긴다.
- [x] 토큰 금고 크립토 백엔드로 OpenBao(ADR-0007) 클라이언트를 연동한다.
      **완료(2026-10-25)**: `internal/vault/openbao.go`에 `OpenBaoVault`를
      추가했다 — HashiCorp 클라이언트가 아니라 OpenBao 공식 Go 클라이언트
      (`github.com/openbao/openbao/api/v2`, MPL-2.0/Apache 계열, HashiCorp
      Vault BSL 클라이언트는 아예 안 씀)로 Transit 엔진의 encrypt/decrypt만
      호출한다. 실제 OpenBao 서버(docker `openbao/openbao:latest server
      -dev`)를 띄워 Transit 엔진·키 생성·암복호화를 curl로 먼저 확인한 뒤,
      Go 클라이언트로 같은 걸 재현했다. `TestFR_PI_04_OpenBao*` 4건을 실제
      서버에 대고 돌려 전부 PASS(라운드트립, 스코프 격리, 만료 fail-closed,
      "로컬에는 ciphertext만 있고 암호화 키는 없다"는 설계 불변식까지
      검증). 환경변수가 없으면 자동 skip이라 CI에서 OpenBao 없이도
      안전하게 통과한다. `internal/gateway.newVault()`에서
      `CARDEA_OPENBAO_ADDR` 유무로 MemoryVault/OpenBaoVault를 선택하고,
      실제 게이트웨이 프로세스를 OpenBao 연동 모드로 띄워 partial 판정
      마스킹이 실제 OpenBao Transit 키로 동작하는 것까지 end-to-end로
      확인했다. **남은 일**: (a) OpenBao 다운 시 재연결/서킷브레이커
      정책(지금은 최초 연결 실패만 fail-closed, 런타임 중 끊기면 각 호출의
      에러로만 드러남), (b) 토큰 발급 메타데이터(scope/만료)는 아직 Cardea
      프로세스 메모리에만 있다 — 영속화하려면 OpenBao KV에도 저장하거나
      별도 스토리지가 필요.
- [x] Presidio(ADR-0007)를 별도 프로세스(gRPC)로 붙여 체크섬 없는 엔티티(이름·주소
      등)를 `internal/detect`에 추가한다 — 지금 Go 구현은 체크섬 가능한 3종만 다룬다.
      **완료(2026-10-25)**: 실제로 연동하는 과정에서 공식 Docker 이미지
      (`mcr.microsoft.com/presidio-analyzer`)가 gRPC가 아니라 **REST**(Flask
      `/analyze`)만 제공한다는 걸 확인 — "gRPC"는 Cardea 자체 ML 분류기(7장)를
      위한 스택 선택이었고 Presidio 자체의 인터페이스는 아니었다(용어 정정,
      8장 안정성 검토 표에 기록). `internal/presidio`에 REST 클라이언트
      (`plugin.Detector` 구현)를 신설, 실제 Presidio 컨테이너로 통합 테스트
      3건 PASS(PERSON/EMAIL_ADDRESS 탐지, 인덱스 정합성, 한국어 미지원
      네거티브 테스트). `internal/gateway.newDetectors()`에서
      `CARDEA_PRESIDIO_ADDR` 설정 시 체크섬 탐지기에 추가로 꽂히도록 선택적
      구성, 연결 실패 시 fail-closed로 기동 중단. 서버를 실제로 띄워 curl로
      end-to-end 확인: "My name is John Smith and my email is
      john@example.com" → `findings=3`(PERSON/EMAIL_ADDRESS/URL) →
      `decision=partial`. **남은 일**: (a) 공식 이미지는 영어(en) NLP 엔진만
      탑재 — 한국어는 "No matching recognizers" 에러로 거부됨, 한국어
      NLP 엔진 구성은 별도 작업으로 남김(FR-PI 로케일별 탐지의 핵심 격차),
      (b) Presidio 탐지 결과를 `internal/processor`의 마스킹 토큰 발급과
      엔드투엔드로 묶는 테스트는 아직 없음(체크섬 탐지기 경로만 테스트됨).
- [x] NFR-SS-05 자가 탐지(gitleaks 래퍼)를 CI에 연동한다. **완료(2026-10-25)**:
      `go install github.com/zricethezav/gitleaks/v8`로 실제 설치해 저장소
      스캔 실행(현재 깨끗함, 0건). `.github/workflows/gitleaks.yml`(push/PR마다
      `gitleaks/gitleaks-action@v2` 실행, 실패 시 머지 차단), `.gitleaks.toml`
      (기본 룰셋 확장 + 합성 테스트 데이터 경로만 좁게 허용목록), `scripts/
      self-detect.sh`(로컬에서 CI와 같은 설정으로 미리 실행) 3개 신설. 로컬에서
      실제 실행해 깨끗한 스캔 확인(두 모드 — 전체 git 히스토리 스캔 및
      `--no-git` 워킹트리 스캔 모두).
- [x] `eval/nfr-dq/` 평가셋을 수백 건 규모로 확장하고, 확장된 Go 탐지기를 다시
      돌려 NFR-DQ-01 기준을 재확인한다. **완료(2026-10-25)**:
      `internal/detect/benchmark_test.go`에 고정 시드(결정론적, CI에서 매번
      동일) 생성기로 470건(양성 360/음성 110) 평가셋을 실제 코드로 만들고,
      실제 배포되는 `detect.ChecksumRecognizer`(Go, Presidio 파이썬 스파이크가
      아니라 Cardea가 실제로 선적하는 코드)로 돌렸다. 결과: 재현율 1.0000,
      정밀도 0.9945 — NFR-DQ-01 목표(99%/95%) 충족, 1차 파이썬 스파이크(46건)
      보다 10배 큰 규모에서도 유지됨. 운송장번호류 디스트랙터 20건 중 2건이
      우연히 RRN 체크섬을 통과하는 거짓양성을 재현 — 1차 스파이크에서 발견한
      "체크섬 단독 탐지의 구조적 한계"가 큰 표본에서도 똑같이(~1/11 확률대로)
      나타남을 확인, 문맥 가중치 보강이 필요하다는 결론이 더 단단해짐. 이
      테스트는 CI에서 매 빌드마다 자동으로 재확인된다(일회성 벤치마크가 아님).

## 다음 단계 — 실제 격차로 드러난 후속 작업

- [ ] 한국어 민감정보 탐지(FR-PI 로케일 플러그인 핵심 가치)를 Presidio에 붙인다
      — 공식 Docker 이미지는 영어 NLP 엔진만 탑재돼 있어, 별도 한국어 NLP
      엔진(spaCy 한국어 모델 또는 다국어 transformer)을 Presidio
      `NlpEngineProvider` 설정으로 구성해야 한다(`internal/presidio` README 참고).
- [ ] 체크섬 단독 탐지의 거짓양성 구조적 한계(운송장번호 등이 ~1/11 확률로
      우연히 RRN 체크섬 통과)를 Presidio `ContextAwareEnhancer` 패턴처럼
      문맥 단어 가중치로 보강한다 — `internal/detect`/`eval/nfr-dq` 양쪽에서
      반복 확인된 이슈라 우선순위를 올린다.
- [ ] ADR-0006 SDK 어댑터 계층(OpenAI Chat Completions·Responses·Embeddings,
      Anthropic Messages 프로토콜 변환)을 구현한다 — 지금 `BifrostForwarder`는
      "provider/model/messages" 평문 포맷 하나만 받는 자리값이다.
- [ ] 실제 프로바이더 API 키가 있는 환경에서 Bifrost 경유 진짜 모델 호출까지
      end-to-end 재검증한다(이 작업 환경엔 키가 없어 "설정 누락 에러까지"만
      확인했다).
- [ ] MCP 1차 인증/allowlist를 Cardea 코어가 직접 구현한다(Bifrost가 제공한다는
      전제를 폐기했으므로, AG-02/03 범위가 됨).
- [ ] FR-PI-04 토큰 금고 메타데이터(scope/만료)를 영속화한다(지금은 Cardea
      프로세스 메모리에만 있어 재시작 시 사라진다 — OpenBao KV 등 검토).

## 보류 중인 기존 미결 사항 (참고)

- `docs/requirements.md` 9장 미결 사항 목록(PL-06 경계, GW-05/08 설계 디테일, 시맨틱
  라우팅 지연 목표, Bifrost 라이선스 경계 재확인, OpenBao 성숙도 추적, TinyGo 컴파일
  스파이크 등) — 구현 착수 시 모듈별로 순서를 정해 처리한다.
