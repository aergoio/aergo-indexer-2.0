all: bin/indexer

protoc:
	protoc \
	--proto_path=./aergo-protobuf/proto \
	--go-grpc_out=./types \
	--go-grpc_opt=paths=source_relative \
	--go_out=./types \
	--go_opt=paths=source_relative \
	./aergo-protobuf/proto/*.proto

bin/indexer: *.go indexer/*.go indexer/**/*.go types/*.go go.sum go.mod
	go build -o bin/indexer main.go

unittest:
	go test ./... -short

test:
	go test ./...

cover-test:
	go test -coverprofile=coverage.out ./...
	gocover-cobertura < coverage.out > coverage.xml

clean:
	go clean -testcache

run:
	go run main.go