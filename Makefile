BINARY=tailnginx

.PHONY: build run test tidy

build:
	go build -o $(BINARY) ./cmd/tailnginx

run: build
	./$(BINARY)

test:
	go test ./...

tidy:
	go mod tidy
