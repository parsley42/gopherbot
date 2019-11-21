# Makefile - just builds the binary, for dev mainly

.PHONY: clean test generate testbot

commit := $(shell git rev-parse --short HEAD)

GOOS ?= linux
CGO ?= 1

gopherbot: main.go bot/* brains/*/* connectors/*/* goplugins/*/* history/*/*
	CGO_ENABLED=${CGO} GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -ldflags "-X main.Commit=$(commit)" -tags 'netgo osusergo static_build' -o gopherbot

clean:
	rm -f gopherbot

# Run test suite
test:
	go test -v --tags 'test integration netgo osusergo static_build' -mod vendor -cover -race -coverprofile coverage.out -coverpkg ./... ./bot

# Generate Stringer methods
generate:
	go generate -v --tags 'test integration netgo osusergo static_build' -mod vendor ./bot/

testbot:
	CGO_ENABLED=0 GOOS=${GOOS} GOARCH=amd64 go build -mod vendor -tags 'netgo osusergo static_build test' -o gopherbot
