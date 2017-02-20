MAIN     := daemon
BIN      := $(CURDIR)/bin/daemon
GOVENDOR := bin/env -c src/$(MAIN) $(CURDIR)/bin/govendor
ID_RSA   := $(CURDIR)/bin/id_rsa
GO       := bin/env go
GOLINT   := bin/env golint
PACKAGES =  $(shell $(GOVENDOR) list -no-status +local)

all: build.dev

debug:
	echo $(OS)

fmt:
	$(GO) fmt $(PACKAGES)

build.cross:
	GOOS=darwin go build $(RELEASE_FLAGS) -o $(DAEMON)-darwin-$(GOARCH) daemon
	GOOS=linux go build $(RELEASE_FLAGS) -o $(DAEMON)-linux-$(GOARCH) daemon

vet:
	$(GO) vet $(PACKAGES)

lint:
	$(GOLINT) -set_exit_status $(PACKAGES)

test: fmt vet lint
	$(GO) test -cover -race -timeout 1m -v $(PACKAGES)

build.dev: fmt vet lint
	$(GO) build -race -v -o $(BIN) $(MAIN)

build.release:
	GOOS=linux  GOARCH=amd64 $(GO) build -ldflags "-s -w" -o $(BIN)-Linux-x86_64  $(MAIN)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-s -w" -o $(BIN)-Darwin-x86_64 $(MAIN)

run: build.dev $(ID_RSA)
	$(BIN) -sshd.pkey $(ID_RSA) -debug

deps:
	bin/install-deps
	$(GOVENDOR) sync

pkg.list:
	bin/install-deps
	$(GOVENDOR) list

pkg.remove.unused:
	bin/install-deps
	$(GOVENDOR) remove +unused

pkg.fetch.missing:
	bin/install-deps
	$(GOVENDOR) fetch -v +missing

$(ID_RSA):
	ssh-keygen -t rsa -P '' -C '' -f $(ID_RSA)
