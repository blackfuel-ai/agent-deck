package tmux

import (
	"testing"
)

// --- Busy regex tests (patterns.go fix) ---

func TestBusyRegex_DoesNotMatchStartupBanner(t *testing.T) {
	// The Claude Code startup banner contains · and … which previously
	// false-matched the busy regex, causing GetStatus() to return "active"
	// for an idle session.
	raw := DefaultRawPatterns("claude")
	resolved, err := CompilePatterns(raw)
	if err != nil {
		t.Fatalf("CompilePatterns: %v", err)
	}

	bannerLines := []string{
		`  ▘▘ ▝▝    Opus 4.6 is here · $50 free extra usage · Try fast mode or use i…`,
		` ▐▛███▜▌   Opus 4.6 · Claude Max`,
		`▝▜█████▛▘  ~/.agent-deck/conductor/sre`,
	}
	content := joinLines(bannerLines)

	for _, re := range resolved.BusyRegexps {
		if re.MatchString(content) {
			t.Errorf("busy regex %q should NOT match startup banner, but did", re.String())
		}
	}
}

func TestBusyRegex_MatchesRealWorkingIndicator(t *testing.T) {
	raw := DefaultRawPatterns("claude")
	resolved, err := CompilePatterns(raw)
	if err != nil {
		t.Fatalf("CompilePatterns: %v", err)
	}

	// Real working indicators have spinner char at line start
	workingLines := []struct {
		name    string
		content string
	}{
		{"asterisk spinner", "  ✳ Reading file…"},
		{"cross spinner", "✢ Clauding…"},
		{"dot spinner", "  · Thinking…"},
		{"star spinner", "✽ Brewing…"},
	}

	for _, tc := range workingLines {
		matched := false
		for _, re := range resolved.BusyRegexps {
			if re.MatchString(tc.content) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("busy regex should match %s: %q", tc.name, tc.content)
		}
	}
}

func TestBusyRegex_BannerWithWorkingIndicator(t *testing.T) {
	// When both banner and working indicator are present (multiline content),
	// the regex should match the working indicator line but not the banner.
	raw := DefaultRawPatterns("claude")
	resolved, err := CompilePatterns(raw)
	if err != nil {
		t.Fatalf("CompilePatterns: %v", err)
	}

	content := joinLines([]string{
		`  ▘▘ ▝▝    Opus 4.6 is here · $50 free extra usage · Try fast mode or use i…`,
		``,
		`✳ Reading file…`,
	})

	matched := false
	for _, re := range resolved.BusyRegexps {
		if re.MatchString(content) {
			matched = true
			break
		}
	}
	if !matched {
		t.Error("busy regex should match content containing a real working indicator even with banner present")
	}
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}
