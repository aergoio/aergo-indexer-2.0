all: bin/indexer

protoc:
	protoc -I./aergo-protobuf/proto --go_out=plugins=grpc,paths=source_relative:./types ./aergo-protobuf/proto/*.proto

bin/indexer: *.go indexer/*.go indexer/**/*.go types/*.go go.sum go.mod
	cp indexer/documents/documents.single indexer/documents/documents.go
	go build -o bin/indexer_single main.go
	cp indexer/documents/documents.cluster indexer/documents/documents.go
	go build -o bin/indexer_cluster main.go

clean:
	go clean

run:
	go run main.go

