FROM golang:1.15 as builder
WORKDIR /workspace

COPY scripts .
RUN apt-get update && apt-get -y install cmake libssl-dev
RUN ./install_libgit2.sh

COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=linux GOARCH=amd64 GO111MODULE=on go build -tags static,system_libgit2 -a -o git-proxy cmd/git-proxy/main.go

FROM debian:buster-slim
WORKDIR /app/
COPY --from=builder /workspace/git-proxy .

RUN apt-get update && \
    apt-get install -y git && \
    rm -rf /var/lib/apt/lists/*

RUN groupadd -r user && useradd -r -g user user

USER user

ENTRYPOINT ["/app/git-proxy"]