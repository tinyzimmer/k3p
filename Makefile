DIST   ?= $(CURDIR)/dist
BIN    ?= $(DIST)/k3p
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(GOPATH)/bin

GOOS        ?= linux
CGO_ENABLED ?= 0

GOLANGCI_VERSION ?= v1.33.0
GOLANGCI_LINT    ?= $(GOBIN)/golangci-lint
GINKGO           ?= $(GOBIN)/ginkgo
GOX              ?= $(GOBIN)/gox
UPX              ?= $(shell which upx 2> /dev/null)

VERSION  ?= $(shell git describe --tags)
COMMIT   ?= $(shell git rev-parse HEAD)
ZST_DICT ?= $(CURDIR)/hack/zstDictionary

LDFLAGS ?= "-X github.com/tinyzimmer/k3p/pkg/build/package/v1.ZstDictionaryB64=`cat '$(ZST_DICT)' | base64 --wrap=0` \
			-X github.com/tinyzimmer/k3p/pkg/version.K3pVersion=$(VERSION) \
			-X github.com/tinyzimmer/k3p/pkg/version.K3pCommit=$(COMMIT) -s -w"

COMPRESSION ?= 5

build: $(BIN)

$(BIN):
	cd cmd/k3p && \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) \
		go build -o $(BIN) \
			-ldflags $(LDFLAGS)
ifneq ($(UPX),)
	$(UPX) -$(COMPRESSION) $(BIN)
endif

IMG ?= ghcr.io/tinyzimmer/k3p:$(shell git describe --tags)
docker:
	docker build . -t $(IMG)

$(GOX):
	GO111MODULE=off go get github.com/mitchellh/gox

.PHONY: dist
COMPILE_TARGETS ?= "darwin/amd64 linux/amd64 linux/arm linux/arm64 windows/amd64"
COMPILE_OUTPUT  ?= "$(DIST)/{{.Dir}}_{{.OS}}_{{.Arch}}"
dist: $(GOX)
	cd cmd/k3p && \
		CGO_ENABLED=$(CGO_ENABLED) $(GOX) -osarch $(COMPILE_TARGETS) --output $(COMPILE_OUTPUT) -ldflags=$(LDFLAGS)
ifneq ($(UPX),)
	$(UPX) -$(COMPRESSION) $(DIST)/*
endif


install: $(BIN)
	mkdir -p $(GOBIN)
	cp $(BIN) $(GOBIN)/k3p

docs:
	go run hack/docgen.go

clean:
	find . -name '*.coverprofile' -exec rm {} \;
	find . -name '*.tgz' -exec rm {} \;
	find . -name '*.tar' -exec rm {} \;
	find . -name '*.run' -exec rm {} \;
	rm -rf $(DIST) tls/

$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v --timeout 300s

$(GINKGO):
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo

TEST_PKG   ?= ./...
TEST_FLAGS ?=
test: $(GINKGO)
	$(GINKGO) \
		-cover -coverprofile=k3p.coverprofile -outputdir=. -coverpkg=$(TEST_PKG) \
		$(TEST_FLAGS) $(TEST_PKG)
	go tool cover -func k3p.coverprofile

tls:
	bash hack/gen-cert-chain.sh
	cat tls/intermediate-1.crt tls/ca.crt > tls/ca-bundle.crt

tls-args:
	@echo -n '--registry-tls-cert="$(CURDIR)/tls/leaf.crt" --registry-tls-key="$(CURDIR)/tls/leaf.key" --registry-tls-ca="$(CURDIR)/tls/ca-bundle.crt"'