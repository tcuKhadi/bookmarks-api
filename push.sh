#!/usr/bin/env bash
set -euo pipefail

if [ $# -lt 1 ] || [ -z "$1" ]; then
  echo "Usage: ./push.sh \"commit message\""
  exit 1
fi
msg="$1"

run() {
  local label="$1"
  shift
  printf "%-10s" "$label"
  if out="$("$@" 2>&1)"; then
    echo "ok"
  else
    echo "failed"
    echo "$out"
    exit 1
  fi
}

printf "%-10s" "Format"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
  echo "failed"
  echo "$unformatted"
  exit 1
fi
echo "ok"

run "Vet" go vet ./...
run "Test" go test -race ./...

git add -A
if git diff --cached --quiet; then
  echo "Nothing to commit."
  exit 0
fi

run "Commit" git commit -m "$msg"
run "Push" git push origin "$(git branch --show-current)"
