BIN ?= dist/k3p
PACKAGE ?= dist/package.tar
GOPATH ?= $(shell go env GOPATH)
GOLANGCI_LINT ?= $(GOPATH)/bin/golangci-lint

# Builds the k3p binary
build: $(BIN)

# Runs k3p build with any extra arguments
pkg: $(PACKAGE)

$(BIN):
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o ../../$(BIN) .

PKG_ARGS ?=
$(PACKAGE): $(BIN)
	$(BIN) build -o $(PACKAGE) $(PKG_ARGS)

# Cleans binaries and packages from the repo
clean:
	rm -rf dist/

# Linting
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin v1.33.0

lint: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) run -v

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

# Uninstall the k3s server from the node
clean-server-all: clean-server-1 clean-server-2 clean-server-3
clean-server-%:
	$(MAKE) node-shell-$* CMD=k3s-uninstall.sh

# Uninstall the k3s agent from the node
clean-agent-all: clean-agent-1 clean-agent-2 clean-agent-3
clean-agent-%:
	$(MAKE) node-shell-$* CMD=k3s-agent-uninstall.sh
