set -e

REPO="sviridovkonstantin42/godsl"
BINARY_NAME="godsl"
INSTALL_DIR="/usr/local/bin"

LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | cut -d'"' -f4)
VERSION="${LATEST_VERSION#v}"

OS=$(uname -s)
ARCH=$(uname -m)

if [[ "$OS" == "Linux" ]]; then
    PLATFORM="linux"
elif [[ "$OS" == "Darwin" ]]; then
    PLATFORM="macOS"
else
    echo "❌ Unsupported OS: $OS"
    exit 1
fi

if [[ "$ARCH" == "x86_64" ]]; then
    ARCH_TYPE="x86_64"
elif [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
    ARCH_TYPE="arm64"
else
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
fi

FILENAME="${BINARY_NAME}_${VERSION}_${PLATFORM}_${ARCH_TYPE}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${FILENAME}"

echo "📦 Downloading $FILENAME from $URL..."
curl -L "$URL" -o "$FILENAME"

echo "📂 Extracting..."
tar -xzf "$FILENAME"

chmod +x "$BINARY_NAME"
sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"

rm "$FILENAME"

echo "✅ Installed $BINARY_NAME to $INSTALL_DIR"
echo "📦 Version: $LATEST_VERSION"