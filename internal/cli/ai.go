package cli

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"mhtodo/internal/store"
)

// IntegrationVersion is the agent-integration contract version emitted by
// `mhtodo ai`. Bump when §3/§4 behavioural rules change in a way that
// upgrades must notice — independent of the binary version.
const IntegrationVersion = 8

// integrationChangelog is rendered into §9 of the ai document. Newest first.
const integrationChangelog = `v8  Task cwd (--cwd) and human_only (--human-only on add/edit). Default list
    excludes human-only tasks; pass --human-only on list to include them. Agents
    must never adopt or work human_only tasks.
v7  Task-picker options show status, updated_at, title, optional description —
    not task ids in the labels.
v6  Task-picker turns ("what's next", todos, etc.) must use AskUserQuestion /
    AskQuestion with every root task from mhtodo list --roots --json in board
    order (status visible, no grouping or auto-pick). REVERSES v3-v5 grouped
    prose list — see §6.
v5  Sub-tasks are a mandatory step plan on root-task start (3+ steps), not optional
    mini-lifecycles. Activities attach to the step being worked. Blocking sets
    parent waiting (sub-tasks never waiting/review). All sub-tasks must be done
    before parent review. Parallel sub-task wip for subagents. REVERSES v3-v4
    "own lifecycle" / "as useful" wording — see §6.
v4  Activity labels are Title Case noun phrases (2-4 words), not sentences; all
    detail moves to --comment. REVERSES v3's "do not post per tool call": the
    unit is now one activity per step forward, and fine-grained is preferred.
    Upgraders must delete the old wording, not just add the new — see §6.
    Feedback (--feedback) is a short post-work summary + notes/takeaways at
    hand-back. Description, feedback, and activity comments are markdown in the GUI.
v3  Sub-tasks (--parent, one level) and activities. review status added.
    Ownership split: never rewrite a user-authored title or description.
    Search-then-adopt replaces register-always. Comment relay from the GUI.
v2  waiting status; SessionEnd ghost cleanup.
v1  Initial: register / edit / done, description as status field.`

//go:embed ai.md
var aiFS embed.FS

// aiNow is the clock used for the Generated header (tests inject a fixed time).
var aiNow = func() time.Time { return time.Now().UTC() }

// AINowForTest swaps the ai document clock and returns a restore func.
func AINowForTest(f func() time.Time) (restore func()) {
	prev := aiNow
	aiNow = f
	return func() { aiNow = prev }
}

// aiDoc is the --json envelope for `mhtodo ai`.
type aiDoc struct {
	IntegrationVersion int    `json:"integration_version"`
	MhtodoVersion      string `json:"mhtodo_version"`
	DBPath             string `json:"db_path"`
	Generated          string `json:"generated"`
	Content            string `json:"content"`
}

func statusEnum() string {
	return "pending|wip|waiting|review|done"
}

func renderAIDoc(version string) (aiDoc, error) {
	raw, err := aiFS.ReadFile("ai.md")
	if err != nil {
		return aiDoc{}, fmt.Errorf("read embedded ai.md: %w", err)
	}
	ts := aiNow().Format(time.RFC3339)
	db := store.DBPath()
	repl := map[string]string{
		"{{INTEGRATION_VERSION}}":   fmt.Sprintf("%d", IntegrationVersion),
		"{{MHTODO_VERSION}}":        version,
		"{{MHTODO_DB_PATH}}":        db,
		"{{TIMESTAMP}}":             ts,
		"{{STATUS_ENUM}}":           statusEnum(),
		"{{SORT_FIELDS}}":           strings.Join(sortFields, "|"),
		"{{INTEGRATION_CHANGELOG}}": integrationChangelog,
	}
	content := string(raw)
	for k, v := range repl {
		content = strings.ReplaceAll(content, k, v)
	}
	return aiDoc{
		IntegrationVersion: IntegrationVersion,
		MhtodoVersion:      version,
		DBPath:             db,
		Generated:          ts,
		Content:            content,
	}, nil
}

func newAICmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "ai",
		Short: "Print agent integration instructions (install/upgrade contract)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			doc, err := renderAIDoc(version)
			if err != nil {
				return &errExit{code: ExitStorage, name: "storage", msg: err.Error()}
			}
			if o.json {
				return o.printJSON(doc)
			}
			_, err = fmt.Fprint(o.out, doc.Content)
			return err
		},
	}
}
