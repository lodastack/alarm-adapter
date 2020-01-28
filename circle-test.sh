#!/bin/bash

go version

export GO111MODULE=on
make build

go test -timeout 60s -v ./...
