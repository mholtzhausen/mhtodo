package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	shellBlockBegin = "# >>> mhtodo integration >>>"
	shellBlockEnd   = "# <<< mhtodo integration <<<"
)

// ClaudeTodoSnippet is the managed shell function body (between markers).
const ClaudeTodoSnippet = `claude.todo() {
  if [ -z "${MHTODO_SESSION:-}" ]; then
    echo "mhtodo: MHTODO_SESSION is not set" >&2
    return 1
  fi
  if command claude --resume "$MHTODO_SESSION" "$@"; then
    return 0
  fi
  command claude --name "$MHTODO_SESSION" "$@"
}`

func newIntegrationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integration",
		Short: "Install or remove shell helpers (claude.todo)",
	}
	cmd.AddCommand(newIntegrationShellCmd("bash", ".bashrc"))
	cmd.AddCommand(newIntegrationShellCmd("zsh", ".zshrc"))
	return cmd
}

func newIntegrationShellCmd(shell, rcName string) *cobra.Command {
	var remove bool
	cmd := &cobra.Command{
		Use:   shell,
		Short: fmt.Sprintf("Install or update the mhtodo block in ~/%s", rcName),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			path, err := shellRCPath(rcName)
			if err != nil {
				return err
			}
			if remove {
				changed, err := removeShellBlock(path)
				if err != nil {
					return err
				}
				if changed {
					_, err = fmt.Fprintf(o.out, "Removed mhtodo integration from %s\nRestart your shell or run: source %s\n", path, path)
				} else {
					_, err = fmt.Fprintf(o.out, "No mhtodo integration block in %s\n", path)
				}
				return err
			}
			if err := upsertShellBlock(path); err != nil {
				return err
			}
			_, err = fmt.Fprintf(o.out, "Updated mhtodo integration in %s\nRestart your shell or run: source %s\n", path, path)
			return err
		},
	}
	cmd.Flags().BoolVar(&remove, "remove", false, "remove the managed mhtodo block")
	return cmd
}

func shellRCPath(rcName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home: %w", err)
	}
	home = filepath.Clean(home)
	path := filepath.Join(home, rcName)
	// Refuse anything outside $HOME (symlink escape / odd paths).
	rel, err := filepath.Rel(home, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("refusing to write outside home: %s", path)
	}
	return path, nil
}

func managedBlock() string {
	return shellBlockBegin + "\n" + ClaudeTodoSnippet + "\n" + shellBlockEnd
}

func upsertShellBlock(path string) error {
	var content string
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		content = ""
	} else {
		content = string(data)
	}
	block := managedBlock()
	next, err := replaceOrAppendBlock(content, block)
	if err != nil {
		return err
	}
	if next == content {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(next), 0o600)
}

func removeShellBlock(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	next, removed, err := stripBlock(string(data))
	if err != nil {
		return false, err
	}
	if !removed {
		return false, nil
	}
	if err := os.WriteFile(path, []byte(next), 0o600); err != nil {
		return false, err
	}
	return true, nil
}

func replaceOrAppendBlock(content, block string) (string, error) {
	stripped, had, err := stripBlock(content)
	if err != nil {
		return "", err
	}
	stripped = strings.TrimRight(stripped, "\n")
	if stripped == "" {
		return block + "\n", nil
	}
	out := stripped + "\n\n" + block + "\n"
	if had {
		return out, nil
	}
	return out, nil
}

func stripBlock(content string) (string, bool, error) {
	begin := strings.Index(content, shellBlockBegin)
	if begin < 0 {
		return content, false, nil
	}
	end := strings.Index(content[begin:], shellBlockEnd)
	if end < 0 {
		return "", false, fmt.Errorf("malformed mhtodo integration block: missing end marker")
	}
	endAbs := begin + end + len(shellBlockEnd)
	// Drop trailing newline after end marker if present.
	if endAbs < len(content) && content[endAbs] == '\n' {
		endAbs++
	}
	// Drop a blank line immediately before the block when present.
	start := begin
	if start > 0 && content[start-1] == '\n' {
		start--
		if start > 0 && content[start-1] == '\n' {
			start--
		}
	}
	out := content[:start] + content[endAbs:]
	return out, true, nil
}
