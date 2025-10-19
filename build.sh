#!/usr/bin/env bash

CGO_ENABLED=1 go build -ldflags '-extldflags "-static" -w -s' -o app ./cmd/dashboard/main.go

docker build -t twoonefour1/nezha:v0 .
docker login
docker push twoonefour1/nezha:v0

python3 -m http.server -b ::