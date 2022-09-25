GO:=$(shell which go)
GO_TARGET:=$(shell go list ./...)
ifeq (${PLATFORM},windows-amd64)
	GO_GOOS:=windows
	GO_GOARCH:=amd64
	GO_CGO_ENABLED:=1
	GO_CXX:=x86_64-w64-mingw32-g++
	GO_CC:=x86_64-w64-mingw32-gcc
endif

.PHONY: build
build:
	@GOOS=${GO_GOOS} \
	GOARCH=${GO_GOARCH} \
	CGO_ENABLED=${GO_CGO_ENABLED} \
	CXX=${GO_CXX}\
	CC=${GO_CC} \
	${GO} build ./cmd/gambit

.PHONY: test
test:
	@${GO} test -v -cover ${GO_TARGET}

.PHONY: clean
clean:
	@rm -f gambit
	@rm -f gambit.exe
