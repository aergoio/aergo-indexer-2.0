all: bin/indexer

protoc:
	protoc -I./aergo-protobuf/proto --go_out=plugins=grpc,paths=source_relative:./types ./aergo-protobuf/proto/*.proto

bin/indexer: *.go indexer/*.go indexer/**/*.go types/*.go go.sum go.mod
	go build -o bin/indexer main.go

clean:
	go clean

run:
	go run main.go

run-reindex:
	go run main.go --reindex
