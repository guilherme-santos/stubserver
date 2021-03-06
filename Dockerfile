# Builder

FROM golang:1.10-alpine as builder

RUN apk update \
    && apk upgrade \
    && apk add --no-cache git bash make curl \
    && curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh

WORKDIR /go/src/github.com/guilherme-santos/stubserver

COPY Makefile Gopkg.toml Gopkg.lock ./

RUN dep ensure -vendor-only

COPY . ./

RUN make build-static install

# Final docker image

FROM alpine:3.7

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/stubserver /usr/local/bin/

EXPOSE 80

CMD stubserver serve -c config.yml
