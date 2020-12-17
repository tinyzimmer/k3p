DIST ?= $(CURDIR)/dist
BIN ?= $(DIST)/k3p
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin
GOOS ?= linux
CGO_ENABLED ?= 0

GOLANGCI_VERSION ?= v1.33.0
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GINKGO ?= $(GOBIN)/ginkgo
GOX ?= $(GOBIN)/gox

LDFLAGS ?= "-X github.com/tinyzimmer/k3p/pkg/build/package/v1.ZstDictionaryB64=`cat ../../hack/zstDictionary | base64 --wrap=0` \
			-X github.com/tinyzimmer/k3p/pkg/version.K3pVersion=`git describe --tags` \
			-X github.com/tinyzimmer/k3p/pkg/version.K3pCommit=`git rev-parse HEAD`"

# Builds the k3p binary
build: $(BIN)

$(GOX):
	GO111MODULE=off go get github.com/mitchellh/gox

.PHONY: dist
COMP_TARGETS ?= "darwin/amd64 linux/amd64 linux/arm linux/arm64 windows/amd64"
COMP_OUTPUT ?= "$(DIST)/{{.Dir}}_{{.OS}}_{{.Arch}}"
dist: $(GOX)
	cd cmd/k3p && \
		CGO_ENABLED=0 $(GOX) -osarch $(COMP_TARGETS) --output $(COMP_OUTPUT) -ldflags=$(LDFLAGS)
	which upx 2> /dev/null && upx $(DIST)/*


install: $(BIN)
	mkdir -p $(GOBIN)
	cp $(BIN) $(GOBIN)/k3p

$(BIN):
	cd cmd/k3p && \
		CGO_ENABLED=$(CGO_ENABLED) GOOS=$(GOOS) \
		go build -o $(BIN) \
			-ldflags $(LDFLAGS)
	which upx 2> /dev/null && upx $(BIN)

docs:
	go run hack/docgen.go

# Cleans binaries and packages from the repo
clean:
	find . -name *.coverprofile -exec rm {} \;
	rm -rf $(DIST)/

# Linting
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_VERSION)

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v --timeout 300s

$(GINKGO):
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo

TEST_PKG ?= ./...
TEST_FLAGS ?=
test: $(GINKGO)
	$(GINKGO) \
		-cover -coverprofile=k3p.coverprofile -outputdir=. -coverpkg=$(TEST_PKG) \
		$(TEST_FLAGS) $(TEST_PKG)
	go tool cover -func k3p.coverprofile