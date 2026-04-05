BINARY_NAME=yomite

setup:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0

build:
	go build -o $(BINARY_NAME) ./cmd/yomite

setup-gui:
	cd frontend && npm install

build-gui:
	mkdir -p frontend/dist && touch frontend/dist/.gitkeep
	wails build -tags gui -o $(BINARY_NAME)-gui

clean:
	go clean
	rm -f $(BINARY_NAME)
	rm -rf build/bin

test:
	go test ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run ./...

run:
	go run ./cmd/yomite ${options}

run-gui:
	wails dev -tags gui -appargs "${options}"

all: build

.PHONY: all setup setup-gui build build-gui clean test fmt lint run run-gui
