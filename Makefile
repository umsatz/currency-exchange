GO ?= go

.PHONY: default

default:
	pushd api && $(GO) test; popd
	pushd data && $(GO) test; popd