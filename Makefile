mkfile_path := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

tag       = $(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD)
build     = $(shell git rev-parse --short HEAD)
date      = $(shell date "+%Y-%m-%d")
buildargs = -ldflags "-X main.version=$(tag) -X main.build=$(build) -X main.buildDate=${date}" -v

all: test build

test:
	go test -v -race ./...

build:
	cd cmd/stubserver && go build $(buildargs)

build-static:
	cd cmd/stubserver && CGO_ENABLED=0 go build -a -installsuffix cgo $(buildargs)

run: build
	@export `cat ${mkfile_path}.env | xargs`; ./cmd/stubserver/stubserver serve -config example.yml

install:
	cd cmd/stubserver && go install
