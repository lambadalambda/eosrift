package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"eosrift.com/eosrift/internal/auth"
	"eosrift.com/eosrift/internal/server"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "token", "tokens":
			tokenCmd(os.Args[2:])
			return
		case "reserve", "reservations":
			reserveCmd(os.Args[2:])
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
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if cfg.AuthToken != "" {
		if err := store.EnsureToken(ctx, cfg.AuthToken, "bootstrap"); err != nil {
			log.Fatalf("bootstrap token: %v", err)
		}
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewHandler(cfg, server.Dependencies{TokenValidator: store, TokenResolver: store, Reservations: store}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("eosrift-server listening on %s", addr)

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func tokenCmd(args []string) {
	if len(args) < 1 {
		tokenUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "create":
		tokenCreateCmd(args[1:])
	case "list":
		tokenListCmd(args[1:])
	case "revoke":
		tokenRevokeCmd(args[1:])
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

func tokenCreateCmd(args []string) {
	fs := flag.NewFlagSet("token create", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	label := fs.String("label", "", "Token label")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	rec, token, err := store.CreateToken(ctx, *label)
	if err != nil {
		log.Fatalf("create token: %v", err)
	}

	fmt.Printf("id: %d\n", rec.ID)
	if rec.Label != "" {
		fmt.Printf("label: %s\n", rec.Label)
	}
	fmt.Printf("token: %s\n", token)
}

func tokenListCmd(args []string) {
	fs := flag.NewFlagSet("token list", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	tokens, err := store.ListTokens(ctx)
	if err != nil {
		log.Fatalf("list tokens: %v", err)
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

func tokenRevokeCmd(args []string) {
	fs := flag.NewFlagSet("token revoke", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: eosrift-server token revoke [--db path] <id>")
		os.Exit(2)
	}

	id, err := strconv.ParseInt(fs.Arg(0), 10, 64)
	if err != nil || id <= 0 {
		log.Fatalf("invalid id: %q", fs.Arg(0))
	}

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.RevokeToken(ctx, id); err != nil {
		log.Fatalf("revoke token: %v", err)
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

func reserveCmd(args []string) {
	if len(args) < 1 {
		reserveUsage()
		os.Exit(2)
	}

	switch args[0] {
	case "add":
		reserveAddCmd(args[1:])
	case "list":
		reserveListCmd(args[1:])
	case "remove", "rm", "delete":
		reserveRemoveCmd(args[1:])
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

func reserveAddCmd(args []string) {
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
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.ReserveSubdomain(ctx, *tokenID, subdomain); err != nil {
		log.Fatalf("reserve subdomain: %v", err)
	}

	fmt.Printf("reserved %s\n", subdomain)
}

func reserveListCmd(args []string) {
	fs := flag.NewFlagSet("reserve list", flag.ExitOnError)
	dbPath := fs.String("db", getenv("EOSRIFT_DB_PATH", "/data/eosrift.db"), "SQLite DB path")
	_ = fs.Parse(args)

	ctx := context.Background()
	store, err := auth.Open(ctx, *dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	list, err := store.ListReservedSubdomains(ctx)
	if err != nil {
		log.Fatalf("list reserved subdomains: %v", err)
	}

	if len(list) == 0 {
		fmt.Println("no reserved subdomains")
		return
	}

	for _, r := range list {
		fmt.Printf("%s\t%d\t%s\n", r.Subdomain, r.TokenID, r.TokenPrefix)
	}
}

func reserveRemoveCmd(args []string) {
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
		log.Fatalf("open db: %v", err)
	}
	defer store.Close()

	if err := store.UnreserveSubdomain(ctx, subdomain); err != nil {
		log.Fatalf("unreserve subdomain: %v", err)
	}

	fmt.Printf("unreserved %s\n", subdomain)
}
