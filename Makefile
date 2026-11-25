GO ?= go
PACKAGE := dist/moonwing
SOURCE_PACKAGE := dist/moonwing-source
SOURCES := go.mod Makefile app.json icon.png icon-prompt.txt README.md $(wildcard *.go)
ARTWORK := $(wildcard artwork/*.png)

.PHONY: all package test
all:
	mkdir -p $(PACKAGE)
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build -trimpath -buildvcs=false -ldflags='-s -w' -o $(PACKAGE)/moonwing .
	cp app.json icon.png $(PACKAGE)/

package: all
	mkdir -p $(SOURCE_PACKAGE)/artwork
	@rm -f $(SOURCE_PACKAGE)/*.go
	cp $(SOURCES) $(SOURCE_PACKAGE)/
	@cp $(ARTWORK) $(SOURCE_PACKAGE)/artwork/
	@cd dist && zip -q -FS moonwing-compas.zip moonwing/moonwing moonwing/app.json moonwing/icon.png $(addprefix moonwing-source/,$(SOURCES) $(ARTWORK))

test:
	$(GO) test ./...
