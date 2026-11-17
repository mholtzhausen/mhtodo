package core

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SlackLink is one Slack conversation attached to a ticket (v21). The first
// link in Task.SlackLinks is the primary thread and is mirrored in
// Task.SlackThread, so the pre-v21 `slack_thread` contract keeps working.
type SlackLink struct {
	URL         string `json:"url"`
	Description string `json:"description"` // optional; "" = none
}

// SlackLinkInfo describes what a Slack URL points at (display only; no API).
type SlackLinkInfo struct {
	Valid     bool   `json:"valid"`
	Kind      string `json:"kind"`      // thread|channel|dm|group|workspace|link
	Workspace string `json:"workspace"` // e.g. "acme" for acme.slack.com; "" when unknown
	Label     string `json:"label"`     // short human label, e.g. "Thread · acme"
	Error     string `json:"error"`     // why it is invalid (Valid=false)
}

// InvalidSlackLinkError is returned for a URL that is not a Slack link.
type InvalidSlackLinkError struct{ URL, Reason string }

func (e *InvalidSlackLinkError) Error() string {
	return fmt.Sprintf("invalid slack link %q: %s", e.URL, e.Reason)
}

// SlackLinkNotFoundError is returned when rm/edit names a link the ticket lacks.
type SlackLinkNotFoundError struct{ Ref string }

func (e *SlackLinkNotFoundError) Error() string {
	return fmt.Sprintf("slack link %q not found on this ticket (use a URL or a 1-based index from `slack link list`)", e.Ref)
}

// DescribeSlackLink recognises Slack URLs: https://<ws>.slack.com/… (and
// app.slack.com/client/…) and slack:// deep links. Anything else is invalid.
func DescribeSlackLink(raw string) SlackLinkInfo {
	s := strings.TrimSpace(raw)
	if s == "" {
		return SlackLinkInfo{Error: "empty"}
	}
	u, err := url.Parse(s)
	if err != nil {
		return SlackLinkInfo{Error: "not a URL"}
	}
	info := SlackLinkInfo{Valid: true, Kind: "link"}
	switch strings.ToLower(u.Scheme) {
	case "slack":
		q := u.Query()
		info.Workspace = q.Get("team")
		switch strings.ToLower(u.Host) {
		case "channel":
			info.Kind = channelKind(q.Get("id"))
		case "user":
			info.Kind = "dm"
		case "open":
			info.Kind = "workspace"
		}
	case "http", "https":
		host := strings.ToLower(u.Hostname())
		if host != "slack.com" && !strings.HasSuffix(host, ".slack.com") {
			return SlackLinkInfo{Error: "not a slack.com link"}
		}
		if ws := strings.TrimSuffix(host, ".slack.com"); ws != host && ws != "app" && ws != "www" {
			info.Workspace = strings.TrimSuffix(ws, ".enterprise")
		}
		parts := strings.FieldsFunc(u.Path, func(r rune) bool { return r == '/' })
		// /archives/<C…>[/p<ts>] or /client/<T…>/<C…>[/thread/<C…>-<ts>]
		var ch string
		msg, thread := false, false
		for i, p := range parts {
			if (p == "archives" || p == "client") && i+1 < len(parts) {
				ch = parts[i+1]
				if p == "client" && i+2 < len(parts) {
					if info.Workspace == "" {
						info.Workspace = parts[i+1]
					}
					ch = parts[i+2]
				}
			}
			if strings.HasPrefix(p, "p") && len(p) > 1 && isDigits(p[1:]) {
				msg = true
			}
			if p == "thread" {
				thread = true
			}
		}
		if ch != "" {
			info.Kind = channelKind(ch)
		}
		if msg || thread {
			// A message permalink is how Slack shares a thread (thread_ts marks
			// a reply inside one); either way people paste it as "the thread".
			info.Kind = "thread"
		}
	default:
		return SlackLinkInfo{Error: "not a Slack link (want https://….slack.com/… or slack://…)"}
	}
	label := map[string]string{
		"thread": "Thread", "channel": "Channel", "dm": "Direct message",
		"group": "Group DM", "workspace": "Workspace", "link": "Slack link",
	}[info.Kind]
	if info.Workspace != "" {
		label += " · " + info.Workspace
	}
	info.Label = label
	return info
}

func channelKind(id string) string {
	switch {
	case strings.HasPrefix(id, "D"):
		return "dm"
	case strings.HasPrefix(id, "G"):
		return "group"
	case id != "":
		return "channel"
	}
	return "link"
}

// slackKey is the dedupe key: scheme/host case-folded, trailing slash and
// fragment dropped. Query is kept (thread_ts distinguishes replies).
func slackKey(raw string) string {
	s := strings.TrimSpace(raw)
	u, err := url.Parse(s)
	if err != nil {
		return strings.ToLower(s)
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/")
	return u.String()
}

// NormalizeSlackLinks trims, drops empty URLs and removes duplicates (first
// occurrence wins; a later duplicate's description fills an empty one).
// Always returns a non-nil slice.
func NormalizeSlackLinks(in []SlackLink) []SlackLink {
	out := make([]SlackLink, 0, len(in))
	idx := map[string]int{}
	for _, l := range in {
		u := strings.TrimSpace(l.URL)
		if u == "" {
			continue
		}
		d := strings.TrimSpace(strings.ReplaceAll(l.Description, "\n", " "))
		k := slackKey(u)
		if i, ok := idx[k]; ok {
			if out[i].Description == "" {
				out[i].Description = d
			}
			continue
		}
		idx[k] = len(out)
		out = append(out, SlackLink{URL: u, Description: d})
	}
	return out
}

// syncSlack normalizes t.SlackLinks and mirrors the primary into SlackThread.
func syncSlack(t *Task) {
	t.SlackLinks = NormalizeSlackLinks(t.SlackLinks)
	t.SlackThread = ""
	if len(t.SlackLinks) > 0 {
		t.SlackThread = t.SlackLinks[0].URL
	}
}

// promoteSlackThread makes url the primary link: moved to the front when
// already present (keeping its description), inserted otherwise. Nothing is
// dropped. url == "" removes the current primary (the next link is promoted).
func promoteSlackThread(links []SlackLink, raw string) []SlackLink {
	u := strings.TrimSpace(raw)
	if u == "" {
		if len(links) == 0 {
			return links
		}
		return append([]SlackLink{}, links[1:]...)
	}
	k := slackKey(u)
	out := []SlackLink{{URL: u}}
	for _, l := range links {
		if slackKey(l.URL) == k {
			out[0].Description = l.Description
			continue
		}
		out = append(out, l)
	}
	return out
}

// findSlackLink resolves a URL or 1-based index to a position.
func findSlackLink(links []SlackLink, ref string) (int, error) {
	r := strings.TrimSpace(ref)
	if n, err := strconv.Atoi(r); err == nil {
		if n >= 1 && n <= len(links) {
			return n - 1, nil
		}
		return -1, &SlackLinkNotFoundError{Ref: ref}
	}
	k := slackKey(r)
	for i, l := range links {
		if slackKey(l.URL) == k {
			return i, nil
		}
	}
	return -1, &SlackLinkNotFoundError{Ref: ref}
}

func validateSlackURL(raw string) error {
	if info := DescribeSlackLink(raw); !info.Valid {
		return &InvalidSlackLinkError{URL: raw, Reason: info.Error}
	}
	return nil
}

// AddSlackLink attaches a Slack conversation (validated). primary=true puts
// it first. Adding a URL that is already linked updates its description when
// one is given (and moves it when primary) instead of duplicating it.
func (s *Service) AddSlackLink(ctx context.Context, ref string, link SlackLink, primary bool) (Task, error) {
	if err := validateSlackURL(link.URL); err != nil {
		return Task{}, err
	}
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	links := append([]SlackLink{}, t.SlackLinks...)
	link.URL = strings.TrimSpace(link.URL)
	if i, err := findSlackLink(links, link.URL); err == nil {
		if strings.TrimSpace(link.Description) != "" {
			links[i].Description = link.Description
		}
		if primary && i > 0 {
			l := links[i]
			links = append(links[:i], links[i+1:]...)
			links = append([]SlackLink{l}, links...)
		}
	} else if primary {
		links = append([]SlackLink{link}, links...)
	} else {
		links = append(links, link)
	}
	return s.SetSlackLinks(ctx, t.ID, links)
}

// EditSlackLink changes one link's description and/or makes it primary.
func (s *Service) EditSlackLink(ctx context.Context, ref, linkRef string, desc *string, primary bool) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	links := append([]SlackLink{}, t.SlackLinks...)
	i, err := findSlackLink(links, linkRef)
	if err != nil {
		return Task{}, err
	}
	if desc != nil {
		links[i].Description = *desc
	}
	if primary && i > 0 {
		l := links[i]
		links = append(links[:i], links[i+1:]...)
		links = append([]SlackLink{l}, links...)
	}
	return s.SetSlackLinks(ctx, t.ID, links)
}

// RemoveSlackLink drops one link by URL or 1-based index; removing the
// primary promotes the next one.
func (s *Service) RemoveSlackLink(ctx context.Context, ref, linkRef string) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	links := append([]SlackLink{}, t.SlackLinks...)
	i, err := findSlackLink(links, linkRef)
	if err != nil {
		return Task{}, err
	}
	links = append(links[:i], links[i+1:]...)
	return s.SetSlackLinks(ctx, t.ID, links)
}

// SetSlackLinks replaces the full ordered list (GUI reorder/edit). New URLs
// must be Slack links; ones already on the ticket are kept as-is so legacy
// non-Slack slack_thread values survive a reorder.
func (s *Service) SetSlackLinks(ctx context.Context, ref string, links []SlackLink) (Task, error) {
	t, err := s.Get(ctx, ref)
	if err != nil {
		return Task{}, err
	}
	for _, l := range links {
		if strings.TrimSpace(l.URL) == "" {
			continue
		}
		if _, err := findSlackLink(t.SlackLinks, l.URL); err == nil {
			continue
		}
		if err := validateSlackURL(l.URL); err != nil {
			return Task{}, err
		}
	}
	cp := append([]SlackLink{}, links...)
	return s.Edit(ctx, t.ID, UpdateInput{SlackLinks: &cp})
}

// SlackLinksNotice is the agent-facing Slack block: the primary-thread line
// (unchanged wording) plus one line per other conversation. "" when none.
func SlackLinksNotice(t Task) string {
	links := t.SlackLinks
	if len(links) == 0 && strings.TrimSpace(t.SlackThread) != "" {
		links = []SlackLink{{URL: t.SlackThread}}
	}
	if len(links) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(SlackThreadNotice(links[0].URL))
	if d := links[0].Description; d != "" {
		b.WriteString(" (" + d + ")")
	}
	if len(links) > 1 {
		b.WriteString("\nother slack conversations for this ticket:")
		for _, l := range links[1:] {
			b.WriteString("\n- " + l.URL)
			if l.Description != "" {
				b.WriteString(" — " + l.Description)
			}
		}
	}
	return b.String()
}
