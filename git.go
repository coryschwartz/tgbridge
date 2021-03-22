package main

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"gopkg.in/yaml.v2"
)

// clone a git repo into a randomized temporary directory
func GitClone(url string, checkout string, dir string) (headsha string, err error) {
	gitrepo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:           url,
		ReferenceName: plumbing.NewBranchReferenceName(checkout),
		SingleBranch:  true,
	})
	if err != nil {
		return
	}
	head, err := gitrepo.Head()
	if err != nil {
		return
	}
	headsha = head.Hash().String()
	return
}

// Downloads git repo, reads .testground/config.yml to create composition run opts.
func ParseConfigsFromGit(githubrepo string, gitbranch string) (cfg *TestgroundRunConfig, err error) {
	dir, err := ioutil.TempDir("/tmp", "tgbridge-gitclone-")
	if err != nil {
		return
	}
	giturl := strings.Join([]string{"https://github.com", githubrepo}, "/")
	head, err := GitClone(giturl, gitbranch, dir)
	if err != nil {
		return
	}

	cfgbytes, err := ioutil.ReadFile(filepath.Join(dir, ".testground/config.yml"))

	cfg = new(TestgroundRunConfig)
	err = yaml.Unmarshal(cfgbytes, cfg)
	if err != nil {
		return
	}

	for _, opts := range cfg.Compositions {
		opts.Branch = gitbranch
		opts.Repo = githubrepo
		opts.Commit = head
		opts.CompFile = filepath.Join(dir, opts.CompFile)
		opts.PlanDir = filepath.Join(dir, opts.PlanDir)
	}
	return
}
