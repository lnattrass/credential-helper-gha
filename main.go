package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/alecthomas/kong"
	"github.com/bradleyfalzon/ghinstallation"
	"github.com/google/go-github/github"
)

var (
	ErrGithubOrgNotFound = errors.New("github application was not found in the org")
)

type Cli struct {
	Get   Get  `cmd:"" help:"Get credentials"`
	Store NoOp `cmd:"" help:"No-Op" hidden:"" `
	Erase NoOp `cmd:"" help:"No-Op" hidden:""`
}

type NoOp struct{}

func (c *NoOp) Run() error {
	return nil
}

type Get struct {
	AppID     int64  `short:"a" required:"" help:"GitHub Application ID (required)" env:"GITHUB_APP_ID"`
	Key       string `short:"k" required:"" help:"GitHub Private Key file (required)" type:"path" env:"GITHUB_APP_KEY"`
	GithubOrg string `short:"o" required:"" help:"GitHub Organization Slug (required)"  env:"GITHUB_ORG"`
	TokenOnly bool   `short:"t" help:"Output Token Only"`

	client *github.Client
}

func (c *Get) Run() error {
	tr := http.DefaultTransport
	appsTransport, err := ghinstallation.NewAppsTransportKeyFromFile(tr, c.AppID, c.Key)
	if err != nil {
		return err
	}

	// Use apps transport to get an installation ID
	c.client = github.NewClient(&http.Client{Transport: appsTransport})

	installID, err := c.getInstallationID()
	if err != nil {
		return fmt.Errorf("%w: %s", err, c.GithubOrg)
	}

	installTransport := ghinstallation.NewFromAppsTransport(appsTransport, installID)
	token, err := installTransport.Token(context.Background())
	if err != nil {
		return err
	}

	if c.TokenOnly {
		fmt.Println(token)
		return nil
	}

	fmt.Printf("username=x-access-token\npassword=%s\n", token)
	return nil
}

func (c *Get) getInstallationID() (int64, error) {

	installations, _, err := c.client.Apps.ListInstallations(context.Background(), nil)
	if err != nil {
		return 0, err
	}

	for i := range installations {
		if installations[i].Account.GetLogin() == c.GithubOrg {
			return installations[i].GetID(), nil
		}
	}

	return 0, ErrGithubOrgNotFound
}

func main() {
	ctx := kong.Parse(&Cli{})
	ctx.FatalIfErrorf(ctx.Run())
}
