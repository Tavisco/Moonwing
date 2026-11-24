GO ?= go
PACKAGE := dist/moonwing
SOURCE_PACKAGE := dist/moonwing-source
DEBUG_PACKAGE := dist/debug-norotation/moonwing
SOURCES := go.mod Makefile app.json icon.png icon-prompt.txt README.md $(wildcard *.go)
ARTWORK := $(wildcard artwork/*.png)

.PHONY: all package debug-norotation test
all:
	mkdir -p $(PACKAGE)
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build -trimpath -buildvcs=false -ldflags='-s -w' -o $(PACKAGE)/moonwing .
	cp app.json icon.png $(PACKAGE)/

package: all
	mkdir -p $(SOURCE_PACKAGE)/artwork
	cp $(SOURCES) $(SOURCE_PACKAGE)/
	@cp $(ARTWORK) $(SOURCE_PACKAGE)/artwork/
	@cd dist && zip -q -FS moonwing-compas.zip moonwing/moonwing moonwing/app.json moonwing/icon.png $(addprefix moonwing-source/,$(SOURCES) $(ARTWORK))

debug-norotation:
	mkdir -p $(DEBUG_PACKAGE)
	CGO_ENABLED=0 GOOS=linux GOARCH=mipsle GOMIPS=softfloat $(GO) build -tags=norotation -trimpath -buildvcs=false -ldflags='-s -w' -o $(DEBUG_PACKAGE)/moonwing .
	cp icon.png $(DEBUG_PACKAGE)/
	sed -e 's/"version": "\([^"]*\)"/"version": "\1-norotate"/' -e 's/"name": "Moonwing"/"name": "Moonwing (No Rotation)"/' app.json > $(DEBUG_PACKAGE)/app.json
	@cd dist/debug-norotation && zip -q -FS ../moonwing-compas-norotation.zip moonwing/moonwing moonwing/app.json moonwing/icon.png

test:
	$(GO) test ./...
