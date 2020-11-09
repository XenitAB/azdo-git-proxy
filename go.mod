module github.com/xenitab/git-proxy

go 1.15

require (
	github.com/go-git/go-git/v5 v5.2.0
	github.com/go-logr/logr v0.3.0 // indirect
	github.com/go-logr/zapr v0.3.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/libgit2/git2go/v31 v31.3.0
	github.com/nulab/go-git-http-xfer v1.3.2
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/sosedoff/gitkit v0.2.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/testify v1.6.1 // indirect
	go.uber.org/zap v1.16.0 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
)

replace github.com/libgit2/git2go/v31 => ../../../../go/src/github.com/libgit2/git2go
