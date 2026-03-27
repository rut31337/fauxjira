.PHONY: test lint fmt build clean

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

build:
	CGO_ENABLED=0 go build -o fauxjira .

clean:
	rm -f fauxjira fauxjira.db

pre-commit-install:
	pre-commit install

check: fmt lint test build
