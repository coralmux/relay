.PHONY: build build-all run test clean dist

BINARY=coralmux-relay
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION)"
DIST=dist

build:
	go build $(LDFLAGS) -o $(BINARY) ./cmd/relay/

# Build for all platforms
build-all: clean
	@mkdir -p $(DIST)
	@echo "Building $(VERSION)..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-amd64 ./cmd/relay/
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-darwin-arm64 ./cmd/relay/
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-amd64 ./cmd/relay/
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(DIST)/$(BINARY)-linux-arm64 ./cmd/relay/
	@echo "Done! Binaries in $(DIST)/"
	@ls -lh $(DIST)/

run:
	go run ./cmd/relay/ -addr :8080

test:
	go test ./...

clean:
	rm -rf $(BINARY) $(DIST)

# Create release archives
dist: build-all
	@cd $(DIST) && for f in $(BINARY)-*; do \
		tar -czf $$f.tar.gz $$f && rm $$f; \
	done
	@echo "Archives created:"
	@ls -lh $(DIST)/
