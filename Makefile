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

.PHONY: asset
asset:
	cd gotty && make asset && cd ..

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