PY?=python3
PELICAN?=pelican
PELICANOPTS=

BASEDIR=$(CURDIR)
TOOLDIR=$(BASEDIR)/script

BINARY=twivility
SOURCES := $(shell find $(BASEDIR) -name '*.go')
TESTED=.tested

$(BINARY): $(SOURCES) $(TESTED)
	$(TOOLDIR)/build
build: $(binary)

clean:
	rm -f $(BINARY) debug debug.test cover.out $(TESTED)

$(TESTED): $(SOURCES)
	$(TOOLDIR)/test
test: $(TESTED)

cover: $(SOURCES)
	$(TOOLDIR)/cover

run: build
	@echo "Use ./script/run directly"

update: clean
	$(TOOLDIR)/update

.PHONY: clean test cover build run update
