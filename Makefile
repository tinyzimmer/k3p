build:
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o ../../dist/k3p .

clean:
	rm -rf dist package.tar

# I use these locally to distribute packages and binaries across nodes
# and other playing around
REMOTE_USER ?= core
REMOTE_HOSTS ?= 172.17.113.136 172.17.113.137 172.17.113.130

dist-local: build
	@ - $(foreach HOST,$(REMOTE_HOSTS), \
		scp dist/k3p package.tar $(REMOTE_USER)@$(HOST):~/ ; \
		ssh $(REMOTE_USER)@$(HOST) "sudo rm -rf /usr/local/bin/k3p && sudo mv /var/home/core/k3p /usr/local/bin/k3p && sudo chmod +x /usr/local/bin/k3p" ; \
	)

CMD ?=
shell-node-%:
	ssh $(shell sed '$*q;d' hack/hosts.txt) $(CMD)

clean-node-all: clean-node-1 clean-node-2 clean-node-3

clean-node-%:
	$(MAKE) shell-node-$* CMD=k3s-uninstall.sh
