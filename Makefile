build:
	cd cmd/k3p && \
		go build -o ../../dist/k3p .

clean:
	rm -rf dist package.tar