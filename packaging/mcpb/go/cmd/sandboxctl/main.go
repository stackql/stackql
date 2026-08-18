// Command sandboxctl is the demo app for stackql-mcp-go: an on-demand
// infrastructure concierge in a single compiled binary. It embeds the
// stackql MCP server (no runtime dependencies) and a Claude agent loop.
//
// Developers ask for ephemeral cloud sandboxes in plain language; the
// agent plans, prices, provisions, and labels everything with an expiry so
// `sandboxctl reap` can tear it down later. Nothing is created before the
// plan and cost estimate pass an explicit approval gate, and every SQL
// statement the agent runs is printed.
//
//	sandboxctl request "a small linux vm and a bucket in sydney for 2 days" --project my-gcp-project
//	sandboxctl reap --project my-gcp-project
//	sandboxctl tools
//
// Requires ANTHROPIC_API_KEY (request/reap) and GOOGLE_CREDENTIALS (a GCP
// service account key JSON, stackql's default google auth) in the
// environment. Build with `go generate ./cmd/sandboxctl` first, which
// fetches and pin-verifies the embedded server binary.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	stackqlmcp "github.com/stackql/stackql-mcp-go/embed"
)

//go:generate go run github.com/stackql/stackql-mcp-go/cmd/stackql-mcp-fetch -platform auto -package main

const defaultModel = "claude-sonnet-4-6"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "sandboxctl:", err)
		os.Exit(1)
	}
}

func run(argv []string) error {
	if len(argv) == 0 {
		usage()
		return fmt.Errorf("a subcommand is required")
	}
	switch argv[0] {
	case "request":
		return cmdRequest(argv[1:])
	case "reap":
		return cmdReap(argv[1:])
	case "tools":
		return cmdTools(argv[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		usage()
		return fmt.Errorf("unknown subcommand %q", argv[0])
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  sandboxctl request "<what you need>" --project <gcp-project> [--expires 48h] [--approve] [--model <id>]
  sandboxctl reap --project <gcp-project> [--approve] [--model <id>]
  sandboxctl tools
`)
}

func model(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("ANTHROPIC_MODEL"); env != "" {
		return env
	}
	return defaultModel
}

// startServer spawns the embedded stackql MCP server in the given mode.
func startServer(ctx context.Context, mode stackqlmcp.Mode) (*stackqlmcp.Client, error) {
	fmt.Fprintf(os.Stderr, "starting embedded stackql mcp server (mode=%s)\n", mode)
	return stackqlmcp.StartServer(ctx, stackqlmcp.Options{
		Binary: StackqlMCPBinary(),
		Mode:   mode,
	})
}

func checkEnv(needAnthropic bool) error {
	if needAnthropic && os.Getenv("ANTHROPIC_API_KEY") == "" {
		return fmt.Errorf("ANTHROPIC_API_KEY is not set")
	}
	if os.Getenv("GOOGLE_CREDENTIALS") == "" {
		fmt.Fprintln(os.Stderr, "warning: GOOGLE_CREDENTIALS is not set; google provider calls will fail to authenticate")
	}
	return nil
}

// confirm is the approval boundary: nothing mutating runs before a yes.
func confirm(approved bool, prompt string) (bool, error) {
	if approved {
		fmt.Fprintln(os.Stderr, "pre-approved with --approve")
		return true, nil
	}
	fmt.Printf("\n%s [y/N]: ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("reading approval (use --approve for non-interactive runs): %w", err)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func cmdRequest(argv []string) error {
	fs := flag.NewFlagSet("request", flag.ContinueOnError)
	project := fs.String("project", "", "GCP project id (required)")
	expires := fs.Duration("expires", 48*time.Hour, "sandbox lifetime")
	approve := fs.Bool("approve", false, "skip the interactive approval gate")
	modelFlag := fs.String("model", "", "Anthropic model id")
	// Accept the description before or after the flags: the flag package
	// stops at the first positional, so re-parse the remainder.
	if err := fs.Parse(argv); err != nil {
		return err
	}
	rest := fs.Args()
	var request string
	if len(rest) > 0 {
		request = rest[0]
		if err := fs.Parse(rest[1:]); err != nil {
			return err
		}
		rest = fs.Args()
	}
	if request == "" || len(rest) != 0 {
		return fmt.Errorf(`request takes exactly one quoted description, for example: sandboxctl request "a small linux vm in sydney" --project my-project`)
	}
	if *project == "" {
		return fmt.Errorf("--project is required")
	}
	if err := checkEnv(true); err != nil {
		return err
	}
	expiresAt := time.Now().Add(*expires).UTC()
	ctx := context.Background()

	// Phase 1: plan and price with a read_only server. The agent cannot
	// create anything here even if it tries.
	planServer, err := startServer(ctx, stackqlmcp.ModeReadOnly)
	if err != nil {
		return err
	}
	agent, err := newAgentSession(ctx, planServer, model(*modelFlag))
	if err != nil {
		planServer.Close()
		return err
	}
	fmt.Println("== phase 1: plan and cost estimate (read_only) ==")
	plan, err := agent.run(ctx, planSystemPrompt, planUserPrompt(request, *project, expiresAt), 40)
	planServer.Close()
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", plan)

	ok, err := confirm(*approve, "create these resources?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("not approved; nothing was created")
		return nil
	}

	// Phase 2: provision with a fresh server in safe mode.
	provServer, err := startServer(ctx, stackqlmcp.ModeSafe)
	if err != nil {
		return err
	}
	defer provServer.Close()
	agent, err = newAgentSession(ctx, provServer, model(*modelFlag))
	if err != nil {
		return err
	}
	fmt.Println("\n== phase 2: provision (safe) ==")
	report, err := agent.run(ctx, provisionSystemPrompt, provisionUserPrompt(plan, *project, expiresAt), 40)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", report)
	return nil
}

func cmdReap(argv []string) error {
	fs := flag.NewFlagSet("reap", flag.ContinueOnError)
	project := fs.String("project", "", "GCP project id (required)")
	approve := fs.Bool("approve", false, "skip the interactive approval gate (for cron)")
	modelFlag := fs.String("model", "", "Anthropic model id")
	if err := fs.Parse(argv); err != nil {
		return err
	}
	if *project == "" {
		return fmt.Errorf("--project is required")
	}
	if err := checkEnv(true); err != nil {
		return err
	}
	ctx := context.Background()
	now := time.Now().UTC()

	// Phase 1: find expired sandboxes with a read_only server.
	findServer, err := startServer(ctx, stackqlmcp.ModeReadOnly)
	if err != nil {
		return err
	}
	agent, err := newAgentSession(ctx, findServer, model(*modelFlag))
	if err != nil {
		findServer.Close()
		return err
	}
	fmt.Println("== phase 1: find expired sandboxes (read_only) ==")
	findings, err := agent.run(ctx, reapFindSystemPrompt, reapFindUserPrompt(*project, now), 40)
	findServer.Close()
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", findings)
	if strings.Contains(findings, "NOTHING-TO-REAP") {
		fmt.Println("no expired sandboxes")
		return nil
	}

	ok, err := confirm(*approve, "tear down the expired resources listed above?")
	if err != nil {
		return err
	}
	if !ok {
		fmt.Println("not approved; nothing was deleted")
		return nil
	}

	// Phase 2: tear down with delete_safe.
	reapServer, err := startServer(ctx, stackqlmcp.ModeDeleteSafe)
	if err != nil {
		return err
	}
	defer reapServer.Close()
	agent, err = newAgentSession(ctx, reapServer, model(*modelFlag))
	if err != nil {
		return err
	}
	fmt.Println("\n== phase 2: tear down (delete_safe) ==")
	report, err := agent.run(ctx, reapTeardownSystemPrompt, reapTeardownUserPrompt(findings, *project), 40)
	if err != nil {
		return err
	}
	fmt.Printf("\n%s\n", report)
	return nil
}

// cmdTools lists the embedded server's MCP tools. It needs no credentials
// and doubles as a smoke check that the embedded binary starts.
func cmdTools(argv []string) error {
	fs := flag.NewFlagSet("tools", flag.ContinueOnError)
	if err := fs.Parse(argv); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	server, err := startServer(ctx, stackqlmcp.ModeReadOnly)
	if err != nil {
		return err
	}
	defer server.Close()
	listed, err := server.Session.ListTools(ctx, nil)
	if err != nil {
		return err
	}
	info := server.Session.InitializeResult()
	fmt.Printf("server: %s %s (binary: %s)\n", info.ServerInfo.Name, info.ServerInfo.Version, server.BinaryPath)
	for _, t := range listed.Tools {
		fmt.Printf("  %-24s %s\n", t.Name, firstLine(t.Description))
	}
	return nil
}
