build: clean
	go build -tags latest -a -o bin/meshery-operator cmd/manager/main.go

clean:
	rm -rf bin
	go mod tidy

run: build
	./bin/meshery-operator
