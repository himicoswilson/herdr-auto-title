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

	// Pi titles its terminal `π - <name> - <directory>`, and the directory
	// only repeats what the context half of a title already says.
	if pane.Agent == "pi" {
		title = withoutPiDirectory(title, pane.Dir)
	}

	// An agent is a field of its own rather than a bound kind, so it stays.
	if s.unbound && !pane.HasAgent() {
		return activityAs(pane, title, "")
	}

	return activityFrom(pane, title)
}

// withoutPiDirectory drops the trailing directory from a Pi terminal
// title. A title without a name in it — `π - <directory>` alone — keeps
// its directory: it is all the title says. A last part that is not the
// pane's own directory is part of the name and stays too.
func withoutPiDirectory(title, dir string) string {
	rest, named := strings.CutPrefix(title, "π - ")
	if !named || dir == "" {
		return title
	}

	last := strings.LastIndex(rest, " - ")
	if last < 0 || rest[last+len(" - "):] != filepath.Base(dir) {
		return title
	}

	return "π - " + rest[:last]
}
