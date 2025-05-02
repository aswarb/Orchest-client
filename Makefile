BUILD_DIR=./bin/

all: build

build:
	mkdir -p ${BUILD_DIR}
	go build -o ${BUILD_DIR} ./cmd/...

clean:
	go clean
	rm ${BUILD_DIR}/*
