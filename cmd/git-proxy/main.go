package main

import (
	"errors"
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
			Bare:           false,
			CheckoutBranch: "master",
		}
		_, err := git.Clone(repoUri, localPath, cloneOptions)
		if err != nil {
			err := PullBranch(localPath, "origin", "master", "", "", "TEST-NAME", "test-email@example.com")
			if err != nil {
				log.Printf("Error pulling branch: %s", err)
			}

		}
		next.ServeHTTP(w, r)
	})
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
