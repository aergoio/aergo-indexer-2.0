all: bin/indexer bin/clean_index

protoc:
	protoc -I./aergo-protobuf/proto --go_out=plugins=grpc,paths=source_relative:./types ./aergo-protobuf/proto/*.proto

bin/indexer: *.go indexer/*.go indexer/**/*.go types/*.go go.sum go.mod
	go build -o bin/indexer main.go

bin/clean_index:
	pyinstaller --onefile --clean --workpath ./clean_index/build/ --distpath ./clean_index/dist --specpath ./clean_index clean_index/clean_index.py
	mv ./clean_index/dist/clean_index ./bin

clean:
	go clean

run:
	go run main.go

