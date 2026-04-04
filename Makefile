BINARY_NAME=yomite

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

build:
	go build -o $(BINARY_NAME) .

setup-gui:
	cd frontend && npm install

build-gui:
	cd frontend && npm run build
	go build -tags gui -o $(BINARY_NAME)-gui .

clean:
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME)-gui

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

run:
	go run . ${options}

all: build

.PHONY: all setup setup-gui build build-gui clean test fmt lint
