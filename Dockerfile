FROM golang:1.15 as builder

RUN mkdir -p /workspace \
    && apt-get update  \
    && apt-get install -y upx

WORKDIR /workspace

COPY go.mod /workspace/go.mod
COPY go.sum /workspace/go.sum

RUN go mod download

COPY . /workspace/

RUN make

# Make a small alpine image
FROM alpine:latest

# Create a default working directory
RUN mkdir -p /manifests
WORKDIR /manifests

# Copy the binary over
COPY --from=builder /workspace/dist/k3p /usr/local/bin/k3p
ENTRYPOINT [ "/usr/local/bin/k3p" ]
