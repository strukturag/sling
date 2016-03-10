PWD := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GOPKG = golang.struktur.de/sling
GOPATH = "$(CURDIR)/vendor:$(CURDIR)"
SYSTEM_GOPATH = /usr/share/gocode/src/

DIST := $(PWD)/dist
DIST_SRC := $(DIST)/src

FOLDERS = $(shell find -mindepth 1 -maxdepth 1 -type d -not -path "*.git" -not -path "*debian" -not -path "*vendor" -not -path "*doc" -not -path "*test")

all:

$(DIST_SRC):
	mkdir -p $@

dist_gopath: $(DIST_SRC)
	if [ -d "$(SYSTEM_GOPATH)" ]; then find $(SYSTEM_GOPATH) -mindepth 1 -maxdepth 1 -type d \
		-exec ln -sf {} $(DIST_SRC) \; ; fi
	if [ ! -d "$(SYSTEM_GOPATH)" ]; then find $(CURDIR)/vendor/src -mindepth 1 -maxdepth 1 -type d \
		-exec ln -sf {} $(DIST_SRC) \; ; fi

goget:
	if [ -z "$(DEB_BUILDING)" ]; then GOPATH=$(GOPATH) go get launchpad.net/godeps; fi
	if [ -z "$(DEB_BUILDING)" ]; then GOPATH=$(GOPATH) $(CURDIR)/vendor/bin/godeps -u dependencies.tsv; fi
	mkdir -p $(shell dirname "$(CURDIR)/vendor/src/$(GOPKG)")
	rm -f $(CURDIR)/vendor/src/$(GOPKG)
	ln -sf $(PWD) $(CURDIR)/vendor/src/$(GOPKG)

build: goget
	GOPATH=$(GOPATH) go build

test: goget
	GOPATH=$(GOPATH) go test -v

format:
	go fmt

dependencies.tsv:
	set -e ;\
	TMP=$$(mktemp -d) ;\
	cp -r $(CURDIR)/vendor $$TMP ;\
	GOPATH=$$TMP/vendor:$(CURDIR) $(CURDIR)/vendor/bin/godeps $(GOPKG)/baddsch > $(CURDIR)/dependencies.tsv ;\
	rm -rf $$TMP ;\

.PHONY: all dist_gopath goget build dependencies.tsv

