#!/bin/sh
go build -v -ldflags="-s -w" -buildmode=plugin ./plugins/base
go build -v -ldflags="-s -w" -buildmode=plugin ./plugins/git
go build -v -ldflags="-s -w" -buildmode=plugin ./plugins/goget
