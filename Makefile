PACKAGE_SOURCES=$(wildcard pkg/**/*)
INTERNAL_SOURCES=$(wildcard internal/*)

.PHONY: default
default: binaries

binaries: $(wildcard cmd/**/*) $(PACKAGE_SOURCES) $(INTERNAL_SOURCES)
	go install ./...

.PHONY: run/parser
run/parser: binaries
	./bin/parser

.PHONY: run/crawler
run/downloader: binaries
	./bin/downloader

.PHONY: test/pkg
test/pkg: $(PACKAGE_SOURCES)
	go test -v ./pkg/...

clean:
	rm -rf $(CURDIR)/bin/
