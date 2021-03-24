all: wasm_exec.js app.wasm

wasm_exec.js:
	cp "$$(go env GOROOT)/misc/wasm/wasm_exec.js" ./wasm_exec.js
	patch wasm_exec.js < wasm_exec.js.patch

app.wasm: wasm_exec.js *.go
	GOOS=js GOARCH=wasm go build -o app.wasm

clean:
	rm -f app.wasm

.PHONY: all clean
