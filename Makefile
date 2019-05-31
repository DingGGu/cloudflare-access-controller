GOCMD=go
GOTEST=$(GOCMD) test
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
BINARY_NAME=cloudflare-access-controller

all: test build
docker:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME)
	docker build .
	rm $(BINARY_NAME)
test:
	$(GOTEST) ./...
build:
	$(GOBUILD) -o build/$(BINARY_NAME) -v
clean:
	$(GOCLEAN)
	rm -rf build/
run:
	$(GOBUILD) -o build/$(BINARY_NAME) -v ./...
	./$(BINARY_NAME)