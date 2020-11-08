FROM golang:1.15 as builder
WORKDIR /workspace

# libgit2 / git2go

RUN apt-get update && apt-get -q -y install \
    git openssl apt-transport-https ca-certificates curl g++ gcc libc6-dev make pkg-config \
    libssl-dev libssh2-1-dev cmake

RUN go get -d github.com/libgit2/git2go && \
    cd $GOPATH/src/github.com/libgit2/git2go && \
    git checkout tags/v31.3.0 && \
    git submodule update --init && \
    make install-static

# git-proxy

COPY go.mod go.mod
RUN go mod download
COPY cmd/ cmd/
RUN GOOS=linux GOARCH=amd64 GO111MODULE=on go build -tags static -a -o git-proxy cmd/git-proxy/main.go
RUN mkdir -p /tmp/repos
ENTRYPOINT ["/workspace/git-proxy"]

# runtime

# FROM gcr.io/distroless/static:nonroot as runtime
# WORKDIR /
# COPY --from=builder /workspace/git-proxy .
# USER nonroot:nonroot
# ENTRYPOINT ["/git-proxy"]