#!/bin/bash

set -e

export OWNER=codeperfio
export REPO=codeperf

TAR_FILE="/tmp/$REPO.tar.gz"
RELEASES_URL="https://github.com/$OWNER/$REPO/releases"
test -z "$TMPDIR" && TMPDIR="$(mktemp -d)"

hasCli() {

    hasCurl=$(which curl)
    if [ "$?" = "1" ]; then
        echo "You need curl to use this script."
        exit 1
    fi
}

last_version() {
  curl -sL -o /dev/null -w %{url_effective} "$RELEASES_URL/latest" | 
    rev | 
    cut -f1 -d'/'| 
    rev
}

download() {
echo "getting latest release from $RELEASES_URL/latest..."
echo "detected latest version=$(last_version)"
  test -z "$VERSION" && VERSION="$(last_version)"
  test -z "$VERSION" && {
    echo "Unable to get codeperf version." >&2
    exit 1
  }
  rm -f "$TAR_FILE"
  RELEASE_ARTIFACT="$RELEASES_URL/download/$VERSION/${REPO}_$(uname -s)_$(uname -m).tar.gz"
  echo "getting $RELEASE_ARTIFACT into $TAR_FILE"
  curl -s -L -o "$TAR_FILE" \
    $RELEASE_ARTIFACT
    
}

hasCli
download
tar -xf "$TAR_FILE" -C "$TMPDIR"
rm $TAR_FILE
mv $TMPDIR/$REPO .
echo "Checking we can call the codeperf tool"
./codeperf --help