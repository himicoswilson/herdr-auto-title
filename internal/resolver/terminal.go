package resolver

import (
	"path/filepath"
	"strings"

	"github.com/kryptamine/herdr-auto-title/internal/state"
)

// TerminalTitle derives the activity from the pane's terminal title: a title a
// program went out of its way to set usually says what is happening. A lone
// program in the pane qualifies it, `nvim › auth.provider.ts`.
type TerminalTitle struct {
	// unbound reads the title without the program that set it, for a chain
	// that leaves the foreground process out and must not learn it from here.
	unbound bool
}

var _ Source = TerminalTitle{}

func NewTerminalTitle() TerminalTitle { return TerminalTitle{} }

// NewUnboundTerminalTitle reads the title as it is, `auth.ts` rather than
// `nvim › auth.ts`. topicChain builds on it, for the reason given there.
func NewUnboundTerminalTitle() TerminalTitle { return TerminalTitle{unbound: true} }

func (TerminalTitle) Name() string    { return "terminal_title" }
func (TerminalTitle) Confidence() int { return ConfidenceTerminalTitle }

func (s TerminalTitle) Resolve(pane *state.PaneState) (Parts, bool) {
	// Herdr strips escapes and decorative prefixes for us; the raw field is
	// only a fallback for when it has not.
	title := pane.TerminalTitle
	if title == "" {
		title = pane.TerminalTitleRaw
	}

	// A shell titles its window with the command it runs, so until the remote
	// shell sets a title this one only repeats the session the context names.
	if session, _, remote := remoteOf(pane); remote {
		title = session.withoutPrefix(title)
		if session.echoesCommand(title) {
			return Parts{}, false
		}
	}

	// Pi's title is taken whole, as the activity itself.
	if pane.Agent == "pi" {
		return piActivity(pane, title)
	}

	// An agent is a field of its own rather than a bound kind, so it stays.
	if s.unbound && !pane.HasAgent() {
		return activityAs(pane, title, "")
	}

	return activityFrom(pane, title)
}

// piActivity is a Pi pane's terminal title as its activity. It does not
// go through the kind handling activityFrom applies: Pi's title carries
// its name as a heading — `π - <name> - <directory>` — and the kind
// stripping would take the heading off once the mark is spelled out,
// mistaking it for the agent's name said twice. The checks are the
// ones activityAs makes; only the placing differs.
func piActivity(pane *state.PaneState, title string) (Parts, bool) {
	activity, ok := Meaningful(Sanitize(titledByPi(title, pane.Dir), 0))
	if !ok || echoesAgentName(pane, activity) {
		return Parts{}, false
	}

	return Parts{Activity: activity}, true
}

// titledByPi rewrites a Pi terminal title the way a tab reads it: the
// Greek letter becomes the word, and the trailing directory goes. A
// title without a name in it — the mark and a directory alone — keeps
// its directory: it is all the title says. A last part that is not the
// pane's own directory is part of the name and stays too.
func titledByPi(title, dir string) string {
	rest, pi := strings.CutPrefix(title, "π - ")
	if !pi {
		return title
	}

	if dir != "" {
		if last := strings.LastIndex(
			rest,
			" - ",
		); last >= 0 &&
			rest[last+len(" - "):] == filepath.Base(dir) {
			rest = rest[:last]
		}
	}

	return "Pi - " + rest
}
