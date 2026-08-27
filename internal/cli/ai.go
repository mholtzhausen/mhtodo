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
const IntegrationVersion = 3

// integrationChangelog is rendered into §9 of the ai document. Newest first.
const integrationChangelog = `v3  Sub-tasks (--parent, one level) and activities. review status added.
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
		"{{INTEGRATION_VERSION}}":  fmt.Sprintf("%d", IntegrationVersion),
		"{{MHTODO_VERSION}}":       version,
		"{{MHTODO_DB_PATH}}":       db,
		"{{TIMESTAMP}}":            ts,
		"{{STATUS_ENUM}}":          statusEnum(),
		"{{SORT_FIELDS}}":          strings.Join(sortFields, "|"),
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
