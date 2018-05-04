tag    := $(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD)
build := $(shell git rev-parse --short HEAD)

buildargs = -ldflags "-X main.version=$(tag) -X main.build=$(build)" -v

all: test build

test:
	go test -v -race ./...

build:
	cd cmd/stubserver && go build $(buildargs)
