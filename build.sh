#!/bin/bash

set -e

export GOPATH="${PWD}:${GOPATH}"
export CGO_ENABLED=0
export GOOS=linux

go build -a octo.it/github/bot
