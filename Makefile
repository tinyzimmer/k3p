build:
	cd cmd/k3p && \
		CGO_ENABLED=0 GOOS=linux go build -o ../../dist/k3p .

clean:
	rm -rf dist package.tar