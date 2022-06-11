#!/bin/bash

GOOS=js GOARCH=wasm go build -o web/game.wasm github.com/hdm/facecrime && \
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" web/