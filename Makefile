.PHONY: build test
#
build:
	echo "start build"
	go build . 
	echo "end build"
#
test:

	echo "start build"
	go build .
	go test -v ./...
	echo "end build"
#