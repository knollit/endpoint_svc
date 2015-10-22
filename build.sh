#!/bin/sh

mkdir -p proto
protoc --go_out=proto *.proto
go get
CGO_ENABLED=0 GOOS=linux go build -a --installsuffix cgo --ldflags="-s" -o endpoints .
docker build -t endpoints:latest .
rm endpoints
