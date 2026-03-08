.PHONY: build install clean test fmt lint deploy-ec2

VERSION := 0.1.0
BINARY := ccvalet
BUILD_DIR := bin
EC2_HOST ?= ec2-dev

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/ccvalet

install:
	go install ./cmd/ccvalet

clean:
	rm -rf $(BUILD_DIR)

test:
	go test -v ./...

test-short:
	go test -short -v ./...

test-e2e:
	go test -tags e2e -v ./test/e2e/

test-race:
	go test -race ./...

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo "HTML report: go tool cover -html=coverage.out -o coverage.html"

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

# Deploy to EC2 (Ubuntu)
deploy-ec2:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/ccvalet
	scp $(BUILD_DIR)/$(BINARY)-linux-amd64 $(EC2_HOST):/tmp/$(BINARY)
	ssh $(EC2_HOST) 'sudo mv /tmp/$(BINARY) ~/.local/bin/$(BINARY) && sudo chmod +x ~/.local/bin/$(BINARY)'
	@echo "Deployed $(BINARY) to $(EC2_HOST):~/.local/bin/$(BINARY)"
