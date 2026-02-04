package main

import (
	"context"
	"flag"
	"fmt"
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
			tokenCmd(logger, os.Args[2:])
			return
		case "reserve", "reservations":
			reserveCmd(logger, os.Args[2:])
			return
		case "tcp-reserve", "tcp-reservations":
			tcpReserveCmd(logger, os.Args[2:])
			return
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
		Handler:           server.NewHandler(cfg, server.Dependencies{TokenValidator: store, TokenResolver: store, Reservations: store, Logger: logger}),
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

func tokenCmd(logger logging.Logger, args []string) {
	if len(args) < 1 {
		tokenUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "create":
		tokenCreateCmd(logger, args[1:])
	case "list":
		tokenListCmd(logger, args[1:])
	case "revoke":
		tokenRevokeCmd(logger, args[1:])
	default:
		tokenUsage()
		os.Exit(2)
	}
}

func tokenUsage() {
	fmt.Fprintln(os.Stderr, "usage: eosrift-server token <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  create   create a new authtoken")
	fmt.Fprintln(os.Stderr, "  list     list authtokens")
	fmt.Fprintln(os.Stderr, "  revoke   revoke an authtoken by id")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "env:")
	fmt.Fprintln(os.Stderr, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func tokenCreateCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("token create", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	label := fs.String("label", "", "Token label")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	rec, token, err := store.CreateToken(ctx, *label)
	if err != nil {
		fatal(logger, "create token", logging.F("err", err))
	}

	fmt.Printf("id: %d\n", rec.ID)
	if rec.Label != "" {
		fmt.Printf("label: %s\n", rec.Label)
	}
	fmt.Printf("token: %s\n", token)
}

func tokenListCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("token list", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	tokens, err := store.ListTokens(ctx)
	if err != nil {
		fatal(logger, "list tokens", logging.F("err", err))
	}

	if len(tokens) == 0 {
		fmt.Println("no tokens")
		return
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
		fmt.Printf("%d\t%s\t%s\t%s\n", t.ID, t.Prefix, label, status)
	}
}

func tokenRevokeCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("token revoke", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server token revoke [--db path] <id>")
		os.Exit(2)
	}

	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil || id <= 0 {
		fatal(logger, "invalid id", logging.F("id", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.RevokeToken(ctx, id); err != nil {
		fatal(logger, "revoke token", logging.F("err", err))
	}

	fmt.Printf("revoked %d\n", id)
}

func getenv(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func reserveCmd(logger logging.Logger, args []string) {
	if len(args) < 1 {
		reserveUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "add":
		reserveAddCmd(logger, args[1:])
	case "list":
		reserveListCmd(logger, args[1:])
	case "remove", "rm", "delete":
		reserveRemoveCmd(logger, args[1:])
	default:
		reserveUsage()
		os.Exit(2)
	}
}

func reserveUsage() {
	fmt.Fprintln(os.Stderr, "usage: eosrift-server reserve <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  add      reserve a subdomain for a token id")
	fmt.Fprintln(os.Stderr, "  list     list reserved subdomains")
	fmt.Fprintln(os.Stderr, "  remove   remove a reserved subdomain")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "env:")
	fmt.Fprintln(os.Stderr, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func reserveAddCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("reserve add", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	tokenID := fs.Int64("token-id", 0, "Token id to bind the subdomain to")
	_ = fs.Parse(args)

	if *tokenID <= 0 || fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server reserve add --token-id <id> [--db path] <subdomain>")
		os.Exit(2)
	}

	subdomain := fs.Arg(0)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.ReserveSubdomain(ctx, *tokenID, subdomain); err != nil {
		fatal(logger, "reserve subdomain", logging.F("err", err))
	}

	fmt.Printf("reserved %s\n", subdomain)
}

func reserveListCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("reserve list", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	list, err := store.ListReservedSubdomains(ctx)
	if err != nil {
		fatal(logger, "list reserved subdomains", logging.F("err", err))
	}

	if len(list) == 0 {
		fmt.Println("no reserved subdomains")
		return
	}

	for _, r := range list {
		fmt.Printf("%s\t%d\t%s\n", r.Subdomain, r.TokenID, r.TokenPrefix)
	}
}

func reserveRemoveCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("reserve remove", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server reserve remove [--db path] <subdomain>")
		os.Exit(2)
	}

	subdomain := fs.Arg(0)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.UnreserveSubdomain(ctx, subdomain); err != nil {
		fatal(logger, "unreserve subdomain", logging.F("err", err))
	}

	fmt.Printf("unreserved %s\n", subdomain)
}

func tcpReserveCmd(logger logging.Logger, args []string) {
	if len(args) < 1 {
		tcpReserveUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "add":
		tcpReserveAddCmd(logger, args[1:])
	case "list":
		tcpReserveListCmd(logger, args[1:])
	case "remove", "rm", "delete":
		tcpReserveRemoveCmd(logger, args[1:])
	default:
		tcpReserveUsage()
		os.Exit(2)
	}
}

func tcpReserveUsage() {
	fmt.Fprintln(os.Stderr, "usage: eosrift-server tcp-reserve <command> [args]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "commands:")
	fmt.Fprintln(os.Stderr, "  add      reserve a TCP port for a token id")
	fmt.Fprintln(os.Stderr, "  list     list reserved TCP ports")
	fmt.Fprintln(os.Stderr, "  remove   remove a reserved TCP port")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "env:")
	fmt.Fprintln(os.Stderr, "  EOSRIFT_DB_PATH  sqlite db path (default: /data/eosrift.db)")
}

func tcpReserveAddCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("tcp-reserve add", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	tokenID := fs.Int64("token-id", 0, "Token id to bind the port to")
	_ = fs.Parse(args)

	if *tokenID <= 0 || fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server tcp-reserve add --token-id <id> [--db path] <port>")
		os.Exit(2)
	}

	port, err := strconv.Atoi(fs.Arg(0))
	if err != nil || port <= 0 {
		fatal(logger, "invalid port", logging.F("port", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.ReserveTCPPort(ctx, *tokenID, port); err != nil {
		fatal(logger, "reserve tcp port", logging.F("err", err))
	}

	fmt.Printf("reserved %d\n", port)
}

func tcpReserveListCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("tcp-reserve list", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	list, err := store.ListReservedTCPPorts(ctx)
	if err != nil {
		fatal(logger, "list reserved tcp ports", logging.F("err", err))
	}

	if len(list) == 0 {
		fmt.Println("no reserved tcp ports")
		return
	}

	for _, r := range list {
		fmt.Printf("%d\t%d\t%s\n", r.Port, r.TokenID, r.TokenPrefix)
	}
}

func tcpReserveRemoveCmd(logger logging.Logger, args []string) {
	fs := flag.NewFlagSet("tcp-reserve remove", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server tcp-reserve remove [--db path] <port>")
		os.Exit(2)
	}

	port, err := strconv.Atoi(fs.Arg(0))
	if err != nil || port <= 0 {
		fatal(logger, "invalid port", logging.F("port", fs.Arg(0)))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		fatal(logger, "open db", logging.F("err", err))
	}
	defer store.Close()

	if err := store.UnreserveTCPPort(ctx, port); err != nil {
		fatal(logger, "unreserve tcp port", logging.F("err", err))
	}

	fmt.Printf("unreserved %d\n", port)
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
