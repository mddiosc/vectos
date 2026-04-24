.PHONY: build clean

build:
	go build -o vectos ./cmd/vectos

clean:
	rm -f vectos
