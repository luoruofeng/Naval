# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOGET=$(GOCMD) get
BINARY_NAME=Naval
BINARY_LINUX=$(BINARY_NAME)_linux

all: build
build:
		$(GOBUILD) -o $(BINARY_NAME) -v
clean: 
		$(GOCLEAN)
		rm -f $(BINARY_NAME)
		rm -f $(BINARY_LINUX)
run:
		$(GOBUILD) -o $(BINARY_NAME) -v
		./$(BINARY_NAME)
deps:
		$(GOGET) go.uber.org/zap
		$(GOGET) go.uber.org/fx
		$(GOGET) github.com/gorilla/mux
package:
		$(GOBUILD) -o $(BINARY_NAME) -v
		tar -czvf $(BINARY_NAME).tar.gz $(BINARY_NAME) README.md LICENSE
build-linux:
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_LINUX) -v
build-windows:
		CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BINARY_NAME) -v
