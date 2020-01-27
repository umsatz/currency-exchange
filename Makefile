GO         ?= golang:1.13-alpine
COMMIT     := $(shell git rev-parse --short HEAD)
VERSION    := $(shell git describe --abbrev=0 --tags)

LDFLAGS    := -ldflags \
              "-s \
               -X main.Commit=$(COMMIT)\
               -X main.Version=$(VERSION)"


.PHONY: default download

data/eurofxref-hist.xml:
	curl https://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml -o data/eurofxref-hist.xml

default: *.go
	$(GOBUILD)

archive: dist/$(ARCHIVE)

all: compile build

compile: data/eurofxref-hist.xml
	go build -a --installsuffix cgo $(LDFLAGS) -v

clean:
	rm currency-exchange

test: data/eurofxref-hist.xml
	go test -v .
