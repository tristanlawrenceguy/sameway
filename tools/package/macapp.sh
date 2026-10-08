#!/usr/bin/env bash
# macapp.sh BINARY VERSION OUT.app
#
# Makes the Mac app a person opens with a double-click from the sameway
# program for one processor: the program, its icon, an Info.plist that
# keeps it out of the Dock (it runs in the background; its page is the
# window), all signed as a bundle, ad hoc. The release builds it on Linux
# with rcodesign; on a Mac, Apple's own codesign does the same. The smoke
# test (.github/workflows/smoke.yml) builds it here and checks it with
# Apple's tools, so what a person downloads is what was tested.
set -euo pipefail
bin="$1"; version="$2"; app="$3"
root="$(cd "$(dirname "$0")/../.." && pwd)"
rm -rf "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$bin" "$app/Contents/MacOS/sameway"
chmod +x "$app/Contents/MacOS/sameway"
cp "$root/design/brand/icon.icns" "$app/Contents/Resources/icon.icns"
cat > "$app/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>CFBundleName</key><string>Sameway</string>
<key>CFBundleIdentifier</key><string>io.github.tristanlawrenceguy.sameway</string>
<key>CFBundleExecutable</key><string>sameway</string>
<key>CFBundleIconFile</key><string>icon</string>
<key>CFBundlePackageType</key><string>APPL</string>
<key>CFBundleShortVersionString</key><string>${version}</string>
<key>LSMinimumSystemVersion</key><string>11.0</string>
<key>LSUIElement</key><true/>
</dict></plist>
PLIST
if command -v rcodesign >/dev/null; then
  rcodesign sign "$app"
elif command -v codesign >/dev/null; then
  codesign --force --deep --sign - "$app"
else
  echo "macapp.sh: no rcodesign or codesign to sign $app" >&2
  exit 1
fi
