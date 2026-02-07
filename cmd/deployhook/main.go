package main

import (
	"bytes"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"eosrift.com/eosrift/internal/deployhook"
)

func main() {
	logger := log.New(os.Stdout, "eosrift-deployhook: ", log.LstdFlags|log.LUTC)

	secret := strings.TrimSpace(os.Getenv("EOSRIFT_DEPLOY_WEBHOOK_SECRET"))
	if secret == "" {
		logger.Fatal("EOSRIFT_DEPLOY_WEBHOOK_SECRET is required")
	}

	listenAddr := getenv("EOSRIFT_DEPLOY_WEBHOOK_LISTEN_ADDR", ":8091")
	workflowName := getenv("EOSRIFT_DEPLOY_WEBHOOK_WORKFLOW", "Docker Image")
	branch := getenv("EOSRIFT_DEPLOY_WEBHOOK_BRANCH", "main")
	repository := strings.TrimSpace(os.Getenv("EOSRIFT_DEPLOY_WEBHOOK_REPOSITORY"))
	timeout := parseDuration(getenv("EOSRIFT_DEPLOY_TIMEOUT", "10m"), 10*time.Minute)

	command := getenv("EOSRIFT_DEPLOY_COMMAND", "/workspace/deploy/webhook/eosrift-deploy.sh")
	commandArgs := strings.Fields(strings.TrimSpace(os.Getenv("EOSRIFT_DEPLOY_COMMAND_ARGS")))

	trigger := func(ctx context.Context, run deployhook.Run) error {
		cmd := exec.CommandContext(ctx, command, commandArgs...)
		cmd.Env = os.Environ()

		var output bytes.Buffer
		cmd.Stdout = &output
		cmd.Stderr = &output

		err := cmd.Run()

		logText := strings.TrimSpace(output.String())
		if logText != "" {
			logger.Printf("deploy output run_id=%d:\n%s", run.RunID, logText)
		}

		if err != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return errors.New("deploy command timed out")
			}
			return err
		}
		return nil
	}

	handler, err := deployhook.NewHandler(deployhook.Config{
		Secret:       secret,
		WorkflowName: workflowName,
		Branch:       branch,
		Repository:   repository,
		Timeout:      timeout,
	}, trigger, logger)
	if err != nil {
		logger.Fatalf("init handler: %v", err)
	}

	srv := &http.Server{
		Addr:              listenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Printf("listening addr=%s workflow=%q branch=%q repo=%q", listenAddr, workflowName, branch, repository)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatalf("server error: %v", err)
	}
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
