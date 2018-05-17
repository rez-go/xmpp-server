
PKG_PATH = github.com/exavolt/xmpp-server
DEP_IMAGE ?= exavolt/xmpp-server/dep
GOLANG_IMAGE ?= golang:1.10

.PHONY: run fmt update-dependencies

run:
	docker-compose up --build

fmt:
	docker run --rm \
		-v $(CURDIR):/go \
		--entrypoint gofmt \
		$(GOLANG_IMAGE) -w -l -s ./cmd

update-dependencies:
	docker build -t $(DEP_IMAGE) -f ./tools/dep.dockerfile .
	docker run --rm \
		-v $(CURDIR):/go/src/$(PKG_PATH) \
		--workdir /go/src/$(PKG_PATH) \
		$(DEP_IMAGE) ensure -update -v
