check:
	golangci-lint run

docker: check
	docker build -t layer5/meshery-meshsync .

docker-run:
	(docker rm -f meshery-meshsync) || true
	docker run --name meshery-meshsync -d \
	-p 10007:10007 \
	-e DEBUG=true \
	layer5/meshery-meshsync

run: check
	DEBUG=true go run meshsync.go