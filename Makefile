GOVENDOR := $(shell pwd)/bin/govendor
DAEMON   := $(shell pwd)/bin/daemon
ID_RSA   := $(shell pwd)/bin/id_rsa
PACKAGES := daemon daemon/handlers daemon/payloads daemon/sshd

export GOPATH := $(shell pwd)

all: build

fmt:
	go fmt $(PACKAGES)

build: fmt vet
	go build -o $(DAEMON) daemon

vet:
	go vet $(PACKAGES)

test: fmt vet
	go test -timeout 1m -v $(PACKAGES) && echo "\nall ok."

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

pkg.remove.unused:
	bin/install-deps
	(cd src/daemon ; $(GOVENDOR) remove +unused)

pkg.fetch.missing:
	bin/install-deps
	(cd src/daemon ; $(GOVENDOR) fetch +missing)
