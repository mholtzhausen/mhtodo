package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
	"mhtodo/internal/store"
)

// Exit codes (stable agent contract — see .agent/plan/04-cli-spec.md).
const (
	ExitOK       = 0
	ExitUsage    = 1 // usage/validation error
	ExitNotFound = 2 // not found / ambiguous ID
	ExitStorage  = 3 // storage error
)

// errExit carries the process exit code plus a stable machine-readable error
// name for the --json stderr envelope {"error":<name>,"message":...}.
type errExit struct {
	code int
	name string
	msg  string
}

func (e *errExit) Error() string { return e.msg }

// mapError converts core/store errors into exit-coded CLI errors. Anything
// that is not a known domain error is treated as a storage error (exit 3).
func mapError(err error) error {
	var amb *core.AmbiguousIDError
	switch {
	case errors.Is(err, core.ErrNotFound):
		return &errExit{code: ExitNotFound, name: "not_found", msg: err.Error()}
	case errors.As(err, &amb):
		return &errExit{code: ExitNotFound, name: "ambiguous_id", msg: err.Error()}
	case errors.Is(err, core.ErrEmptyTitle):
		return &errExit{code: ExitUsage, name: "empty_title", msg: err.Error()}
	default:
		var (
			ise *core.InvalidStatusError
			pre *core.ProgressRangeError
		)
		switch {
		case errors.As(err, &ise):
			return &errExit{code: ExitUsage, name: "invalid_status", msg: err.Error()}
		case errors.As(err, &pre):
			return &errExit{code: ExitUsage, name: "progress_range", msg: err.Error()}
		case errors.Is(err, core.ErrNoFieldsToUpdate):
			return &errExit{code: ExitUsage, name: "no_fields", msg: err.Error()}
		case errors.Is(err, core.ErrNotArchived):
			return &errExit{code: ExitUsage, name: "not_archived", msg: err.Error()}
		case errors.Is(err, core.ErrNotDone):
			return &errExit{code: ExitUsage, name: "not_done", msg: err.Error()}
		case errors.Is(err, core.ErrAlreadyArchived):
			return &errExit{code: ExitUsage, name: "already_archived", msg: err.Error()}
		case errors.Is(err, core.ErrParentIsChild):
			return &errExit{code: ExitUsage, name: "parent_is_child", msg: err.Error()}
		case errors.Is(err, core.ErrNotRoot):
			return &errExit{code: ExitUsage, name: "not_root", msg: err.Error()}
		case errors.Is(err, core.ErrReorderStatusMismatch):
			return &errExit{code: ExitUsage, name: "reorder_status_mismatch", msg: err.Error()}
		case errors.Is(err, core.ErrEmptyActivity):
			return &errExit{code: ExitUsage, name: "empty_activity", msg: err.Error()}
		default:
			return &errExit{code: ExitStorage, name: "storage", msg: err.Error()}
		}
	}
}

// usageError is a flag/argument validation failure (exit 1).
func usageError(format string, a ...any) error {
	return &errExit{code: ExitUsage, name: "usage", msg: fmt.Sprintf(format, a...)}
}

// Stdin is the test seam for rm's interactive confirmation. Tests replace it
// with an in-memory reader (which is never a TTY).
var Stdin io.Reader = os.Stdin

type ctxOutKey struct{}

type opts struct {
	json  bool
	quiet bool
	out   io.Writer
}

func o(cmd *cobra.Command) (opts, error) {
	j, err := cmd.Flags().GetBool("json")
	if err != nil {
		return opts{}, err
	}
	q, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		return opts{}, err
	}
	out, _ := cmd.Context().Value(ctxOutKey{}).(io.Writer)
	if out == nil {
		out = os.Stdout
	}
	return opts{json: j, quiet: q, out: out}, nil
}

// openService opens the DB (WAL + migrations) and returns a ready Service.
func openService() (*core.Service, func(), error) {
	repo, err := store.Open(store.DBPath())
	if err != nil {
		return nil, nil, mapError(err)
	}
	svc := core.NewService(repo)
	return svc, func() { repo.Close() }, nil
}

// --- output helpers -----------------------------------------------------------

func (o opts) printTask(t core.Task) error {
	switch {
	case o.json:
		return o.printJSON(t)
	case o.quiet:
		_, err := fmt.Fprintf(o.out, "%s  %s  %d%%  %s\n", t.ID, t.Status, t.Progress, t.Title)
		return err
	default:
		w := newTabWriter(o.out)
		fmt.Fprintf(w, "ID         \t%s\n", t.ID)
		fmt.Fprintf(w, "Title      \t%s\n", t.Title)
		if t.Description != "" {
			fmt.Fprintf(w, "Description\t%s\n", t.Description)
		}
		if t.Feedback != "" {
			fmt.Fprintf(w, "Feedback   \t%s\n", t.Feedback)
		}
		if notice := core.SlackThreadNotice(t.SlackThread); notice != "" {
			fmt.Fprintf(w, "Slack thread\t%s\n", notice)
		}
		fmt.Fprintf(w, "Status     \t%s\n", t.Status)
		fmt.Fprintf(w, "Progress   \t%d%%\n", t.Progress)
		if t.ParentID != nil {
			fmt.Fprintf(w, "Parent     \t%s\n", *t.ParentID)
		}
		fmt.Fprintf(w, "Created    \t%s\n", absTS(t.CreatedAt))
		fmt.Fprintf(w, "Updated    \t%s\n", absTS(t.UpdatedAt))
		if t.CompletedAt != nil {
			fmt.Fprintf(w, "Completed  \t%s\n", absTS(*t.CompletedAt))
		} else {
			fmt.Fprintf(w, "Completed  \t—\n")
		}
		return w.Flush()
	}
}

func (o opts) printTasks(tasks []core.Task) error {
	if o.json {
		if tasks == nil {
			tasks = []core.Task{} // marshal as [], not null
		}
		return o.printJSON(tasks)
	}
	w := newTabWriter(o.out)
	for _, t := range tasks {
		fmt.Fprintf(w, "%s\t%s\t%d%%\t%s (%s)\t%s\n",
			shortID(t.ID), t.Status, t.Progress, relTime(t.UpdatedAt), absShort(t.UpdatedAt), t.Title)
	}
	return w.Flush()
}

func (o opts) printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(o.out, "%s\n", b)
	return err
}

// --- time formatting ------------------------------------------------------------

func absTS(t time.Time) string    { return t.UTC().Format(time.RFC3339) }
func absShort(t time.Time) string { return t.UTC().Format("2006-01-02 15:04") }

// relTime is the coarse relative age used in list output.
func relTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < 45*time.Second:
		return "just now"
	case d < 90*time.Minute:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func shortID(id string) string {
	return core.ShortID(id)
}

// newTabWriter builds the aligned-column writer used by human output.
func newTabWriter(w io.Writer) *tabwriter.Writer { return tabwriter.NewWriter(w, 0, 4, 2, ' ', 0) }

// --- command tree + entrypoint -------------------------------------------------

// NewRootCmd builds the CLI command tree. It deliberately imports no GUI code:
// main.go dispatches bare `mhtodo` / `mhtodo gui` to Wails before this runs.
func NewRootCmd(version, commit string) *cobra.Command {
	root := &cobra.Command{
		Use:           "mhtodo",
		Short:         "Personal todo manager (CLI = agentic interface; bare mhtodo opens the GUI)",
		Version:       version,
		SilenceErrors: true, // we format errors ourselves ("mhtodo: <msg>" / JSON envelope)
		SilenceUsage:  true,
	}
	if commit != "" && commit != "none" {
		root.Version = version + " (commit " + commit + ")"
	}
	// Keep the command surface exactly as specified (04-cli-spec.md): no
	// auto-added `completion` subcommand; `help` stays callable but hidden.
	root.CompletionOptions.DisableDefaultCmd = true
	root.PersistentFlags().Bool("json", false, "emit JSON instead of human format")
	root.PersistentFlags().BoolP("quiet", "q", false, "suppress non-essential output")

	for _, c := range []*cobra.Command{
		newAddCmd(), newListCmd(), newShowCmd(), newEditCmd(),
		newStatusCmd(), newDoneCmd(), newArchiveCmd(), newUnarchiveCmd(), newReorderCmd(),
		newActivityCmd(), newRmCmd(), newPathCmd(), newSlackCmd(), newAICmd(version),
		newUpdateCmd(version),
	} {
		root.AddCommand(c)
	}
	root.InitDefaultHelpCmd() // then hide it so `help` stays callable but off the contract surface
	for _, c := range root.Commands() {
		if c.Name() == "help" {
			c.Hidden = true
		}
	}
	return root
}

// Execute runs the CLI, writing to stdout/stderr, and returns the process exit
// code (0 ok, 1 usage/validation, 2 not-found/ambiguous, 3 storage).
func Execute(args []string, version, commit string, stdout, stderr io.Writer) int {
	root := NewRootCmd(version, commit)
	// Subcommands inherit this context; o(cmd) pulls the stdout writer from it.
	root.SetContext(context.WithValue(context.Background(), ctxOutKey{}, stdout))
	root.SetArgs(args)
	root.SetOut(stdout)
	root.SetErr(stderr)

	err := root.Execute()
	if err == nil {
		return ExitOK
	}
	var ee *errExit
	if errors.As(err, &ee) {
		writeError(stderr, containsFlag(args, "--json"), ee.name, ee.msg)
		return ee.code
	}
	// Cobra flag/usage error (unknown command, bad flag value): exit 1.
	msg := strings.TrimSpace(err.Error())
	if strings.HasPrefix(msg, "unknown command") {
		names := make([]string, 0, len(root.Commands()))
		for _, c := range root.Commands() {
			if !c.Hidden && !c.IsAdditionalHelpTopicCommand() {
				names = append(names, c.Name())
			}
		}
		sort.Strings(names)
		msg += "; available commands: " + strings.Join(names, ", ")
	}
	writeError(stderr, containsFlag(args, "--json"), "usage", msg)
	return ExitUsage
}

// Run is the os.Stdout/os.Stderr convenience wrapper used by main.
func Run(args []string, version, commit string) int {
	return Execute(args, version, commit, os.Stdout, os.Stderr)
}

func writeError(w io.Writer, jsonMode bool, name, msg string) {
	if jsonMode {
		b, err := json.Marshal(map[string]string{"error": name, "message": msg})
		if err != nil {
			fmt.Fprintf(w, "mhtodo: %s\n", msg) // unmarshalable message; fall back to plain
			return
		}
		fmt.Fprintln(w, string(b))
		return
	}
	fmt.Fprintf(w, "mhtodo: %s\n", msg)
}

func containsFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}
