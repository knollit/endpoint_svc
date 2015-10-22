#!/bin/sh

./build.sh
docker run --link endpoints-postgres:postgres -e POSTGRES_PASSWORD=mysecretpassword -e TLS_CA_PATH=/test-ca.crt -e TLS_CERT_PATH=/test-server.crt -e TLS_KEY_PATH=/test-server.key --name endpoints --rm endpoints:latest
