MAIN     := daemon
BIN      := $(CURDIR)/bin/daemon
GOVENDOR := bin/env -c src/$(MAIN) $(CURDIR)/bin/govendor
ID_RSA   := $(CURDIR)/bin/id_rsa
GO       := bin/env go
GOLINT   := bin/env golint
PACKAGES =  $(shell $(GOVENDOR) list -no-status +local)

all: build.dev

check.fmt:
	$(GO) fmt $(PACKAGES)

check.vet:
	$(GO) vet $(PACKAGES)

check.lint:
	$(GOLINT) -set_exit_status $(PACKAGES)

check: check.fmt check.vet check.lint

test: check
	$(GO) test -cover -race -timeout 1m -v $(PACKAGES)

build.dev:
	$(GO) build -race -o $(BIN) $(MAIN)

build.release:
	GOOS=linux  GOARCH=amd64 $(GO) build -ldflags "-s -w" -o $(BIN)-Linux-x86_64  $(MAIN)
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "-s -w" -o $(BIN)-Darwin-x86_64 $(MAIN)

run: build.dev $(ID_RSA)
	$(BIN) -ssh -ssh.key $(ID_RSA) -api -api.marathon.url http://marathon.mesos:8080/v2 -debug

deps.install:
	bin/install-deps
	$(GOVENDOR) sync

deps.list:
	bin/install-deps
	$(GOVENDOR) list

deps.remove.unused:
	bin/install-deps
	$(GOVENDOR) remove +unused

deps.fetch.missing:
	bin/install-deps
	$(GOVENDOR) fetch -v +missing

$(ID_RSA):
	ssh-keygen -t rsa -P '' -C '' -f $(ID_RSA)
