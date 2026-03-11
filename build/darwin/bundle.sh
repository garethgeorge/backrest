#!/bin/bash
# bundle.sh - Assembles the macOS .app bundle for Backrest tray mode.
#
# Usage:
#   ./build/darwin/bundle.sh [binary_path] [version]
#
# Arguments:
#   binary_path  Path to the compiled backrest binary (default: ./backrest)
#   version      Version string for the bundle (default: "unknown")
#
# Output:
#   Backrest.app/ in the current directory

set -euo pipefail

BINARY="${1:-./backrest}"
VERSION="${2:-unknown}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
APP_DIR="Backrest.app"

if [ ! -f "$BINARY" ]; then
    echo "Error: binary not found at $BINARY"
    echo "Build it first: go build -o backrest ./cmd/backrest/"
    exit 1
fi

# Clean previous bundle
rm -rf "$APP_DIR"

# Create bundle structure
mkdir -p "$APP_DIR/Contents/MacOS"
mkdir -p "$APP_DIR/Contents/Resources"

# Copy binary
cp "$BINARY" "$APP_DIR/Contents/MacOS/backrest"
chmod +x "$APP_DIR/Contents/MacOS/backrest"

# Create a launcher wrapper that passes --tray by default
cat > "$APP_DIR/Contents/MacOS/backrest-launcher" << 'LAUNCHER'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$DIR/backrest" --tray "$@"
LAUNCHER
chmod +x "$APP_DIR/Contents/MacOS/backrest-launcher"

# Generate Info.plist with version
sed "s|__VERSION__|$VERSION|g" "$SCRIPT_DIR/Info.plist" > "$APP_DIR/Contents/Info.plist"

# Generate .icns from the PNG icon
ICON_PNG="$SCRIPT_DIR/icon.png"
if [ -f "$ICON_PNG" ]; then
    ICONSET_DIR=$(mktemp -d)/backrest.iconset
    mkdir -p "$ICONSET_DIR"

    # Generate required icon sizes
    for size in 16 32 64 128 256 512; do
        sips -z $size $size "$ICON_PNG" --out "$ICONSET_DIR/icon_${size}x${size}.png" >/dev/null 2>&1
    done
    # Retina variants
    for size in 16 32 128 256 512; do
        double=$((size * 2))
        sips -z $double $double "$ICON_PNG" --out "$ICONSET_DIR/icon_${size}x${size}@2x.png" >/dev/null 2>&1
    done

    iconutil -c icns -o "$APP_DIR/Contents/Resources/backrest.icns" "$ICONSET_DIR" 2>/dev/null || {
        echo "Warning: iconutil failed, bundling without .icns icon"
    }
    rm -rf "$(dirname "$ICONSET_DIR")"
else
    echo "Warning: icon.png not found at $ICON_PNG, bundling without icon"
fi

echo "Created $APP_DIR (version: $VERSION)"
echo "To run: open $APP_DIR"
