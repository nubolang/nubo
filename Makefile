.PHONY: build build-all run prepare format serve

build:
	@go build -o bin/nubo ./cmd/nubo
	@echo "🚀 Build complete"

build-all:
	@mkdir -p bin/releases
	@GOOS=darwin GOARCH=amd64 go build -o bin/releases/nubo-darwin-amd64 ./cmd/nubo
	@GOOS=darwin GOARCH=arm64 go build -o bin/releases/nubo-darwin-arm64 ./cmd/nubo
	@GOOS=linux GOARCH=amd64 go build -o bin/releases/nubo-linux-amd64 ./cmd/nubo
	@GOOS=linux GOARCH=arm64 go build -o bin/releases/nubo-linux-arm64 ./cmd/nubo
	@GOOS=windows GOARCH=amd64 go build -o bin/releases/nubo-windows-amd64.exe ./cmd/nubo
	@GOOS=windows GOARCH=arm64 go build -o bin/releases/nubo-windows-arm64.exe ./cmd/nubo
	@echo "🚀 Cross-platform build complete (darwin/linux/windows)"

run: build
	@./bin/nubo ./example/v2/$(FILE).nubo --dev --loglevel=WARN --nocolor

runinfo: build
	@./bin/nubo ./example/v2/$(FILE).nubo --dev --loglevel=INFO

runex: build
	@./bin/nubo ./examples/$(FILE).nubo --dev --loglevel=INFO

prepare: build
	@./bin/nubo prepare ./example/v2 --dev

format: build
	@./bin/nubo format ./example/v2/$(FILE) --dev

serve: build
	@./bin/nubo serve ./example/v2/$(FOLDER) --dev --loglevel=WARN

smoketest: build
	@./bin/nubo ./tests/run_all.nubo
