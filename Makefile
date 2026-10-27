include .env
export

run:
	go run ./cmd/app

test:
	go test ./... -race
