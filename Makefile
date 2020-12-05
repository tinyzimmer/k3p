DIST ?= $(CURDIR)/dist
BIN ?= $(DIST)/k3p
PACKAGE ?= $(DIST)/package.tar
GOPATH ?= $(shell go env GOPATH)
GOBIN ?= $(GOPATH)/bin

GOLANGCI_VERSION ?= v1.33.0
GOLANGCI_LINT ?= $(GOBIN)/golangci-lint
GINKGO ?= $(GOBIN)/ginkgo

# Builds the k3p binary
build: $(BIN)

# Runs k3p build with any extra arguments
pkg: $(PACKAGE)

$(BIN):
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o $(BIN) .

PKG_ARGS ?=
$(PACKAGE): $(BIN)
	$(BIN) build -o $(PACKAGE) $(PKG_ARGS)

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

# I use these locally to distribute packages and binaries across nodes
# in my lab.
#
# However, if you have pubkey authentication setup, you can override the NODE_USER
# and NODES variables in your environment and probably make use of these functions
# as well.
NODE_USER ?= core
NODES ?= 172.17.113.136 172.17.113.137 172.17.113.130

# Gets node by index: Usage $(call get-node,1)
get-node = $(word $1,$(NODES))

# Distribute the k3p binary and package.tar to the nodes
dist-node-all: dist-node-1 dist-node-2 dist-node-3
dist-node-%: $(BIN) $(PACKAGE)
	scp -r dist/ $(NODE_USER)@$(call get-node,$*):~/
	$(MAKE) node-shell-$* CMD="sudo rm -rf /usr/local/bin/k3p && sudo mv /var/home/core/dist/k3p /usr/local/bin/k3p && sudo chmod +x /usr/local/bin/k3p"

# Get a bash shell on one of the nodes
CMD ?=
node-shell-%:
	ssh $(NODE_USER)@$(call get-node,$*) "$(CMD)"

kubeconfig:
	@ssh $(NODE_USER)@$(call get-node,1) sudo cat /etc/rancher/k3s/k3s.yaml | sed 's/127.0.0.1/$(call get-node,1)/'

deploy: $(BIN) $(PACKAGE) dist-node-1
	$(MAKE) node-shell-1 CMD="sudo k3p install dist/package.tar"

ha-cluster: $(BIN) $(PACKAGE) dist-node-1
	$(MAKE) node-shell-1 CMD="sudo k3p install dist/package.tar --init-ha -v"
	sleep 7
	$(MAKE) node-shell-1 CMD="sudo k3p node add $(call get-node,2) --node-role=server --ssh-user=core --private-key=/var/home/core/.ssh/id_rsa -v"
	$(MAKE) node-shell-1 CMD="sudo k3p node add $(call get-node,3) --node-role=server --ssh-user=core --private-key=/var/home/core/.ssh/id_rsa -v"

# Uninstall the k3s server from the node
clean-all-servers: clean-server-1 clean-server-2 clean-server-3
clean-server-%:
	$(MAKE) node-shell-$* CMD=k3s-uninstall.sh

# Uninstall the k3s agent from the node
clean-all-agents: clean-agent-1 clean-agent-2 clean-agent-3
clean-agent-%:
	$(MAKE) node-shell-$* CMD=k3s-agent-uninstall.sh
