# Cardea 게이트웨이 최소 골격 + 선행 검증 4건

## 요약

`docs/requirements.md` 1장 "최종 방향 결정(확정)"에 따라 Build 지속(조건부)을
택한 뒤, 구현 착수 전 선행 조건 4개를 처리하고 처음으로 실제 Go 코드를 작성한
PR이다. 문서(포지셔닝·경쟁분석·요구사항 명세·온보딩 가이드) → CI 거버넌스
(gitleaks 자가탐지, 커밋 템플릿) → 탐지 로직 검증 스파이크(Python) → 게이트웨이
최소 골격(Go) 순서로 4개 커밋에 나눴다.

핵심 원칙(CLAUDE.md)을 그대로 지켰다: 원문 미저장, fail-closed 기본값, 등급은
내려가지 않음, 판정→전달 순서 고정. 전부 **실제 실행해서 검증**했고, 환경
제약(프로바이더 API 키 없음)으로 확인 못 한 부분은 솔직하게 "남은 일"로
남겼다 — 가짜 성공을 만들지 않았다.

## 커밋 구성

1. **📝 docs** — 요구사항 명세(`docs/requirements.md`, ADR-0001~0008),
   경쟁분석·포지셔닝(`docs/COMPETITIVE_ANALYSIS.md`, `docs/POSITIONING.md`),
   오픈코어 에디션 경계(`docs/EDITIONS.md`), v0.1 온보딩 가이드
   (`docs/GETTING_STARTED.md`), README 라이선스 공약 절, 각 문서 변경이력
   (`docs/history/`)
2. **👷 ci** — gitleaks 자가탐지 워크플로(NFR-SS-05), gitmoji 커밋 템플릿,
   CODEOWNERS/PR 템플릿
3. **🧪 eval** — NFR-DQ-02 체크섬 탐지 검증 스파이크(Python, `eval/nfr-dq/`)
4. **✨ feat** — 게이트웨이 최소 골격(Go, `cmd/`, `internal/`)

## 구현 범위 (4번 커밋)

| 패키지 | 내용 |
|---|---|
| `internal/plugin` | ADR-0002 6종 플러그인 슬롯 인터페이스(정규화·탐지·분류·정책·처리·연동) |
| `internal/detect` | FR-PI-01/08/11 체크섬 식별번호 탐지(KR_RRN/KR_CARD_NUMBER/KR_BIZ_REG_NO) + 정규화. eval 스파이크를 Go로 포팅 |
| `internal/policy` | OPA/Rego 기반 정책 평가, NFR-AV-02 fail-closed(평가 실패 시 무조건 deny) |
| `internal/vault` | FR-PI-04 토큰 금고 — `MemoryVault`(AES-256-GCM, 기본값)와 `OpenBaoVault`(ADR-0007, 실 OpenBao Transit 연동) |
| `internal/processor` | 토큰 발급/복원(Mask/Restore) — CSPRNG 토큰, 스코프 격리, 참조 일관성 |
| `internal/presidio` | Microsoft Presidio REST 연동(체크섬 없는 엔티티용) |
| `internal/forward` | ADR-0001 Forwarder 인터페이스. `StubForwarder`(기본값, 솔직한 502)와 `BifrostForwarder`(Bifrost core Go SDK, 옵트인) |
| `internal/gateway`, `cmd/cardea` | GW-01 듀얼 프로토콜(OpenAI/Anthropic 엔드포인트 경로) HTTP 게이트웨이, 127.0.0.1 바인딩 |

처리 순서는 **정규화 → 탐지 → 정책 → 마스킹 → 전달**로 코드 구조에 고정돼
있다 — `partial` 판정에서 평문이 Forwarder까지 그대로 가는 경로가 없다
(`internal/gateway/gateway.go`의 `handle` 참고).

## 실제로 검증한 것 (가짜 결과 아님)

- **NFR-DQ-02**: Python 스파이크 46건 → Go 포팅 후 고정 시드 470건으로 확장,
  재현율 1.0000/정밀도 0.9945(NFR-DQ-01 목표 99%/95% 충족). 운송장번호류가
  ~1/11 확률로 RRN 체크섬을 우연히 통과하는 구조적 한계를 양쪽에서 재현 —
  문맥 가중치 보강 필요성을 실측으로 확인.
- **OpenBao**: 실제 docker 컨테이너(`openbao/openbao:latest server -dev`)를
  띄워 Transit 엔진 암복호화를 curl로 먼저 확인한 뒤 Go 클라이언트로 재현.
  통합 테스트 4건 실서버 대상 PASS(라운드트립, 스코프 격리, 만료 fail-closed,
  "로컬엔 ciphertext만, 키는 없음" 불변식).
- **Presidio**: 실제 `mcr.microsoft.com/presidio-analyzer` 컨테이너로 PERSON/
  EMAIL_ADDRESS 탐지, 인덱스 정합성, 한국어 미지원(공식 이미지는 영어 NLP
  엔진만 탑재) 3건 PASS. **이 과정에서 ADR-0007의 "gRPC" 기재가 틀렸다는 걸
  확인** — 실제 인터페이스는 REST(Flask `/analyze`)뿐이라 8장을 정정했다.
- **Bifrost core**: `go get github.com/maximhq/bifrost/core`(Apache-2.0)로
  실제 SDK 초기화 + `ChatCompletionRequest` 호출 경로 테스트. 이 환경엔
  프로바이더 API 키가 없어 "설정 누락 구조화 에러"까지만 확인했다(실제 모델
  응답은 미검증 — 비밀값을 임의로 만들어 넣지 않았다).
  **임베딩 전 재확인 중 발견**: Bifrost 공식 자료 자체가 모순(README "Key
  Features"는 MCP를 OSS처럼 적고, "Enterprise Deployments"는 "MCP gateway"를
  엔터프라이즈 전용으로 명시) → ADR-0007의 "Bifrost가 MCP allowlist 제공"
  전제를 폐기, MCP 1차 인증/allowlist는 Cardea가 직접 구현하는 쪽으로 범위
  수정.
- **gitleaks**: 실제 설치해 저장소 스캔(0건, 깨끗함). 두 모드(전체 git
  히스토리/워킹트리) 모두 확인.
- 전체 `go build`/`go vet`/`go test ./...` 그린(캐시 아닌 상태로도 확인).

## 요구사항 추적

Implements FR-PI-01, FR-PI-04, FR-PI-08, FR-PI-11, FR-AC-03, NFR-AV-02,
NFR-DQ-01, NFR-DQ-02, NFR-SS-05, GW-01

Ref: ADR-0001, ADR-0002, ADR-0006, ADR-0007(일부 전제 수정), ADR-0008

## 위협 모델 체크리스트 (CLAUDE.md 5장)

- **내부자 실수/악의적 내부자**: 토큰 금고는 CSPRNG 난수 + 암호화 매핑만 쓰고
  (해시 아님), 감사 로그엔 토큰·타입·규칙ID만 남기고 원문/ciphertext는 안
  남김(`internal/vault`, `internal/processor` 테스트로 검증).
- **공급망 공격**: gitleaks CI가 매 PR마다 돌아 시크릿 유출을 1차 방어.
- **Cardea 자체 공격(게이트웨이 장악)**: 정책/탐지/토큰 금고 초기화 실패 시
  기동 자체를 막는 fail-closed 기본값(`NFR-AV-02`) — "일단 허용" 경로 없음.
- **해당 없음/이번 PR 범위 밖**: 프롬프트 인젝션, 악성 MCP 서버, 메모리·스킬
  오염 — 아직 MCP 1차 구현 전이라 다음 PR에서 다룬다.

## 남은 일 (후속 PR)

- 한국어 민감정보 탐지(Presidio NLP 엔진 구성) — 지금 Presidio 경로는 영어만
- 체크섬 단독 탐지의 거짓양성 보강(문맥 가중치, `ContextAwareEnhancer` 패턴)
- ADR-0006 SDK 어댑터 계층(OpenAI/Anthropic 프로토콜 정식 변환)
- 실제 프로바이더 키로 Bifrost 경유 end-to-end 재검증
- MCP 1차 인증/allowlist 자체 구현(AG-02/03)
- FR-PI-04 토큰 금고 메타데이터 영속화(현재 프로세스 메모리 한정)

상세 내역과 각 항목의 완료/남은 일 구분은 `TODO.md`, 날짜별 변경 이유는
`docs/history/requirements.md` 참고.

## 리뷰/로컬 실행

```
go build ./... && go vet ./... && go test ./...
```

OpenBao/Presidio 통합 테스트는 실제 컨테이너가 있어야 돈다(없으면 자동 skip):

```
docker run --rm -p 8300:8200 -e BAO_DEV_ROOT_TOKEN_ID=dev openbao/openbao:latest server -dev
docker run --rm -p 8301:3000 mcr.microsoft.com/presidio-analyzer:latest
CARDEA_OPENBAO_TEST_ADDR=http://127.0.0.1:8300 CARDEA_OPENBAO_TEST_TOKEN=dev \
CARDEA_PRESIDIO_TEST_ADDR=http://127.0.0.1:8301 go test ./...
```
