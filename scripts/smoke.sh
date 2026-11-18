#!/bin/sh
# Runs a built kongkow through its main commands in a new, empty data
# folder, and prints what each one says. Usage: scripts/smoke.sh ./bin/kongkow
set -eu

bin=$1
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export KONGKOW_HOME="$work/home" KONGKOW_OUTPUT=json

step() {
  echo "== kongkow $*"
  "$bin" "$@"
  echo
}

# a tiny PNG header is enough for the image checks
printf '\211PNG\r\n\032\n\000\000\000\rIHDR' >"$work/pic.png"
mkdir -p "$work/out"
past=$(date -u -d '-1 hour' '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || date -u -v-1H '+%Y-%m-%dT%H:%M:%SZ')

step version
"$bin" guide >/dev/null && echo "== kongkow guide: ok" && echo
step doctor
step dest add out --kind file --target "$work/out" --slots 09:00,18:00
step dest list
step dest test out
step post draft hello --title Hi --body 'Hello from the smoke test' --media "$work/pic.png" --alt 'A test image'
step post edit hello --body 'Hello again'
step post publish hello --to out --dry-run
step post publish hello --to out
step post queue hello --to out
step post schedule hello --to out --at "$past"
step post status --run
step audit --limit 3

test "$(ls "$work/out" | wc -l)" -eq 2 || { echo "want 2 published files in $work/out"; exit 1; }
echo "smoke: all commands ran"
