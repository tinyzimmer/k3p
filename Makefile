DIST ?= $(CURDIR)/dist
BIN ?= $(DIST)/k3p
PACKAGE ?= $(DIST)/package.tar
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

GOLANGCI_VERSION ?= v1.33.0
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GINKGO ?= $(GOBIN)/ginkgo
GOX ?= $(GOBIN)/gox

# Builds the k3p binary
build: $(BIN)

$(GOX):
	GO111MODULE=off go get github.com/mitchellh/gox

.PHONY: dist
dist: $(GOX)
	cd cmd/k3p && \
		CGO_ENABLED=0 $(GOX) -os "linux windows" -arch "amd64 arm" --output "../../dist/{{.Dir}}_{{.OS}}_{{.Arch}}"
	cd cmd/k3p && \
		CGO_ENABLED=0 $(GOX) -os "darwin" -arch "amd64" --output "../../dist/{{.Dir}}_macOS_{{.Arch}}"


install: $(BIN)
	mkdir -p $(GOBIN)
	cp $(BIN) $(GOBIN)/k3p

# Runs k3p build with any extra arguments
pkg: $(PACKAGE)

$(BIN):
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o $(BIN) .

PKG_ARGS ?=
$(PACKAGE): $(BIN)
	$(BIN) build -o $(PACKAGE) --name k3p-test --manifests examples/whoami $(PKG_ARGS)

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
	$(GOLANGCI_LINT) run -v

$(GINKGO):
	GO111MODULE=off go get github.com/onsi/ginkgo/ginkgo

TEST_PKG ?= ./...
TEST_FLAGS ?=
test: $(GINKGO)
	$(GINKGO) \
		-cover -coverprofile=k3p.coverprofile -outputdir=. -coverpkg=$(TEST_PKG) \
		$(TEST_FLAGS) $(TEST_PKG)
	go tool cover -func k3p.coverprofile