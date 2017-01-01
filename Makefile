GO         ?= golang:1.7.4-onbuild
COMMIT     := $(shell git rev-parse --short HEAD)
VERSION    := $(shell git describe --abbrev=0 --tags)

LDFLAGS    := -ldflags \
              "-s \
               -X main.Commit=$(COMMIT)\
               -X main.Version=$(VERSION)"


.PHONY: default download

data/eurofxref-hist.xml:
	curl http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml -o data/eurofxref-hist.xml

default: *.go
	$(GOBUILD)

archive: dist/$(ARCHIVE)

all: compile build

compile: data/eurofxref-hist.xml
	docker run --rm -v "$(PWD)":/usr/src/currency-exchange -w /usr/src/currency-exchange -e CGO_ENABLED=0 -e GOOS=linux -e GOARCH=amd64 $(GO) go build -a --installsuffix cgo $(LDFLAGS) -v

clean:
	rm currency-exchange

build: compile
	docker build -t currency-exchange:$(VERSION) .

test: data/eurofxref-hist.xml
	docker run --rm -v "$(PWD)":/usr/src/currency-exchange -w /usr/src/currency-exchange $(GO) go test -v .
