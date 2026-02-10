.PHONY: build run watch

BINARY := backend

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

run:
	go run .

watch:
	air
