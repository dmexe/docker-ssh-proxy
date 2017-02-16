GOVENDOR := $(shell pwd)/bin/govendor
DAEMON   := $(shell pwd)/bin/daemon
ID_RSA   := $(shell pwd)/bin/id_rsa

export GOPATH := $(shell pwd)

.PHONY: all build fmt vet run deps

all: build

fmt:
	go fmt daemon

build: fmt vet
	go build -o $(DAEMON) daemon

vet:
	go vet daemon

test:
	go test -v daemon/payload

run: all $(ID_RSA)
	bin/daemon -k $(ID_RSA) -d

$(ID_RSA):
	ssh-keygen -t rsa -P '' -C '' -f $(ID_RSA)

$(GOVENDOR):
	go get -u github.com/kardianos/govendor

deps: $(GOVENDOR)
	(cd src/daemon ; $(GOVENDOR) fetch +missing)
