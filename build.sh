#! /bin/bash
echo 'building linux'
GOOS=linux GOARCH=amd64 go build -o mdict-linux app.go
echo 'building mac'
GOOS=darwin GOARCH=arm64 go build -o mdict-mac app.go

