#!/usr/bin/env bash
# firstrun.sh COMMAND...
#
# A newcomer's first run, on a real machine: a home folder with nothing of
# Sameway's in it, the program started as a double-click starts it (with
# nothing after it), and then what the person would see: their workspace
# made in Documents, Sameway answering, the welcome on its first page, and
# Quit Sameway stopping it. Used by .github/workflows/smoke.yml on Linux,
# macOS and Windows, with the downloads as a release packs them.
set -euo pipefail
home="$(mktemp -d)"
mkdir -p "$home/Documents"
export HOME="$home" USERPROFILE="$home" XDG_CONFIG_HOME="$home/.config" XDG_DATA_HOME="$home/.local/share"
export SAMEWAY_KNOWN="$home/known.json" BROWSER=true
if [ -n "${LOCALAPPDATA:-}" ]; then
  export APPDATA="$home/AppData/Roaming" LOCALAPPDATA="$home/AppData/Local"
  mkdir -p "$APPDATA" "$LOCALAPPDATA"
fi
log="$home/run.log"
"$@" </dev/null >"$log" 2>&1 &
started=$!
addr=""
for _ in $(seq 1 60); do
  sleep 1
  if [ -f "$SAMEWAY_KNOWN" ]; then
    addr=$(grep -o '"addr": *"[^"]*"' "$SAMEWAY_KNOWN" | head -1 | sed 's/.*"\([^"]*\)"$/\1/')
    if [ -n "$addr" ] && curl -sf "http://$addr/api/describe" >/dev/null; then break; fi
  fi
done
echo "--- what the window said"; cat "$log" || true
[ -n "$addr" ] || { echo "Sameway never answered"; exit 1; }
test -f "$home/Documents/Sameway/workspace.yaml" || { echo "no workspace in Documents"; exit 1; }
page=$(curl -sf "http://$addr/")
echo "$page" | grep -q "Welcome to Sameway" || { echo "the first page has no welcome"; exit 1; }
curl -sf "http://$addr/favicon.svg" >/dev/null || { echo "no icon"; exit 1; }
echo "Sameway answered at $addr with the welcome"
curl -sf -X POST -H "Origin: http://$addr" "http://$addr/quit" >/dev/null || true
for _ in $(seq 1 15); do
  sleep 1
  curl -sf "http://$addr/api/describe" >/dev/null || { echo "Quit Sameway stopped it"; kill "$started" 2>/dev/null || true; exit 0; }
done
echo "Sameway did not stop when asked"; exit 1
