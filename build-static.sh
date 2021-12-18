#!/usr/bin/env sh

export PATH="$PWD/wrappers:$PATH"
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
export CC=$(which musl-gcc)

go build --ldflags '-linkmode external -extldflags "-static"' -o qml-lsp-static
