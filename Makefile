BINARY  := menukey
VERSION ?= dev
LDFLAGS := -X github.com/seiji/menukey/cmd.version=$(VERSION)

.PHONY: build test test-integration fmt vet clean install

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

# Writes to a scratch preference domain through defaults(1).
test-integration:
	go test -tags integration ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

install:
	go install -ldflags "$(LDFLAGS)" .

clean:
	rm -f $(BINARY)
