TAG = dev
IMG ?= quay.io/xenitab/ingress-healthz:$(TAG)

lint:
	golangci-lint run -E misspell

fmt:
	go fmt ./...

vet:
	go vet ./...

test: fmt vet
	go test -timeout 1m ./...

prep-git2go:
	# NOT TESTED YET
	go get -d github.com/libgit2/git2go 
	GOPATH=$$(go env GOPATH)
	cd $GOPATH/src/github.com/libgit2/git2go
	git submodule update --init
	make install-static

build:
	go build -tags static -a -o bin/git-proxy cmd/git-proxy/main.go
	#go build -a -o bin/go-git-client cmd/go-git-client/main.go

docker-build:
	docker build -t $(IMG) .