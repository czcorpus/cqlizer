VERSION=`git describe --tags`
BUILD=`date +%FT%T%z`
HASH=`git rev-parse --short HEAD`

LDFLAGS=-ldflags "-w -s -X main.version=${VERSION} -X main.buildDate=${BUILD} -X main.gitCommit=${HASH}"


.PHONY: clean tools build generate

all:	generate build

build:
	@echo "building the project without running unit tests"
	@go build ${LDFLAGS} -o cqlizer


tools:
	@echo "installing local dependencies"
	@go install github.com/mna/pigeon

generate:
	@echo "generating query parser code"
	@go generate ./cqlizer.go

