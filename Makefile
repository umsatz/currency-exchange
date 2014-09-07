GO ?= go

.PHONY: default

default:
	pushd ./api && $(GO) test
	popd

	pushd ./data && $(GO) test
	popd

download:
	curl http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml -o data/euro-hist.xml

	pushd ./importer && $(GO) build; popd
	./importer/importer -history=./data/euro-hist.xml -out=./data