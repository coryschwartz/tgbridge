package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/testground/testground/pkg/api"
	"github.com/testground/testground/pkg/client"
	"github.com/testground/testground/pkg/config"
)

type TestgroundRunConfig struct {
	Backend      string               `yaml:"backend"`
	Compositions []*TestgroundRunOpts `yaml:"compositions"`
}

func (l *TestgroundRunConfig) String() string {
	return fmt.Sprintf("TestgroundRunList[%d]", len(l.Compositions))
}

type TestgroundRunOpts struct {
	Name     string `yaml:"Name"`
	PlanDir  string `yaml:"PlanDir"`
	CompFile string `yaml:"CompFile"`
	User     string
	Repo     string
	Branch   string
	Commit   string
}

func (o *TestgroundRunOpts) String() string {
	return strings.Join([]string{"TestgroundRunOpts", o.PlanDir, o.CompFile, o.User, o.Repo, o.Branch, o.Commit}, ":")
}

// Sets up client from .env.toml
func NewTestgroundClient(endpoint string) (tgclient *client.Client, err error) {
	tgcfg := &config.EnvConfig{}
	if err = tgcfg.Load(); err != nil {
		return
	}
	tgcfg.Client.Endpoint = endpoint
	tgclient = client.New(tgcfg)
	return
}

// decodes a compositoin toml file
func compositionFromFile(filename string) (comp *api.Composition, err error) {
	comp = new(api.Composition)
	_, err = toml.DecodeFile(filename, comp)
	if err != nil {
		return
	}
	err = comp.ValidateForRun()
	return
}

func testPlanManifestFromPath(path string) (tpm *api.TestPlanManifest, err error) {
	tpm = new(api.TestPlanManifest)
	manifest := filepath.Join(path, "manifest.toml")
	switch fi, err := os.Stat(manifest); {
	case err != nil:
		return nil, fmt.Errorf("failed to access plan manifest at %s: %w", manifest, err)
	case fi.IsDir():
		return nil, fmt.Errorf("failed to access plan manifest at %s: not a file", manifest)
	}
	_, err = toml.DecodeFile(manifest, tpm)
	return
}

func TestgroundRun(tgclient *client.Client, opts *TestgroundRunOpts) (string, error) {
	planDir := opts.PlanDir
	compFile := opts.CompFile

	manifest, err := testPlanManifestFromPath(planDir)
	if err != nil {
		panic(err)
	}

	comp, err := compositionFromFile(compFile)

	var buildIdx []int
	for i, grp := range comp.Groups {
		if grp.Run.Artifact == "" {
			buildIdx = append(buildIdx, i)
		}
	}

	var extraSrcs []string
	if len(buildIdx) > 0 {
		// if there are extra sources to include for this builder, contextualize
		// them to the plan's dir.
		builder := comp.Global.Builder
		extraSrcs = manifest.ExtraSources[builder]
		for i, dir := range extraSrcs {
			if !filepath.IsAbs(dir) {
				// follow any symlinks in the plan dir.
				evalPlanDir, err := filepath.EvalSymlinks(planDir)
				if err != nil {
					return "", fmt.Errorf("failed to follow symlinks in plan dir: %w", err)
				}
				extraSrcs[i] = filepath.Clean(filepath.Join(evalPlanDir, dir))
			}
		}
	} else {
		planDir = ""
	}
	req := &api.RunRequest{
		BuildGroups: buildIdx,
		Composition: *comp,
		Manifest:    *manifest,
		CreatedBy: api.CreatedBy{
			User:   opts.User,
			Repo:   opts.Repo,
			Branch: opts.Branch,
			Commit: opts.Commit,
		},
		//Priority: 1,
	}

	resp, err := tgclient.Run(context.Background(), req, planDir, "", extraSrcs)
	if err != nil {
		return "", err
	}
	return client.ParseRunResponse(resp)
}
