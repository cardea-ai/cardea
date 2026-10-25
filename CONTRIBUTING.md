# Contributing to Cardea

## 커밋 메시지 규칙

커밋 메시지는 [gitmoji](https://gitmoji.dev) 기반으로 작성한다.

```
<gitmoji> <영역>: <한 줄 요약, 명령문·현재시제>

<본문 — 무엇을·왜. 비워 둘 수 있음>

Implements <요구사항 ID>   (예: FR-PI-01, AG-04 — 여러 개면 전부 나열)
Ref: <관련 ADR>            (있으면)
```

로컬에 템플릿을 적용하려면(최초 1회):

```
git config commit.template .gitmessage
```

규칙:

- 영역은 요구사항 ID 접두어(FR-PI, AG, GW, policy, gateway 등)나 디렉터리명으로
  짧게 적는다.
- 요구사항 ID(FR-xx-NN, AG-NN, PL-NN, NFR-xx-NN)가 있는 작업은 본문에
  `Implements <ID>`를 반드시 적는다(CLAUDE.md 2장 추적성 규칙). ID가 없는 새
  작업이면 커밋 전에 ID를 먼저 정한다.
- 깃모지 전체 목록은 `.gitmessage` 파일 하단 치트시트 또는 https://gitmoji.dev
  참고. 자주 쓰는 것: ✨ 기능, 🐛 버그수정, 🔒️ 보안, ♻️ 리팩터링, 📝 문서,
  ✅ 테스트, 👷 CI, 🏗️ 아키텍처(ADR급).
- 보안 기본값(fail-closed, 원문 미저장, 서명 검증 등)에 영향을 주는 커밋은
  본문에 `Security-Note:`로 한 줄 짚어준다.
