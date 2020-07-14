build: clean
	go build -tags latest -a -o bin/meshsync cmd/meshsync/main.go

clean:
	rm -rf bin
	go mod tidy

run: build
	./bin/meshsync
