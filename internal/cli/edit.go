package cli

import (
	"context"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"mhtodo/internal/core"
)

func newEditCmd() *cobra.Command {
	var title, desc, feedback, cwd, slackThread, session, gitRepo, gitDetect string
	var prURLs, gitBranches []string
	var progress int
	var humanOnly, noHumanOnly bool
	var includeInReport, noIncludeInReport bool
	cmd := &cobra.Command{
		Use:   "edit ID",
		Short: "Update a task's title/description/feedback/progress/session (at least one flag)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o, err := o(cmd)
			if err != nil {
				return err
			}
			in := core.UpdateInput{}
			if cmd.Flags().Changed("title") {
				v := title
				in.Title = &v
			}
			if cmd.Flags().Changed("desc") {
				v := desc
				in.Desc = &v
			}
			if cmd.Flags().Changed("feedback") {
				v := feedback
				in.Feedback = &v
			}
			if cmd.Flags().Changed("progress") {
				v := progress
				in.Progress = &v
			}
			if cmd.Flags().Changed("cwd") {
				v := cwd
				in.Cwd = &v
			}
			if cmd.Flags().Changed("slack-thread") {
				v := slackThread
				in.SlackThread = &v
			}
			if cmd.Flags().Changed("pr-url") {
				v := core.JoinPRURLs(prURLs)
				in.PRURL = &v
			}
			if cmd.Flags().Changed("git-repo") {
				v := gitRepo
				in.GitRepo = &v
			}
			if cmd.Flags().Changed("git-branch") {
				v := core.JoinGitBranches(gitBranches)
				in.GitBranches = &v
			}
			if cmd.Flags().Changed("session") {
				v := session
				in.TodoSession = &v
			}
			switch {
			case cmd.Flags().Changed("human-only"):
				v := humanOnly
				in.HumanOnly = &v
			case cmd.Flags().Changed("no-human-only"):
				v := false
				in.HumanOnly = &v
			case cmd.Flags().Changed("include-in-report"):
				v := includeInReport
				in.IncludeInReport = &v
			case cmd.Flags().Changed("no-include-in-report"):
				v := false
				in.IncludeInReport = &v
			}

			svc, closeDB, err := openService()
			if err != nil {
				return err
			}
			defer closeDB()

			ctx := context.Background()
			detect := cmd.Flags().Changed("git-detect")
			var t core.Task
			if !detect || in.HasFields() {
				t, err = svc.Edit(ctx, args[0], in)
				if err != nil {
					return mapError(err) // no flags set → ErrNoFieldsToUpdate (exit 1)
				}
			}
			if detect {
				dir, err := detectDir(gitDetect)
				if err != nil {
					return err
				}
				t, err = svc.DetectGit(ctx, args[0], dir)
				if err != nil {
					return mapError(err)
				}
			}
			return o.printTask(t)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&desc, "desc", "", "new description")
	cmd.Flags().StringVar(&feedback, "feedback", "", "agent feedback (shown in GUI when set)")
	cmd.Flags().IntVar(&progress, "progress", 0, "new progress 0-100")
	cmd.Flags().StringVar(&cwd, "cwd", "", "working directory path (empty clears)")
	cmd.Flags().StringVar(&slackThread, "slack-thread", "", "Slack thread URL (empty clears)")
	cmd.Flags().StringArrayVar(&prURLs, "pr-url", nil, "pull request URL(s); repeatable; empty clears; setting advances status to pr")
	cmd.Flags().StringVar(&gitRepo, "git-repo", "", "git repository (URL or owner/name; empty clears)")
	cmd.Flags().StringArrayVar(&gitBranches, "git-branch", nil, "git branch name(s); repeatable (replaces the list); empty clears")
	cmd.Flags().StringVar(&gitDetect, "git-detect", "", "record repo + current branch; bare flag uses the current directory, --git-detect=DIR another; sets repo when empty, appends the branch")
	cmd.Flags().Lookup("git-detect").NoOptDefVal = "."
	cmd.Flags().StringVar(&session, "session", "", "Claude session UUID linked to this ticket (empty clears)")
	cmd.Flags().BoolVar(&humanOnly, "human-only", false, "mark as human-only")
	cmd.Flags().BoolVar(&noHumanOnly, "no-human-only", false, "clear human-only flag")
	cmd.Flags().BoolVar(&includeInReport, "include-in-report", false, "include in Slack board report")
	cmd.Flags().BoolVar(&noIncludeInReport, "no-include-in-report", false, "exclude from Slack board report")
	return cmd
}

// detectDir resolves the --git-detect value to an absolute directory
// ("." or empty → the process working directory).
func detectDir(v string) (string, error) {
	if v == "" || v == "." {
		wd, err := os.Getwd()
		if err != nil {
			return "", usageError("--git-detect: %v", err)
		}
		return wd, nil
	}
	abs, err := filepath.Abs(v)
	if err != nil {
		return "", usageError("--git-detect: %v", err)
	}
	return abs, nil
}
