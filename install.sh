#!/bin/sh
# parley installer: downloads the right prebuilt binary from GitHub Releases and
# drops it on your PATH. macOS and Linux only (use `go install` elsewhere).
#
#   curl -fsSL https://raw.githubusercontent.com/dhitalkamal/parley/main/install.sh | sh
#
# Overrides (env vars):
#   PARLEY_VERSION=v0.2.0     pin a specific release instead of the latest
#   PARLEY_BIN_DIR=/some/bin  install into a specific directory
set -e

REPO="dhitalkamal/parley"
BINARY="parley"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$arch" in
	x86_64 | amd64) arch="amd64" ;;
	arm64 | aarch64) arch="arm64" ;;
	*) echo "parley: unsupported architecture: $arch" >&2 && exit 1 ;;
esac
case "$os" in
	darwin | linux) ;;
	*)
		echo "parley: no prebuilt binary for $os - install with: go install github.com/${REPO}/cmd/parley@latest" >&2
		exit 1
		;;
esac

version="${PARLEY_VERSION:-}"
if [ -z "$version" ]; then
	version=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
		grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/')
fi
if [ -z "$version" ]; then
	echo "parley: could not determine the latest release (set PARLEY_VERSION to pin one)" >&2
	exit 1
fi

asset="${BINARY}_${os}_${arch}.tar.gz"
url="https://github.com/${REPO}/releases/download/${version}/${asset}"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

echo "parley: downloading ${version} (${os}/${arch})..."
# --progress-bar shows a live download bar (the asset is a few MB, so a silent
# curl -s looked like a hang on slow links). -f fails on http errors, -L
# follows the release redirect to the asset CDN. Fall back to wget if curl is
# absent.
if command -v curl >/dev/null 2>&1; then
	curl -fL --progress-bar "$url" -o "$tmp/$asset"
elif command -v wget >/dev/null 2>&1; then
	wget --show-progress -qO "$tmp/$asset" "$url"
else
	echo "parley: need curl or wget to download" >&2
	exit 1
fi
tar -C "$tmp" -xzf "$tmp/$asset"

# pick an install dir: first writable of the usual spots, else ~/.local/bin.
bindir="${PARLEY_BIN_DIR:-}"
if [ -z "$bindir" ]; then
	for d in /usr/local/bin "$HOME/.local/bin"; do
		if [ -d "$d" ] && [ -w "$d" ]; then
			bindir="$d"
			break
		fi
	done
fi
[ -z "$bindir" ] && bindir="$HOME/.local/bin"
# always ensure the target exists - a PARLEY_BIN_DIR pointing at a missing
# directory used to make the install silently fail while still printing
# "installed". mkdir -p failing (e.g. not writable) exits here via set -e.
mkdir -p "$bindir"

install -m 0755 "$tmp/$BINARY" "$bindir/$BINARY" 2>/dev/null ||
	{ mv "$tmp/$BINARY" "$bindir/$BINARY" && chmod 0755 "$bindir/$BINARY"; }

echo "parley: installed to $bindir/$BINARY"
case ":$PATH:" in
	*":$bindir:"*) ;;
	*)
		echo "parley: note - $bindir is not on your PATH. Add it, e.g.:"
		echo "  echo 'export PATH=\"$bindir:\$PATH\"' >> ~/.zshrc"
		;;
esac
echo "parley: done - run 'parley' to start."
