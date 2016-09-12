BASEDIR=$(CURDIR)
TOOLDIR=$(BASEDIR)/script

BINARY=twivility
SOURCES := $(shell find $(BASEDIR) -name '*.go')
TESTED=.tested

build: $(BINARY)
$(BINARY): $(SOURCES) $(TESTED)
	$(TOOLDIR)/build

clean:
	rm -f $(BINARY) debug debug.test cover.out $(TESTED)

test: $(TESTED)
$(TESTED): $(SOURCES)
	$(TOOLDIR)/test

cover: $(SOURCES)
	$(TOOLDIR)/cover

run: build
	@echo ""
	@echo "------------------------------------"
	@echo "Use ./script/run directly"
	@echo "------------------------------------"
	@echo ""

update: clean
	$(TOOLDIR)/update

.PHONY: clean test cover build run update
