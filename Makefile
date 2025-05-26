.PHONY: build run

build:
	@go build -o bin/nubo

run: build
	@./bin/nubo --dev
