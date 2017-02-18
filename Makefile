GOVENDOR := bin/env -c src/daemon $(CURDIR)/bin/govendor
DAEMON   := $(CURDIR)/bin/daemon
ID_RSA   := $(CURDIR)/bin/id_rsa
PACKAGES =  $(shell $(GOVENDOR) list -no-status +local)
GOARCH   =  $(shell go env GOARCH)


all: build.dev

fmt:
	bin/env go fmt $(PACKAGES)

vet:
	bin/env go vet $(PACKAGES)

lint:
	bin/env golint -set_exit_status $(PACKAGES)

test: fmt vet lint
	bin/env go test -cover -race -timeout 1m -v $(PACKAGES)

test.convey:
	bin/env -c src/daemon goconvey

build.dev: fmt vet lint
	bin/env go build -race -v -o $(DAEMON) daemon

build.release:
	GOOS=linux  bin/env go build -ldflags "-s -w" -o $(DAEMON)-linux-$(GOARCH)  daemon
	GOOS=darwin bin/env go build -ldflags "-s -w" -o $(DAEMON)-darwin-$(GOARCH) daemon

run: build.dev $(ID_RSA)
	bin/daemon -sshd.pkey $(ID_RSA) -debug

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
