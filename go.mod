module github.com/xenitab/git-proxy

go 1.15

require (
	github.com/go-git/go-git/v5 v5.2.0
	github.com/libgit2/git2go/v31 v31.3.0
	github.com/nulab/go-git-http-xfer v1.3.2
	github.com/stretchr/testify v1.6.1 // indirect
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897 // indirect
)

// replace github.com/libgit2/git2go/v31 => ../../../../go/src/github.com/libgit2/git2go
