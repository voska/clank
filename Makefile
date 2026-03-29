.PHONY: build test lint fmt clean install run vulncheck ci

VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/clank .

install:
	go install -ldflags "$(LDFLAGS)" .

run:
	go run .

test:
	go test -race ./...

test-cover:
	go test -race -coverprofile=coverage.txt ./...
	go tool cover -html=coverage.txt -o coverage.html

lint:
	golangci-lint run

fmt:
	golangci-lint fmt

vulncheck:
	govulncheck ./...

clean:
	rm -rf bin/ dist/ coverage.txt coverage.html

ci: lint test vulncheck build
