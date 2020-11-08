package main

import (
	"log"
	"net/http"
	"strings"
	"time"

	git "github.com/libgit2/git2go/v31"
	"github.com/nulab/go-git-http-xfer/githttpxfer"
)

func main() {
	repoPath := "/tmp/repos"
	gitBin := "/usr/local/bin/git"

	// remoteGit := "https://dev.azure.com/simongottschlag/test/_git/test"

	ghx, err := githttpxfer.New(repoPath, gitBin)
	if err != nil {
		log.Fatalf("GitHTTPXfer instance could not be created. %s", err.Error())
		return
	}

	handler := Logging(ghx)
	handler = ProxyMiddleware(handler, repoPath)

	if err := http.ListenAndServe(":5050", handler); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	})
}

func ProxyMiddleware(next http.Handler, repoPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		azdoDomain := "dev.azure.com"
		azdoOrg := strings.Split(r.URL.Path, "/")[1]
		azdoProj := strings.Split(r.URL.Path, "/")[2]
		azdoRepo := strings.Split(r.URL.Path, "/")[4]
		repoUri := "https://" + azdoDomain + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		localPath := repoPath + "/" + azdoOrg + "/" + azdoProj + "/_git/" + azdoRepo
		log.Printf("Path: %s", r.URL.Path)
		log.Printf("Query: %s", r.URL.RawQuery)
		log.Printf("Organization: %s", azdoOrg)
		log.Printf("Project: %s", azdoProj)
		log.Printf("Repository: %s", azdoRepo)
		log.Printf("repoUri: %s", repoUri)
		cloneOptions := &git.CloneOptions{
			Bare:           true,
			CheckoutBranch: "master",
		}
		repo, err := git.Clone(repoUri, localPath, cloneOptions)
		if err != nil {
			log.Printf("LOCAL REPO ALREADY EXISTS - FETCH INSTEAD: %s", err)
			remote, err := repo.Remotes.Lookup("origin")
			if err != nil {
				log.Printf("Unable to locate origin: %s", err)
			}
			if err := remote.Fetch([]string{}, nil, ""); err != nil {
				log.Printf("Unable to fetch remote: %s", err)
			}

		}
		next.ServeHTTP(w, r)
	})
}
