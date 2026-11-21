#!/usr/bin/env bash
# Rebuild the design canvas's frames from the page as built
# (docs/design/canvas, AGENTS.md "Design Canvas"). A frame is never drawn by
# hand: it holds the page's own markup, captured from a server with the demo
# set, and the rules of pkg/ui/static/ui.css that the markup uses. After a
# change to the stylesheet or the templates, run this, look at `git diff`
# and the screenshots, then publish the changed files to the canvas.
#
#   scripts/canvas.sh [<frame>...]     # Main Materials Diagram Fitment Controls States
#   scripts/canvas.sh --check [<frame>...]
#
# --check writes no frame. It builds each one, compares it with the committed
# file, and exits 1 naming those that differ, each left in build/canvas as
# built. A frame's height, the data month and the release's hash are left out
# of the comparison: fonts and go.mod's pin set them, not the page.
# canvas.yml runs it on a pull request that touches the page.
#
# It builds the Service Worker (scripts/sw.sh), starts two servers, one with
# the demo set (pkg/ui/demo) and one with no vocabulary, and drives headless
# Chrome over them (NITROVIN_CHROME overrides its path). It writes:
#
#   docs/design/canvas/project/<frame>[-Dark].dc.html   the frames
#   docs/design/canvas/project/canvas.json              their heights, and the dark row's place
#   build/canvas/<frame>[-Dark].{html,png}              a preview and a screenshot of each
#
# It does not publish. Publishing follows AGENTS.md's loop: read the canvas
# down first, never force. The two uploaded images' addresses are fixed in
# scripts/canvas/engine.mjs and resolve only on the maintainer's canvas.
#
# NITROVIN_CANVAS_PORT (default 8791) is the demo server's port; the bare
# server takes the next but one. Needs node (mise) and Chrome.
set -euo pipefail
cd "$(dirname "$0")/.."
port="${NITROVIN_CANVAS_PORT:-8791}"
bare=$((port + 2))
scripts/sw.sh
bin="$(mktemp -d)/nitrovin"
# -buildvcs=false: the page prints its version, "(devel)" on every checkout.
go build -buildvcs=false -o "$bin" ./cmd/nitrovin
"$bin" --vocab pkg/ui/demo serve --addr "127.0.0.1:$port" >/dev/null 2>&1 &
demo_pid=$!
"$bin" serve --addr "127.0.0.1:$bare" >/dev/null 2>&1 &
bare_pid=$!
trap 'kill "$demo_pid" "$bare_pid" 2>/dev/null || true' EXIT
for p in "$port" "$bare"; do
  for _ in $(seq 50); do curl -fsS -o /dev/null 2>/dev/null "http://127.0.0.1:$p/" && break; sleep 0.2; done
done
# A port already taken answers too, with some other server's page: the
# frames are this checkout's only if both of its servers are still up.
if ! kill -0 "$demo_pid" 2>/dev/null || ! kill -0 "$bare_pid" 2>/dev/null; then
  echo "canvas: a server did not start: is port $port or $bare in use? Set NITROVIN_CANVAS_PORT" >&2
  exit 1
fi
NITROVIN_CANVAS_DEMO="http://127.0.0.1:$port" NITROVIN_CANVAS_BARE="http://127.0.0.1:$bare" \
  node scripts/canvas/build.mjs "$@"
