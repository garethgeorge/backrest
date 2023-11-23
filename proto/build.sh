#! /bin/bash 
buf generate
find ../gen/go -name "*.pb.go" -exec protoc-go-inject-tag -input="{}" \;