GO ?= go

.PHONY: default

default:
	cd ./api && $(GO) test
	cd ./data && $(GO) test

download:
	curl http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml -o data/euro-hist.xml

	cd ./importer && $(GO) build
	./importer/importer -history=./data/euro-hist.xml -out=./data