package main

import (
	"context"
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
	"github.com/gorilla/mux"
	"github.com/ldez/go-git-cmd-wrapper/v2/clone"
	"github.com/ldez/go-git-cmd-wrapper/v2/fetch"
	"github.com/ldez/go-git-cmd-wrapper/v2/git"
	"github.com/ldez/go-git-cmd-wrapper/v2/reset"
	"github.com/ldez/go-git-cmd-wrapper/v2/types"
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
	gitProxy := ProxyMiddleware(ghx, repoPath, log.WithName("proxy"))

	router := mux.NewRouter()
	router.HandleFunc("/readyz", readinessHandler(log.WithName("readiness"))).Methods("GET")
	router.HandleFunc("/healthz", livenessHandler(log.WithName("liveness"))).Methods("GET")
	router.PathPrefix("/").HandlerFunc(gitProxy)

	srv := &http.Server{Addr: ":" + strconv.Itoa(port), Handler: router}

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

func readinessHandler(log logr.Logger) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func livenessHandler(log logr.Logger) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte("{\"status\": \"ok\"}")); err != nil {
			log.Error(err, "Could not write response data")
		}
	}
}

func ProxyMiddleware(next http.Handler, repoPath string, log logr.Logger) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		azdoDomain := "dev.azure.com"
		azdoOrg := strings.Split(r.URL.Path, "/")[1]
		azdoProj := strings.Split(r.URL.Path, "/")[2]
		azdoRepo := strings.Split(r.URL.Path, "/")[4]
		repoUri := "https://" + azdoDomain + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		localPath := repoPath + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		_, err := git.Clone(clone.Repository(repoUri), clone.Directory(localPath))
		if err != nil {
			_, err = git.Reset(GitDir("reset", localPath), reset.Hard)
			if err != nil {
				log.Error(err, "Error running git reset")
				http.NotFound(w, r)
			}
			_, err = git.Pull(GitDir("pull", localPath))
			if err != nil {
				log.Error(err, "Error running git pull")
				http.NotFound(w, r)
			}
			_, err = git.Fetch(GitDir("fetch", localPath), fetch.All, fetch.Tags)
			if err != nil {
				log.Error(err, "Error git fetch.")
				http.NotFound(w, r)
			}

		}
		next.ServeHTTP(w, r)
	}
}

func GitDir(command string, localPath string) func(g *types.Cmd) {
	return func(g *types.Cmd) {
		g.Options = []string{
			"--git-dir",
			localPath + "/.git",
			command,
		}
	}
}
