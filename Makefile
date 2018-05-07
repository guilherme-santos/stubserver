mkfile_path := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

tag       := $(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD)
build     := $(shell git rev-parse --short HEAD)
buildargs = -ldflags "-X main.version=$(tag) -X main.build=$(build)" -v

all: test build

test:
	go test -v -race ./...

build:
	cd cmd/stubserver && go build $(buildargs)

run: build
	@export `cat ${mkfile_path}.env | xargs`; ./cmd/stubserver/stubserver serve -config example.yml

install: build
	cd cmd/stubserver && go install
