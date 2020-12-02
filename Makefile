build:
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o ../../dist/k3p .

clean:
	rm -rf dist package.tar

# I use these locally to distribute packages and binaries across nodes
# and other playing around
dist-local: build
	@while IFS= read -r dest ; do \
		echo scp dist/k3p package.tar "$$dest:~/" ; \
		scp dist/k3p package.tar "$$dest:~/" ; \
		echo ; \
	done < hack/hosts.txt

CMD ?=
shell-node-%:
	ssh $(shell sed '$*q;d' hack/hosts.txt) $(CMD)

clean-node-%:
	$(MAKE) shell-node-$* CMD=k3s-uninstall.sh

clean-node-all: clean-node-1 clean-node-2 clean-node-3