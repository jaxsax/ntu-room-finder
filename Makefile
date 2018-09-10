PACKAGE_SOURCES=$(wildcard pkg/**/*)

.PHONY: default
default: bin/crawl

bin/crawl: $(wildcard cmd/**/*) $(PACKAGE_SOURCES)
	go install ./...

.PHONY: run/crawler
run/crawler: bin/crawl
	./$<

.PHONY: test/pkg
test/pkg: $(PACKAGE_SOURCES)
	go test -v ./pkg/...

clean:
	rm -rf $(CURDIR)/bin/
