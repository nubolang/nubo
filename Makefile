.PHONY: build run prepare format serve

build:
	@go build -o bin/nubo ./cmd/nubo

run: build
	@./bin/nubo ./example/v2/$(FILE).nubo --dev --loglevel=WARN

prepare: build
	@./bin/nubo prepare ./example/v2 --dev

format: build
	@./bin/nubo format ./example/v2/$(FILE) --dev

serve: build
	@./bin/nubo serve ./example/v2/$(FOLDER) --dev --loglevel=WARN
