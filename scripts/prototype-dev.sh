#!/usr/bin/env bash
# PROTOTYPE (branch prototype/pare-back-ui), not for main: the dev loop for
# the pared-back page. Builds the browser's Service Worker module and
# fetches EPA's vocabulary once (the page's local data, as a release build
# has them), then builds and serves nitrovin, and rebuilds and restarts
# whenever a source of the pages changes.
#
#   scripts/prototype-dev.sh [ADDR]        (default 127.0.0.1:8137)
#
# Every build serves under a release of its own (NITROVIN_DEV_VERSION, read
# by pkg/nitrovin's Version in this prototype only): the assets are
# immutable for a year under their release and the Service Worker caches
# them by it, so a reload would otherwise show the old stylesheet.
#
# The server loads the demo set and EPA's vocabulary itself, so the page
# has names to show after every rebuild without the browser's own files,
# which a new release pauses until its Service Worker updates.
set -uo pipefail
cd "$(dirname "$0")/.."
addr="${1:-127.0.0.1:8137}"
out=$(mktemp -d)
pid=""
trap '[[ -n $pid ]] && kill "$pid" 2>/dev/null; rm -rf "$out"' EXIT

[[ -f cmd/nitrovin/sw/epa-vehicles.jsonl ]] || scripts/epa.sh || echo "prototype-dev: no EPA vocabulary (offline?): the dialog offers none"

changed() {
  find pkg cmd -type f \( -name '*.go' -o -name '*.html' -o -name '*.css' -o -name '*.js' -o -name '*.json' -o -name '*.jsonl' \) \
    -not -path 'cmd/nitrovin/sw/*' -not -name '*_test.go' -not -path '*/testdata/*' -newer "$out/built" 2>/dev/null | head -1
}

while :; do
  touch "$out/built"
  if scripts/sw.sh && go build -o "$out/nitrovin" ./cmd/nitrovin; then
    vocab=(--vocab pkg/ui/demo)
    [[ -f cmd/nitrovin/sw/epa-vehicles.jsonl ]] && vocab+=(--vocab cmd/nitrovin/sw/epa-vehicles.jsonl)
    NITROVIN_DEV_VERSION="dev-$(date +%H%M%S)" "$out/nitrovin" "${vocab[@]}" serve --addr "$addr" &
    pid=$!
    echo "prototype-dev: http://$addr/ (pid $pid)"
  else
    echo "prototype-dev: build failed; waiting for a change"
  fi
  until [[ -n $(changed) ]]; do sleep 1; done
  echo "prototype-dev: $(changed) changed; rebuilding"
  [[ -n $pid ]] && kill "$pid" 2>/dev/null && wait "$pid" 2>/dev/null
  pid=""
done
