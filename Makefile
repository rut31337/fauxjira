.PHONY: test lint fmt build clean venv ansible-lint pre-commit-install check

test:
	go test -v -race ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

build:
	CGO_ENABLED=0 go build -o fauxjira ./cmd/fauxjira

clean:
	rm -f fauxjira fauxjira.db

venv:
	python3 -m venv .venv
	.venv/bin/pip install -r requirements.txt
	.venv/bin/ansible-galaxy collection install kubernetes.core

ansible-lint:
	.venv/bin/ansible-lint ansible/deploy-fauxjira.yml --profile=basic

pre-commit-install:
	pre-commit install

check: fmt lint test build ansible-lint
