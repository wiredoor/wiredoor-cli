#!/bin/sh

set -e

REPO="wiredoor/wiredoor-cli"

# Get latest release from GitHub API
get_latest_version() {
  curl --silent "https://api.github.com/repos/$REPO/releases/latest" | 
    grep '"tag_name":' | 
    sed -E 's/.*"v?([^"]+)".*/\1/'
}

VERSION=$(get_latest_version)

detect_arch() {
  ARCH=$(uname -m)
  case "$ARCH" in
    x86_64) echo "amd64" ;;
    aarch64 | arm64) echo "arm64" ;;
    *) echo "unsupported" ;;
  esac
}

detect_os() {
  if [ -f /etc/os-release ]; then
    . /etc/os-release
    case "$ID" in
      debian|ubuntu|raspbian)
        echo "debian"
        ;;
      fedora|centos|almalinux|rhel)
        echo "rhel"
        ;;
      alpine)
        echo "alpine"
        ;;
      arch|manjaro)
        echo "archlinux"
        ;;
      *)
        echo "unsupported"
        ;;
    esac
  else
    echo "unsupported"
  fi
}

ARCH=$(detect_arch)
OS=$(detect_os)

if [ "$OS" = "unsupported" ]; then
  echo "❌ Unsupported OS=$OS $ID"
  exit 1
fi

if [ "$ARCH" = "unsupported" ]; then
  echo "❌ Unsupported ARCH=$ARCH"
  exit 1
fi

if [ "$(id -u)" -eq 0 ]; then
  SUDO=""
else
  SUDO="sudo"
fi

echo "🔍 Detected OS: $OS, ARCH: $ARCH"
echo "📦 Installing Wiredoor CLI v$VERSION..."

case "$OS" in
  debian)
    URL="https://github.com/$REPO/releases/download/v$VERSION/wiredoor_${VERSION}-1_debian_${ARCH}.deb"
    curl -fsSL "$URL" -o /tmp/wiredoor.deb
    $SUDO apt install -f /tmp/wiredoor.deb
    rm -f /tmp/wiredoor.deb
    ;;
  rhel)
    URL="https://github.com/$REPO/releases/download/v$VERSION/wiredoor_${VERSION}-1_rpm_${ARCH}.rpm"
    curl -fsSL "$URL" -o /tmp/wiredoor.rpm
    $SUDO dnf install -y /tmp/wiredoor.rpm || $SUDO yum install -y /tmp/wiredoor.rpm
    rm -f /tmp/wiredoor.rpm
    ;;
  alpine)
    URL="https://github.com/$REPO/releases/download/v$VERSION/wiredoor_${VERSION}-1_alpine_${ARCH}.apk"
    curl -fsSL "$URL" -o /tmp/wiredoor.apk
    $SUDO apk add --allow-untrusted /tmp/wiredoor.apk
    rm -f /tmp/wiredoor.apk
    ;;
  archlinux)
    if ! command -v iptables >/dev/null 2>&1; then
      echo "[wiredoor] 'iptables' command not found."
      echo "Install either 'iptables-nft' or 'iptables' and run the installer again."
      exit 1
    fi
    URL="https://github.com/$REPO/releases/download/v$VERSION/wiredoor_${VERSION}-1_archlinux_${ARCH}.pkg.tar.zst"
    curl -fsSL "$URL" -o /tmp/wiredoor.pkg.tar.zst
    DEPS=$(tar -xOf /tmp/wiredoor.pkg.tar.zst .PKGINFO | grep '^depend =' | cut -d= -f2- | xargs)
    $SUDO pacman -Sy --needed $DEPS
    $SUDO pacman -U --noconfirm /tmp/wiredoor.pkg.tar.zst
    rm -f /tmp/wiredoor.pkg.tar.zst
    ;;
esac

echo "✅ Wiredoor CLI installed successfully! Type wiredoor for help."
