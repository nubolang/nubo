.PHONY: build run

build:
	@go build -o bin/nubo ./cmd/nubo

run: build
	@./bin/nubo run ./example/$(FILE).nubo --dev

prepare: build
	@./bin/nubo prepare ./example --dev

format: build
	@./bin/nubo format ./example/$(FILE) --dev
