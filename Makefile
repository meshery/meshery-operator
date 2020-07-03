build: clean
	go build -tags latest -a -o bin/meshery-controller cmd/manager/main.go

clean:
	rm -rf bin
	go mod tidy

run: build
	./bin/meshery-controller