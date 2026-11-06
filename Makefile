APP := ifacematch-docs
TEMPL_VERSION := v0.3.1020

.PHONY: generate test vet build run

generate:
	go run github.com/a-h/templ/cmd/templ@$(TEMPL_VERSION) generate

test: generate
	go test -mod=mod -race ./...

vet: generate
	go vet -mod=mod ./...

build: generate
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=mod -trimpath -ldflags="-s -w" -o dist/$(APP)-linux-amd64 ./cmd/docs

run: generate
	go run -mod=mod ./cmd/docs -listen 127.0.0.1:8091
