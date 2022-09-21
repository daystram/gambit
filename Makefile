GO:=$(shell which go)
GO_TARGET=$(shell go list ./...)

.PHONY: build
build:
	@${GO} build ./cmd/gambit

.PHONY: test
test:
	@${GO} test -v -cover ${GO_TARGET}
