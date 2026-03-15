.PHONY: build build-wasm serve clean

build:
	go build -o bin/cli/luadata ./cmd/luadata

build-wasm:
	GOOS=js GOARCH=wasm go build -o bin/web/luadata.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" bin/web/
	cp web/index.html bin/web/

serve: build-wasm
	@echo "Serving at http://localhost:8080"
	cd bin/web && python3 -m http.server 8080

clean:
	rm -rf bin
