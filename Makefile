GO ?= go
PACKAGE := dist/moonwing
SOURCE_PACKAGE := dist/moonwing-source
ROOT_SOURCES := go.mod Makefile icon.png icon-prompt.txt README.md
SOURCES := src/app.json $(wildcard src/*.go)
ARTWORK := $(wildcard src/artwork/*.png)

.PHONY: all package test
all:
	mkdir -p $(PACKAGE)
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build -trimpath -buildvcs=false -ldflags='-s -w' -o $(PACKAGE)/moonwing ./src
	cp src/app.json icon.png $(PACKAGE)/

package: all
	@rm -rf $(SOURCE_PACKAGE)
	mkdir -p $(SOURCE_PACKAGE)/src/artwork
	cp $(ROOT_SOURCES) $(SOURCE_PACKAGE)/
	cp $(SOURCES) $(SOURCE_PACKAGE)/src/
	@cp $(ARTWORK) $(SOURCE_PACKAGE)/src/artwork/
	@cd dist && zip -q -FS moonwing-compas.zip moonwing/moonwing moonwing/app.json moonwing/icon.png $(addprefix moonwing-source/,$(ROOT_SOURCES) $(SOURCES) $(ARTWORK))

test:
	$(GO) test ./...
