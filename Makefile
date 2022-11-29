all: bin/indexer bin/clean_index

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

bin/clean_index:
	pyinstaller --onefile --clean --workpath ./clean_index/build/ --distpath ./clean_index/dist --specpath ./clean_index clean_index/clean_index.py
	mv ./clean_index/dist/clean_index ./bin

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