GOVENDOR := $(shell pwd)/bin/govendor
DAEMON   := $(shell pwd)/bin/daemon
ID_RSA   := $(shell pwd)/bin/id_rsa
PACKAGES := daemon daemon/agent daemon/payload daemon/utils

export GOPATH := $(shell pwd)

.PHONY: all build fmt vet run deps

all: build

fmt:
	go fmt $(PACKAGES)

build: fmt vet
	go build -o $(DAEMON) daemon

vet:
	go vet $(PACKAGES)

test:
	go test -p 2 -v $(PACKAGES)

run: all $(ID_RSA)
	bin/daemon -k $(ID_RSA) -d

$(ID_RSA):
	ssh-keygen -t rsa -P '' -C '' -f $(ID_RSA)

deps:
	bin/install-deps
	(cd src/daemon ; $(GOVENDOR) sync)

pkg.list:
	bin/install-deps
	(cd src/daemon ; $(GOVENDOR) list)

pkg.clean:
	bin/install-deps
	(cd src/daemon ; $(GOVENDOR) remove +unused)

