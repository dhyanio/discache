build:
	go build -o bin/discache

run: build
	./bin/discache