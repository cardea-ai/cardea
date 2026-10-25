#!/usr/bin/env bash
# NFR-SS-05 자가 탐지 로컬 실행 스크립트.
#
# CI(.github/workflows/gitleaks.yml)와 같은 설정(.gitleaks.toml)으로 로컬에서
# 미리 돌려볼 수 있게 한다. gitleaks가 PATH에 없으면 설치 방법을 안내하고
# 종료한다(설치를 대신 해주지 않는다 — 임의 다운로드/설치는 사용자가 결정).
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v gitleaks >/dev/null 2>&1; then
  echo "gitleaks가 설치돼 있지 않습니다. 설치 방법:" >&2
  echo "  go install github.com/zricethezav/gitleaks/v8@latest" >&2
  echo "  (또는 https://github.com/gitleaks/gitleaks/releases 에서 바이너리 다운로드)" >&2
  exit 1
fi

gitleaks detect --source . --config .gitleaks.toml "$@"
