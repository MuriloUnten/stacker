build:
	go build -tags "linux"

run: build
	./stacker

all: run
