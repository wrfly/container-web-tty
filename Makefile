.PHONY: build test dev

NAME = container-web-tty
PKG = github.com/wrfly/$(NAME)
BIN = bin

VERSION := $(shell cat VERSION)
COMMITID := $(shell git rev-parse --short HEAD)
BUILDAT := $(shell date +%Y-%m-%d)

CTIMEVAR = -X main.CommitID=$(COMMITID) \
	-X main.Version=$(VERSION) \
	-X main.BuildAt=$(BUILDAT)
GO_LDFLAGS = -ldflags "-w $(CTIMEVAR)"
GO_LDFLAGS_STATIC = -ldflags "-w $(CTIMEVAR) -extldflags -static"

.PHONY: bin
bin:
	mkdir -p bin

.PHONY: build
build: bin
	go build $(GO_LDFLAGS) -o $(BIN)/$(NAME) .

.PHONY: test
test:
	go test --cover -v `glide nv`

.PHONY: dev
dev: asset build
	./$(BIN)/$(NAME) -l debug

.PHONY: release
release:
	GOOS=linux GOARCH=amd64 go build $(GO_LDFLAGS) -o $(BIN)/$(NAME)_linux_amd64 .
	GOOS=darwin GOARCH=amd64 go build $(GO_LDFLAGS) -o $(BIN)/$(NAME)_darwin_amd64 .

.PHONY: asset
asset: bindata/static/js/gotty-bundle.js bindata/static/index.html bindata/static/favicon.png bindata/static/css/index.css bindata/static/css/xterm.css bindata/static/css/xterm_customize.css
	go-bindata -prefix bindata -pkg route -ignore=\\.gitkeep -o route/asset.go bindata/...
	gofmt -w route/asset.go

.PHONY: all
all: asset gotty

bindata:
	mkdir bindata

bindata/static: bindata
	mkdir bindata/static

bindata/static/index.html: bindata/static resources/index.html resources/list.html
	cp resources/index.html bindata/static/index.html
	cp resources/list.html bindata/static/list.html

bindata/static/favicon.png: bindata/static resources/favicon.png
	cp resources/favicon.png bindata/static/favicon.png

bindata/static/js: bindata/static
	mkdir -p bindata/static/js


bindata/static/js/gotty-bundle.js: bindata/static/js js/dist/gotty-bundle.js
	cp js/dist/gotty-bundle.js bindata/static/js/gotty-bundle.js

bindata/static/css: bindata/static
	mkdir -p bindata/static/css

bindata/static/css/index.css: bindata/static/css resources/index.css resources/list.css
	cp resources/index.css bindata/static/css/index.css
	cp resources/list.css bindata/static/css/list.css

bindata/static/css/xterm_customize.css: bindata/static/css resources/xterm_customize.css
	cp resources/xterm_customize.css bindata/static/css/xterm_customize.css

bindata/static/css/xterm.css: bindata/static/css js/node_modules/xterm/dist/xterm.css
	cp js/node_modules/xterm/dist/xterm.css bindata/static/css/xterm.css

js/node_modules/xterm/dist/xterm.css:
	cd js && \
	npm install

js/dist/gotty-bundle.js: js/src/* js/node_modules/webpack
	cd js && \
	`npm bin`/webpack

js/node_modules/webpack:
	cd js && \
	npm install


tools:
	go get github.com/jteeuwen/go-bindata/...
