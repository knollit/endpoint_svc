#!/bin/sh

docker build -t endpoint-db:latest .
docker run --name endpoints-postgres -e POSTGRES_PASSWORD=mysecretpassword -d endpoint-db:latest
