.PHONY: build clean install test

BINARY_NAME=sysmetrics-mcp
INSTALL_PATH=/usr/local/bin

build:
	go build -o bin/$(BINARY_NAME) ./cmd/sysmetrics-mcp

clean:
	rm -f bin/$(BINARY_NAME)
	go clean

install: build
	sudo cp bin/$(BINARY_NAME) $(INSTALL_PATH)/
	sudo chmod +x $(INSTALL_PATH)/$(BINARY_NAME)

uninstall:
	sudo rm -f $(INSTALL_PATH)/$(BINARY_NAME)

test:
	go test -v ./...

lint:
	go vet ./...

deps:
	go mod download
	go mod tidy
