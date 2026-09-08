package cli

import (
	"context"

	"mhtodo/internal/core"
	"mhtodo/internal/integrations"
	"mhtodo/internal/settings"
)

func maybeCloseHerdrTabOnDone(svc *core.Service, prevStatus core.Status, t core.Task) {
	if prevStatus == core.StatusDone || t.Status != core.StatusDone {
		return
	}
	s, err := settings.Load(nil)
	if err != nil {
		return
	}
	client := integrations.Client{Herdr: s.Herdr, Claude: s.Claude, Terminal: s.Terminal}
	client.MaybeCloseSessionOnDone(t.ID, core.ShortID(t.ID), t.Title, t.TerminalPID, func() {
		if svc == nil {
			return
		}
		_, _ = svc.SetTerminalPID(context.Background(), t.ID, 0)
	})
}
