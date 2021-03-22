package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/go-github/v33/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/pkg/errors"
	"github.com/testground/testground/pkg/api"
	"github.com/testground/testground/pkg/client"
)

var (
	queued      = "queued"
	in_progress = "in_progress"
	completed   = "completed"
	success     = "success"
	cancelled   = "cancelled"
)

type StartedTask struct {
	Name            string
	Id              string
	CheckSuiteEvent *github.CheckSuiteEvent
	CheckRun        *github.CheckRun
}

type CheckHandler struct {
	githubapp.ClientCreator
}

func (h *CheckHandler) Handles() []string {
	return []string{"check_suite"}
}

func (h *CheckHandler) Handle(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	var event github.CheckSuiteEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return errors.Wrap(err, "failed to parse issue comment event payload")
	}

	installationID := githubapp.GetInstallationIDFromEvent(&event)
	ghclient, err := h.NewInstallationClient(installationID)
	if err != nil {
		return err
	}

	switch event.GetAction() {
	case "requested":
		fmt.Println("requested")
		cs := event.GetCheckSuite()
		repo := event.GetRepo()
		owner := event.GetRepo().GetOwner()

		fmt.Println("starting tasks", repo.GetFullName(), cs.GetHeadBranch())
		tgclient, tsks, err := StartTasks(repo.GetFullName(), cs.GetHeadBranch())
		if err != nil {
			return err
		}
		for _, tsk := range tsks {
			cropts := github.CreateCheckRunOptions{
				Name:    tsk.Name,
				HeadSHA: cs.GetHeadSHA(),
				// DetailsURL:
				ExternalID: &tsk.Id,
				// Status: "queued",
				Status: &queued,
				// Conclusion:
				// StartedAt:
				// CompletedAt:
				// Output:
				// Actions: []*CheckRunAction

			}
			cr, resp, err := ghclient.Checks.CreateCheckRun(ctx, owner.GetLogin(), repo.GetName(), cropts)
			if err != nil {
				return err
			}

			fmt.Println("create cr response", resp.Status)
			tsk.CheckSuiteEvent = &event
			tsk.CheckRun = cr
			go WatchTask(tgclient, ghclient, tsk)
		}

	case "rerequested":
		fmt.Println("rerequested")
	case "completed":
		fmt.Println("completed")
	}

	return nil
}

func StartTasks(giturl string, checkout string) (tgclient *client.Client, started []*StartedTask, err error) {
	cfg, err := ParseConfigsFromGit(giturl, checkout)
	if err != nil {
		return
	}
	tgclient, err = NewTestgroundClient(cfg.Backend)
	if err != nil {
		return
	}
	started = make([]*StartedTask, 0)
	for _, opts := range cfg.Compositions {
		id, err := TestgroundRun(tgclient, opts)
		if err != nil {
			panic(err)
		}
		fmt.Println(id)
		fmt.Println("created task in testground", id)
		started = append(started, &StartedTask{Id: id, Name: opts.Name})
	}
	return
}

// Long poll testground, provide updates to github.
// Stop when testground reaches a terminal state.
func WatchTask(tgclient *client.Client, ghclient *github.Client, tsk *StartedTask) {
	ctx := context.Background()
	previous_tgstate := "scheduled"
	for range time.Tick(10 * time.Second) {
		tgresp, err := tgclient.Status(ctx, &api.StatusRequest{TaskID: tsk.Id})
		if err != nil {
			// TODO report error to github.
			fmt.Println("error 1", tsk.Id)
			return
		}
		tgstatus, err := client.ParseStatusResponse(tgresp)
		if err != nil {
			fmt.Println("error 2", tsk.Id)
			return
		}
		tgstate := tgstatus.State().State
		fmt.Println("state update", tsk.Id, tgstate)
		switch tgstate {
		case "processing":
			if previous_tgstate != "processing" {
				previous_tgstate = "processing"
				UpdateGithub(ctx, ghclient, tsk)
			}
		case "complete":
			UpdateGithubComplete(ctx, tgclient, ghclient, tsk)
		case "canceled":
			UpdateGithubCancel(ctx, ghclient, tsk)
		default:
		}
	}
}

func UpdateGithub(ctx context.Context, ghclient *github.Client, tsk *StartedTask) {
	update := github.UpdateCheckRunOptions{
		Name: tsk.Name,
		// DetailsURL
		ExternalID: &tsk.Id,
		Status:     &in_progress,
		// Conclusion
		// CompletedAt
		// Output
		// Actions
	}

	repo := tsk.CheckSuiteEvent.GetRepo()
	owner := repo.GetOwner()
	_, resp, err := ghclient.Checks.UpdateCheckRun(ctx, owner.GetLogin(), repo.GetName(), tsk.CheckRun.GetID(), update)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("create cr update", resp.Status)
}

func UpdateGithubComplete(ctx context.Context, tgclient *client.Client, ghclient *github.Client, tsk *StartedTask) {
	// TODO: This is supposed to look at the "outcome" of the run. Right now, it always shows success
	// to github.
	update := github.UpdateCheckRunOptions{
		Name: tsk.Name,
		// DetailsURL
		ExternalID: &tsk.Id,
		Status:     &completed,
		Conclusion: &success,
		// CompletedAt
		// Output
		// Actions
	}

	repo := tsk.CheckSuiteEvent.GetRepo()
	owner := repo.GetOwner()
	_, resp, err := ghclient.Checks.UpdateCheckRun(ctx, owner.GetLogin(), repo.GetName(), tsk.CheckRun.GetID(), update)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("create cr update complete", resp.Status)
}

func UpdateGithubCancel(ctx context.Context, ghclient *github.Client, tsk *StartedTask) {
	update := github.UpdateCheckRunOptions{
		Name: tsk.Name,
		// DetailsURL
		ExternalID: &tsk.Id,
		Status:     &completed,
		Conclusion: &cancelled,
		// CompletedAt
		// Output
		// Actions
	}

	repo := tsk.CheckSuiteEvent.GetRepo()
	owner := repo.GetOwner()
	_, resp, err := ghclient.Checks.UpdateCheckRun(ctx, owner.GetLogin(), repo.GetName(), tsk.CheckRun.GetID(), update)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("create cr update complete", resp.Status)
}
