package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	gotype2cli "github.com/pdcalado/gotype2cli/pkg"
)

//go:generate go run github.com/pdcalado/gotype2cli/cmd -type=Repo -w -receiver-print=false
type Repo struct {
	client *http.Client
}

// New creates a new Repo
func New(client *http.Client) Repo {
	return Repo{
		client: client,
	}
}

// CheckHealth checks the health of a http server
func (r Repo) CheckHealth(ctx context.Context, host string, port int) error {
	_, err := r.Fetch(ctx, host, port)
	return err
}

// Fetch fetches something from a http server
func (r Repo) Fetch(ctx context.Context, host string, port int) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://%s:%d", host, port), nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func main() {
	repoCmd, err := makeRepoCommand(
		func(opts *gotype2cli.CreateCommandOptions) {
			opts.DefaultConstructor = func() any {
				r := New(&http.Client{})
				return &r
			}
		},
	)
	if err != nil {
		panic(err)
	}

	err = repoCmd.Execute()
	if err != nil {
		panic(err)
	}
}
