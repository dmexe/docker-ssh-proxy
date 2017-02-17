GOVENDOR := $(shell pwd)/bin/govendor
DAEMON   := $(shell pwd)/bin/daemon
ID_RSA   := $(shell pwd)/bin/id_rsa
PACKAGES := daemon daemon/handlers daemon/payloads daemon/sshd

all: build

fmt:
	bin/env go fmt $(PACKAGES)

vet:
	bin/env go vet $(PACKAGES)

lint:
	bin/env golint $(PACKAGES)

build: fmt vet lint
	bin/env go build -race -o $(DAEMON) daemon

test: fmt vet
	bin/env go test -race -timeout 1m -v $(PACKAGES)

run: all $(ID_RSA)
	bin/daemon -k $(ID_RSA) -d

deps:
	bin/install-deps
	bin/env -c src/daemon $(GOVENDOR) sync

pkg.list:
	bin/install-deps
	bin/env -c src/daemon $(GOVENDOR) list

pkg.remove.unused:
	bin/install-deps
	bin/env -c src/daemon $(GOVENDOR) remove +unused

pkg.fetch.missing:
	bin/install-deps
	bin/env -c src/daemon $(GOVENDOR) fetch -v +missing

$(ID_RSA):
	ssh-keygen -t rsa -P '' -C '' -f $(ID_RSA)
