SHA_SHORT=$(shell git rev-parse --short HEAD)
VERSION?=v0.0.0-${SHA_SHORT}
BIN?=gambit-${SHA_SHORT}
BUILD_DIR?=./build

GO:=$(shell which go)
GO_TARGET:=$(shell go list ./...)
LDFLAGS+="-X 'github.com/daystram/gambit/uci.EngineVersion=${VERSION}'"

ifeq (${PLATFORM},windows-amd64)
	GO_GOOS:=windows
	GO_GOARCH:=amd64
endif
ifeq (${PLATFORM},linux-amd64)
	GO_GOOS:=linux
	GO_GOARCH:=amd64
endif
ifeq (${PLATFORM},darwin-arm64)
	GO_GOOS:=darwin
	GO_GOARCH:=arm64
endif

.PHONY: build-windows-amd64
build-windows-amd64:
	@make PLATFORM=windows-amd64 GOAMD64=v1 BIN=${BIN}-windows-amd64-base build
	@make PLATFORM=windows-amd64 GOAMD64=v2 BIN=${BIN}-windows-amd64-popcnt build
	@make PLATFORM=windows-amd64 GOAMD64=v3 BIN=${BIN}-windows-amd64-avx2 build
	@make PLATFORM=windows-amd64 GOAMD64=v4 BIN=${BIN}-windows-amd64-avx512 build

.PHONY: build-linux-amd64
build-linux-amd64:
	@make PLATFORM=linux-amd64 GOAMD64=v1 BIN=${BIN}-linux-amd64-base build
	@make PLATFORM=linux-amd64 GOAMD64=v2 BIN=${BIN}-linux-amd64-popcnt build
	@make PLATFORM=linux-amd64 GOAMD64=v3 BIN=${BIN}-linux-amd64-avx2 build
	@make PLATFORM=linux-amd64 GOAMD64=v4 BIN=${BIN}-linux-amd64-avx512 build

.PHONY: build-darwin-arm64
build-darwin-amd64:
	@make PLATFORM=darwin-arm64 BIN=${BIN}-darwin-arm64-base build

.PHONY: build
build:
	@mkdir -p ${BUILD_DIR}
	@GOOS=${GO_GOOS} \
	GOARCH=${GO_GOARCH} \
	CGO_ENABLED=${GO_CGO_ENABLED} \
	CXX=${GO_CXX}\
	CC=${GO_CC} \
	${GO} build -ldflags=${LDFLAGS} -o ${BUILD_DIR}/${BIN} ./cmd/gambit

.PHONY: test
test:
	@${GO} test -v -cover ${GO_TARGET}

.PHONY: clean
clean:
	@rm -f gambit
	@rm -f gambit.exe
