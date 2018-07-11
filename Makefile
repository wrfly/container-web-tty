.PHONY: build test dev

NAME = container-web-tty
PKG = github.com/wrfly/$(NAME)
BIN = bin
IMAGE := wrfly/$(NAME)

VERSION := $(shell cat VERSION)
COMMITID := $(shell git rev-parse --short HEAD)
BUILDAT := $(shell date +%Y-%m-%d)

CTIMEVAR = -X main.CommitID=$(COMMITID) \
	-X main.Version=$(VERSION) \
	-X main.BuildAt=$(BUILDAT)
GO_LDFLAGS = -ldflags "-s -w $(CTIMEVAR)"
GO_LDFLAGS_STATIC = -ldflags "-w $(CTIMEVAR) -extldflags -static"

.PHONY: prepare
prepare:
	glide i

.PHONY: bin
bin:
	mkdir -p bin

.PHONY: glide-up
glide-up:
	https_proxy=http://127.0.0.1:1081 glide up

.PHONY: build
build: bin
	go build $(GO_LDFLAGS) -o $(BIN)/$(NAME) .

.PHONY: test
test:
	go test -cover -v `glide nv`

.PHONY: dev
dev: asset build
	./$(BIN)/$(NAME) -l debug

.PHONY: release
release:
	GOOS=linux GOARCH=amd64 go build $(GO_LDFLAGS) -o $(BIN)/$(NAME)_linux_amd64 .
	GOOS=darwin GOARCH=amd64 go build $(GO_LDFLAGS) -o $(BIN)/$(NAME)_darwin_amd64 .

.PHONY: image
image:
	docker build -t $(IMAGE) .

.PHONY: push-image
push-image:
	docker push $(IMAGE)


.PHONY: push-develop
push-develop:
	docker tag $(IMAGE) $(IMAGE):develop
	docker push $(IMAGE):develop

.PHONY: push-tag
push-tag:
	docker tag $(IMAGE) $(IMAGE):$(VERSION)
	docker push $(IMAGE):$(VERSION)

## --- these stages are copied from gotty for asset building --- ##
.PHONY: asset
asset: clear static/js/gotty-bundle.js static/index.html static/favicon.png static/css/index.css static/css/xterm.css static/css/xterm_customize.css
	go-bindata -prefix static -pkg route -ignore=\\.gitkeep -o route/asset.go static/...
	gofmt -w route/asset.go

clear:
	rm -rf static

static:
	mkdir -p static

static/index.html: static resources/index.html resources/list.html
	cp resources/index.html static/index.html
	cp resources/list.html static/list.html

static/favicon.png: static resources/favicon.png
	cp resources/favicon.png static/favicon.png

static/js: static
	mkdir -p static/js

static/js/gotty-bundle.js: static/js js/dist/gotty-bundle.js
	cp js/dist/gotty-bundle.js static/js/gotty-bundle.js
	cp resources/control.js static/js/control.js

static/css: static
	mkdir -p static/css

static/css/index.css: static/css resources/index.css resources/list.css
	cp resources/index.css static/css/index.css
	cp resources/list.css static/css/list.css

static/css/xterm_customize.css: static/css resources/xterm_customize.css
	cp resources/xterm_customize.css static/css/xterm_customize.css

static/css/xterm.css: static/css js/node_modules/xterm/dist/xterm.css
	cp js/node_modules/xterm/dist/xterm.css static/css/xterm.css

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
