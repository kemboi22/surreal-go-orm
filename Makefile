.PHONY: fmt vet lint fix tidy test all

all: fmt vet lint

fmt:
	gofmt -w -s .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

fix:
	go fix ./...
	go mod tidy

tidy:
	go mod tidy

test:
	go test ./...
