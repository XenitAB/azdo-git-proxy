package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	git "github.com/libgit2/git2go/v31"
	"github.com/nulab/go-git-http-xfer/githttpxfer"
	flag "github.com/spf13/pflag"
	"go.uber.org/zap"
)

var (
	port     int
	gitBin   string
	repoPath string
)

func init() {
	flag.IntVar(&port, "port", 8080, "port to bind server to.")
	flag.StringVar(&gitBin, "git-binary-path", "/usr/bin/git", "path to git binary.")
	flag.StringVar(&repoPath, "repository-path", "/tmp/repos", "path to store repositories.")
	flag.Parse()
}

func main() {
	// Initiate logs
	var log logr.Logger
	zapLog, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)
	setupLog := log.WithName("setup")

	// Signal handler
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Setup GitHTTPXfer
	setupLog.Info("Starting azdo-git-proxy", "gitBin", gitBin, "port", port, "repoPath", repoPath)
	ghx, err := githttpxfer.New(repoPath, gitBin)
	if err != nil {
		setupLog.Error(err, "GitHTTPXfer instance could not be created.")
		os.Exit(1)
	}

	handler := ProxyMiddleware(ghx, repoPath)
	srv := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: handler}

	// Start HTTP server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			setupLog.Error(err, "Http Server Error")
		}
	}()
	setupLog.Info("Server started")

	// Blocks until singal is sent
	<-done
	setupLog.Info("Server stopped")

	// Shutdown http server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	if err := srv.Shutdown(ctx); err != nil {
		setupLog.Error(err, "Server shutdown failed")
		os.Exit(1)
	}

	setupLog.Info("Server exited properly")
}

func ProxyMiddleware(next http.Handler, repoPath string) http.Handler {
	// Initiate logs
	var log logr.Logger
	zapLog, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("who watches the watchmen (%v)?", err))
	}
	log = zapr.NewLogger(zapLog)
	proxyLog := log.WithName("proxy")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		azdoDomain := "dev.azure.com"
		azdoOrg := strings.Split(r.URL.Path, "/")[1]
		azdoProj := strings.Split(r.URL.Path, "/")[2]
		azdoRepo := strings.Split(r.URL.Path, "/")[4]
		repoUri := "https://" + azdoDomain + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		localPath := repoPath + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		cloneOptions := &git.CloneOptions{
			Bare:           false,
			CheckoutBranch: "master",
		}
		_, err := git.Clone(repoUri, localPath, cloneOptions)
		if err != nil {
			err := PullBranch(localPath, "origin", "master", "", "", "TEST-NAME", "test-email@example.com")
			if err != nil {
				proxyLog.Error(err, "Error pulling branch.")
			}

		}
		next.ServeHTTP(w, r)
	})

	return handler
}

func PullBranch(repoPath string, remoteName string, branchName string, user string, pass string, name string, email string) error {

	repo, err := git.OpenRepository(repoPath)
	if err != nil {
		return err
	}

	remote, err := repo.Remotes.Lookup(remoteName)
	if err != nil {
		return err
	}

	err = remote.Fetch([]string{}, &git.FetchOptions{}, "")

	if err != nil {
		return err
	}

	remoteBranch, err := repo.References.Lookup("refs/remotes/" + remoteName + "/" + branchName)
	if err != nil {
		return err
	}

	mergeRemoteHead, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return err
	}

	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = mergeRemoteHead
	if err = repo.Merge(mergeHeads, nil, nil); err != nil {
		return err
	}

	// Check if the index has conflicts after the merge
	idx, err := repo.Index()
	if err != nil {
		return err
	}

	currentBranch, err := repo.Head()
	if err != nil {
		return err
	}

	localCommit, err := repo.LookupCommit(currentBranch.Target())
	if err != nil {
		return err
	}

	// If index has conflicts, read old tree into index and
	// return an error.
	if idx.HasConflicts() {

		repo.ResetToCommit(localCommit, git.ResetHard, &git.CheckoutOpts{})

		repo.StateCleanup()

		return errors.New("conflict")
	}

	// If everything looks fine, create a commit with the two parents
	treeID, err := idx.WriteTree()
	if err != nil {
		return err
	}

	tree, err := repo.LookupTree(treeID)
	if err != nil {
		return err
	}

	remoteCommit, err := repo.LookupCommit(remoteBranch.Target())
	if err != nil {
		return err
	}

	sig := &git.Signature{Name: name, Email: email, When: time.Now()}
	_, err = repo.CreateCommit("HEAD", sig, sig, "merged", tree, localCommit, remoteCommit)
	if err != nil {
		return err
	}

	repo.StateCleanup()

	return nil
}
