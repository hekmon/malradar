#!/bin/bash -e

# https://hub.docker.com/_/golang
# https://hub.docker.com/_/alpine

if [ "$0" != "./build.sh" ]; then
    echo "Start the build script from the docker folder: ./build.sh" >&2
    exit 1
fi

echo "* Building alpine malradar binary"
docker run --rm -v "$PWD/..":/usr/src/github.com/hekmon/malradar -w /usr/src/github.com/hekmon/malradar golang:1.17.0-alpine3.14 go build -v -ldflags "-s -w" -o docker/malradar_alpine
echo
echo "* Building alpine container image"
docker build -t hekmon/malradar:1.0.2 -t hekmon/malradar:latest .
echo