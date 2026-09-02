package cli

import (
	"mhtodo/internal/core"
	"mhtodo/internal/integrations"
	"mhtodo/internal/settings"
)

func maybeCloseHerdrTabOnDone(prevStatus core.Status, t core.Task) {
	if prevStatus == core.StatusDone || t.Status != core.StatusDone {
		return
	}
	s, err := settings.Load(nil)
	if err != nil {
		return
	}
	client := integrations.Client{Herdr: s.Herdr, Claude: s.Claude}
	client.MaybeCloseTicketTabOnDone(t.ID, core.ShortID(t.ID), t.Title)
}
