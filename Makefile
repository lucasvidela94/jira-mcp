.PHONY: build test test-race vet fmt clean

build:
	go build ./...

test:
	go test ./...

test-race:
	go test -race ./...

test-cover:
	go test -cover ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

clean:
	rm -f jira-mcp
