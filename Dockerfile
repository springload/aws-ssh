FROM golang:1.17-alpine as build

RUN apk update && apk add git

WORKDIR /app

ADD go.mod go.sum ./
RUN go mod download

ADD ./ ./

ENV CGO_ENABLED=0
RUN go build

ENTRYPOINT ["/app/aws-ssh"]
