.PHONY: test build up smoke clean

test:
	go test ./...

build:
	mkdir -p bin
	go build -o bin/gateway ./cmd/gateway
	go build -o bin/game-server ./cmd/game-server

up:
	docker compose up -d --build

smoke:
	./scripts/smoke-test.sh

clean:
	rm -rf bin
