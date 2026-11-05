// grant-permissions lists — and with --apply creates — the Alert Permissions
// that staff need before deploying the release in which a caller without one
// is refused (alert-api #32). It reads every tenant database of the prefix.
//
//	MONGO_HOST=mongodb://… go run ./cmd/grant-permissions                 # dry run
//	MONGO_HOST=mongodb://… go run ./cmd/grant-permissions --clients 000   # one Client
//	MONGO_HOST=mongodb://… go run ./cmd/grant-permissions --apply         # write
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"alert/app/core/config"
	"alert/app/data/migration"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	writtenBy      = "migration:no-hq-fallback"
	connectTimeout = 30 * time.Second
	planTimeout    = 10 * time.Minute
	applyTimeout   = 10 * time.Minute
)

func main() {
	apply := flag.Bool("apply", false, "create the Permissions listed; without it the tool only prints the plan")
	prefix := flag.String("prefix", config.DbPrefixFromEnv(), "tenant database prefix (MONGO_DB_PREFIX)")
	defaultBranch := flag.String("default-branch", "HQ", "Branch for staff whose activity names none")
	clients := flag.String("clients", "", "comma-separated clientIds to limit the sweep to (default: every tenant of the prefix)")
	flag.Parse()

	uri := os.Getenv("MONGO_HOST")
	if uri == "" {
		fmt.Fprintln(os.Stderr, "MONGO_HOST is required")
		os.Exit(2)
	}
	client, err := connect(uri)
	if err != nil {
		fail(err)
	}
	defer disconnect(client)

	planner := migration.Planner{Client: client, Prefix: *prefix, DefaultBranch: *defaultBranch, Clients: splitClients(*clients)}

	planCtx, cancelPlan := context.WithTimeout(context.Background(), planTimeout)
	defer cancelPlan()
	tenants, skipped, err := planner.Tenants(planCtx)
	if err != nil {
		fail(err)
	}
	grants, err := planner.Plan(planCtx)
	if err != nil {
		fail(err)
	}
	cancelPlan()

	selected, excluded := planner.Selected(tenants)
	report(selected, excluded, skipped, grants, *prefix)
	if len(grants) == 0 {
		return
	}
	if !*apply {
		fmt.Println("dry run — re-run with --apply to create them")
		return
	}

	applyCtx, cancelApply := context.WithTimeout(context.Background(), applyTimeout)
	defer cancelApply()
	created, err := planner.Apply(applyCtx, grants, writtenBy)
	if err != nil {
		fmt.Fprintf(os.Stderr, "created %d before failing: %v\n", created, err)
		disconnect(client)
		os.Exit(1)
	}
	fmt.Printf("created %d Permission(s); %d already existed\n", created, len(grants)-created)
}

func report(selected, excluded []migration.Tenant, skipped []string, grants []migration.Grant, prefix string) {
	fmt.Printf("%d tenant database(s) under prefix %q: %s\n", len(selected), prefix, strings.Join(describe(selected), ", "))
	if len(excluded) > 0 {
		fmt.Printf("left out by --clients: %s\n", strings.Join(describe(excluded), ", "))
	}
	if len(skipped) > 0 {
		fmt.Printf("skipped %d database(s) whose name is not a tenant of this prefix: %s\n", len(skipped), strings.Join(skipped, ", "))
	}
	if len(grants) == 0 {
		fmt.Println("every staff member who has acted already holds an Alert Permission; nothing to do")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "DATABASE\tCLIENT\tUSER\tBRANCH\tLAST SEEN\tSOURCE")
	for _, g := range grants {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n", g.Database, g.ClientId, g.UserId, g.BranchId, g.LastSeen.Format(time.RFC3339), g.Source)
	}
	w.Flush()
	fmt.Printf("%d Permission(s) to create (Active, no Event Types of their own, UpdatedBy %s)\n", len(grants), writtenBy)
}

func describe(tenants []migration.Tenant) []string {
	names := make([]string, 0, len(tenants))
	for _, t := range tenants {
		names = append(names, fmt.Sprintf("%s (Client %s)", t.Database, t.ClientId))
	}
	return names
}

func connect(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return client, nil
}

func disconnect(client *mongo.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()
	_ = client.Disconnect(ctx)
}

func splitClients(list string) []string {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	var out []string
	for _, c := range strings.Split(list, ",") {
		if c = strings.TrimSpace(c); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
