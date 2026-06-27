// Command tools.admin is the GitHub admin automation CLI (snapshot, create-pr,
// cleanup-runs). This file is the composition root: it builds the production Env
// of real seams (HTTP, subprocess, filesystem, clock, env) and hands it to the
// cli package, which holds all the testable logic. It deliberately contains no
// branching worth covering — every decision lives behind an injected seam.
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	tcli "github.com/nicerobot/tools.admin/internal/cli"
	"github.com/nicerobot/tools.admin/internal/githubapi"
	"github.com/nicerobot/tools.admin/internal/gitsvc"
)

const httpTimeout = 30 * time.Second

func main() {
	env := tcli.Env{
		Out:      os.Stdout,
		Doer:     newHTTPClient(),
		BaseURL:  githubapi.DefaultBaseURL,
		Getenv:   os.Getenv,
		Now:      time.Now,
		GitRun:   runCommand,
		ReadFile: os.ReadFile,
		Mkdir:    func(path string) error { return os.MkdirAll(path, 0o755) },
		WriteOut: func(name string, data []byte) error { return os.WriteFile(name, data, 0o644) },
		Glob:     filepath.Glob,
		Remove:   os.Remove,
	}
	if err := tcli.NewCommand(env).Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// newHTTPClient returns a client that does NOT follow redirects, so RepoExists
// can observe a 301 (renamed/transferred repo) instead of chasing it.
func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: httpTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// runCommand executes git/gh, capturing stdout. A clean exit and a non-zero
// exit both return a Result (the latter with the exit code); only a failure to
// execute at all returns an error.
func runCommand(args []string) (gitsvc.Result, error) {
	cmd := exec.Command(args[0], args[1:]...)
	var out strings.Builder
	cmd.Stdout = &out
	err := cmd.Run()
	if err == nil {
		return gitsvc.Result{Stdout: out.String()}, nil
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return gitsvc.Result{Stdout: out.String(), ExitCode: exit.ExitCode()}, nil
	}
	return gitsvc.Result{}, err
}
