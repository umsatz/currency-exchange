GO         ?= go
COMMIT     := $(shell git rev-parse --short HEAD)
VERSION    := 1.1.1

LDFLAGS    := -ldflags \
              "-X main.Commit $(COMMIT)\
               -X main.Version $(VERSION)"

GOOS       := $(shell go env GOOS)
GOARCH     := $(shell go env GOARCH)
GOBUILD    := GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o dist/api $(LDFLAGS)
GOFILES    := $(shell find . -name "*.go" -exec echo {}  \; | sed -e s/.\\/// | grep -ve test)

ARCHIVE    := currency-api-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz
DISTDIR    := dist/$(GOOS)_$(GOARCH)

.PHONY: default archive clean install download


download:
	curl http://www.ecb.europa.eu/stats/eurofxref/eurofxref-hist.xml -o data/eurofxref-hist.xml

default: *.go
	$(GOBUILD)

archive: dist/$(ARCHIVE)

all: build

build:
	$(GO) build

clean:
	git clean -f -x -d

dist/$(ARCHIVE): $(DISTDIR)/api
	tar -C $(DISTDIR) -czvf $@ .

$(DISTDIR)/api: *.go
	$(GOBUILD) -o $@
