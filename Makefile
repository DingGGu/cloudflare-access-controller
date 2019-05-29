GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=cloudflare-access-controller

all: build
build:
	$(GOBUILD) -o build/$(BINARY_NAME) -v
clean:
	$(GOCLEAN)
	rm -rf build/
run:
	$(GOBUILD) -o build/$(BINARY_NAME) -v ./...
	./$(BINARY_NAME)