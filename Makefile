PACKAGE_SOURCES=$(wildcard pkg/**/*)
INTERNAL_SOURCES=$(wildcard internal/*)

.PHONY: default
default: bin/crawl

bin/crawl: $(wildcard cmd/**/*) $(PACKAGE_SOURCES) $(INTERNAL_SOURCES)
	go install ./...

.PHONY: run/crawler
run/crawler: bin/crawl
	./$<

.PHONY: test/pkg
test/pkg: $(PACKAGE_SOURCES)
	go test -v ./pkg/...

clean:
	rm -rf $(CURDIR)/bin/
