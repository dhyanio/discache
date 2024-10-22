build:
	go build -o bin/discache

run: build
	./bin/discache

runfollower: build
	./bin/discache --listenaddr :4000 --leaderaddr :3000

test:
	@go test -v ./...