.PHONY: build clean install test

BINARY_NAME=sysmetrics-mcp
INSTALL_PATH=/usr/local/bin

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -o bin/$(BINARY_NAME) ./cmd/sysmetrics-mcp

clean:
	rm -f bin/$(BINARY_NAME)
	go clean

install: build
	sudo cp bin/$(BINARY_NAME) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)

uninstall:
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)

test:
	CGO_ENABLED=0 go test -v ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run
	go vet ./...

deps:
	go mod download
	go mod tidy
