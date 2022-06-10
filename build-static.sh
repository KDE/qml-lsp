#!/usr/bin/env sh

export PATH="$PWD/wrappers:$PATH"
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=amd64
export CC=$(which musl-gcc)

echo "Building qml-lsp..."
go build --ldflags '-linkmode external -extldflags "-static"' -o qml-lsp-static ./cmd/qml-lsp

echo "Building qml-lint..."
go build --ldflags '-linkmode external -extldflags "-static"' -o qml-lint-static ./cmd/qml-lint

echo "Building qml-refactor-fairy..."
go build --ldflags '-linkmode external -extldflags "-static"' -o qml-refactor-fairy-static ./cmd/qml-refactor-fairy
