#!/bin/bash

export GOMEMLIMIT="800MiB"
export GOGC=100
export GODEBUG=gctrace=1
export UNIDOC_LICENSE_API_KEY="YOUR_API_KEY"

go mod init unipdf_memory_limit && go mod tidy
go get -u github.com/unidoc/unipdf/v3/...
go run main.go unipdf-large-pdf.pdf result-docker.pdf
