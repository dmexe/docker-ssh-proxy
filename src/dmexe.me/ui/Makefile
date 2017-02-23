YARN         := yarn
NODE_MODULES := $(CURDIR)/node_modules
WEBPACK      := $(YARN) run webpack --
DEVSERVER    := $(YARN) run webpack-dev-server --

export NODE_PATH=$(NODE_MODULES)

all: dev.server

dev:
	$(WEBPACK) --config build/dev.js --progress

dev.server:
	$(DEVSERVER)  --config build/dev.js --hot

watch:
	$(WEBPACK) --config build/dev.js --progress --watch

release:
	$(WEBPACK) --config build/release.js -p --progress
