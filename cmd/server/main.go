package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"eosrift.com/eosrift/internal/auth"
	"eosrift.com/eosrift/internal/logging"
	"eosrift.com/eosrift/internal/server"
)

func main() {
	logger := newLogger().With(logging.F("app", "eosrift-server"))

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token", "tokens":
			os.Exit(runTokenCmd(logger, os.Args[2:], os.Stdout, os.Stderr))
		case "reserve", "reservations":
			os.Exit(runReserveCmd(logger, os.Args[2:], os.Stdout, os.Stderr))
		case "tcp-reserve", "tcp-reservations":
			os.Exit(runTCPReserveCmd(logger, os.Args[2:], os.Stdout, os.Stderr))
		}
	}

	addr := os.Getenv("EOSRIFT_LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	cfg := server.ConfigFromEnv()
	if cfg.DBPath == "" {
		cfg.DBPath = getenv("EOSRIFT_DB_PATH", "/data/eosrift.db")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store, err := auth.Open(ctx, cfg.DBPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if cfg.AuthToken != "" {
		if err := store.EnsureToken(ctx, cfg.AuthToken, "bootstrap"); err != nil {
			fatal(logger, "bootstrap token", logging.F("err", err))
		}
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewHandler(cfg, server.Dependencies{TokenValidator: store, TokenResolver: store, Reservations: store, AdminStore: store, Logger: logger}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	logger.Info("listening", logging.F("addr", addr))

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fatal(logger, "server error", logging.F("err", err))
	}
}

func runTokenCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		tokenUsage(stderr)
		return 2
	}

	switch args[0] {
	case "create":
		return runTokenCreateCmd(logger, args[1:], stdout, stderr)
	case "list":
		return runTokenListCmd(logger, args[1:], stdout, stderr)
	case "revoke":
		return runTokenRevokeCmd(logger, args[1:], stdout, stderr)
	default:
		tokenUsage(stderr)
		return 2
	}
}

func tokenUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: eosrift-server token <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  create   create a new authtoken")
	fmt.Fprintln(w, "  list     list authtokens")
	fmt.Fprintln(w, "  revoke   revoke an authtoken by id")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "env:")
	fmt.Fprintln(w, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func runTokenCreateCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("token create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	label := fs.String("label", "", "Token label")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	rec, token, err := store.CreateToken(ctx, *label)
	if err != nil {
		return adminError(logger, stderr, "create token", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "id: %d\n", rec.ID)
	if rec.Label != "" {
		fmt.Fprintf(stdout, "label: %s\n", rec.Label)
	}
	fmt.Fprintf(stdout, "token: %s\n", token)
	return 0
}

func runTokenListCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("token list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	tokens, err := store.ListTokens(ctx)
	if err != nil {
		return adminError(logger, stderr, "list tokens", logging.F("err", err))
	}

	if len(tokens) == 0 {
		fmt.Fprintln(stdout, "no tokens")
		return 0
	}

	for _, t := range tokens {
		status := "active"
		if t.RevokedAt != nil {
			status = "revoked"
		}
		label := t.Label
		if label == "" {
			label = "-"
		}
		fmt.Fprintf(stdout, "%d\t%s\t%s\t%s\n", t.ID, t.Prefix, label, status)
	}
	return 0
}

func runTokenRevokeCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("token revoke", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift-server token revoke [--db path] <id>")
		return 2
	}

	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil || id <= 0 {
		return adminError(logger, stderr, "invalid id", logging.F("id", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.RevokeToken(ctx, id); err != nil {
		return adminError(logger, stderr, "revoke token", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "revoked %d\n", id)
	return 0
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func runReserveCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		reserveUsage(stderr)
		return 2
	}

	switch args[0] {
	case "add":
		return runReserveAddCmd(logger, args[1:], stdout, stderr)
	case "list":
		return runReserveListCmd(logger, args[1:], stdout, stderr)
	case "remove", "rm", "delete":
		return runReserveRemoveCmd(logger, args[1:], stdout, stderr)
	default:
		reserveUsage(stderr)
		return 2
	}
}

func reserveUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: eosrift-server reserve <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  add      reserve a subdomain for a token id")
	fmt.Fprintln(w, "  list     list reserved subdomains")
	fmt.Fprintln(w, "  remove   remove a reserved subdomain")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "env:")
	fmt.Fprintln(w, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func runReserveAddCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reserve add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	tokenID := fs.Int64("token-id", 0, "Token id to bind the subdomain to")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *tokenID <= 0 || fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift-server reserve add --token-id <id> [--db path] <subdomain>")
		return 2
	}

	subdomain := fs.Arg(0)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.ReserveSubdomain(ctx, *tokenID, subdomain); err != nil {
		return adminError(logger, stderr, "reserve subdomain", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "reserved %s\n", subdomain)
	return 0
}

func runReserveListCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reserve list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	list, err := store.ListReservedSubdomains(ctx)
	if err != nil {
		return adminError(logger, stderr, "list reserved subdomains", logging.F("err", err))
	}

	if len(list) == 0 {
		fmt.Fprintln(stdout, "no reserved subdomains")
		return 0
	}

	for _, r := range list {
		fmt.Fprintf(stdout, "%s\t%d\t%s\n", r.Subdomain, r.TokenID, r.TokenPrefix)
	}
	return 0
}

func runReserveRemoveCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reserve remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift-server reserve remove [--db path] <subdomain>")
		return 2
	}

	subdomain := fs.Arg(0)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.UnreserveSubdomain(ctx, subdomain); err != nil {
		return adminError(logger, stderr, "unreserve subdomain", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "unreserved %s\n", subdomain)
	return 0
}

func runTCPReserveCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		tcpReserveUsage(stderr)
		return 2
	}

	switch args[0] {
	case "add":
		return runTCPReserveAddCmd(logger, args[1:], stdout, stderr)
	case "list":
		return runTCPReserveListCmd(logger, args[1:], stdout, stderr)
	case "remove", "rm", "delete":
		return runTCPReserveRemoveCmd(logger, args[1:], stdout, stderr)
	default:
		tcpReserveUsage(stderr)
		return 2
	}
}

func tcpReserveUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: eosrift-server tcp-reserve <command> [args]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  add      reserve a TCP port for a token id")
	fmt.Fprintln(w, "  list     list reserved TCP ports")
	fmt.Fprintln(w, "  remove   remove a reserved TCP port")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "env:")
	fmt.Fprintln(w, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func runTCPReserveAddCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tcp-reserve add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	tokenID := fs.Int64("token-id", 0, "Token id to bind the port to")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *tokenID <= 0 || fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift-server tcp-reserve add --token-id <id> [--db path] <port>")
		return 2
	}

	port, err := strconv.Atoi(fs.Arg(0))
	if err != nil || port <= 0 {
		return adminError(logger, stderr, "invalid port", logging.F("port", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.ReserveTCPPort(ctx, *tokenID, port); err != nil {
		return adminError(logger, stderr, "reserve tcp port", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "reserved %d\n", port)
	return 0
}

func runTCPReserveListCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tcp-reserve list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	list, err := store.ListReservedTCPPorts(ctx)
	if err != nil {
		return adminError(logger, stderr, "list reserved tcp ports", logging.F("err", err))
	}

	if len(list) == 0 {
		fmt.Fprintln(stdout, "no reserved tcp ports")
		return 0
	}

	for _, r := range list {
		fmt.Fprintf(stdout, "%d\t%d\t%s\n", r.Port, r.TokenID, r.TokenPrefix)
	}
	return 0
}

func runTCPReserveRemoveCmd(logger logging.Logger, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("tcp-reserve remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: eosrift-server tcp-reserve remove [--db path] <port>")
		return 2
	}

	port, err := strconv.Atoi(fs.Arg(0))
	if err != nil || port <= 0 {
		return adminError(logger, stderr, "invalid port", logging.F("port", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		return adminError(logger, stderr, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.UnreserveTCPPort(ctx, port); err != nil {
		return adminError(logger, stderr, "unreserve tcp port", logging.F("err", err))
	}

	fmt.Fprintf(stdout, "unreserved %d\n", port)
	return 0
}

func newLogger() logging.Logger {
	level, _ := logging.ParseLevel(os.Getenv("EOSRIFT_LOG_LEVEL"))
	format, _ := logging.ParseFormat(os.Getenv("EOSRIFT_LOG_FORMAT"))

	return logging.New(logging.Options{
		Out:    os.Stderr,
		Level:  level,
		Format: format,
	})
}

func fatal(logger logging.Logger, msg string, fields ...logging.Field) {
	if logger != nil {
		logger.Error(msg, fields...)
	} else {
		fmt.Fprintln(os.Stderr, msg)
	}
	os.Exit(1)
}

func adminError(logger logging.Logger, stderr io.Writer, msg string, fields ...logging.Field) int {
	if logger != nil {
		logger.Error(msg, fields...)
	} else {
		fmt.Fprintln(stderr, msg)
	}
	return 1
}
