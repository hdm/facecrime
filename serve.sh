go install github.com/hajimehoshi/wasmserve@latest && \
echo "Serving application on http://127.0.0.1:9999/" && \
wasmserve -allow-origin '*' -http 127.0.0.1:9999 .
