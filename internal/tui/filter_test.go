package tui_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/lyarwood/acptui/internal/ambient"
	"github.com/lyarwood/acptui/internal/tui"
)

var _ = Describe("Filter", func() {
	session := ambient.Session{
		Name:      "fix-login-bug",
		ProjectID: "my-project",
		Phase:     "Running",
		LlmModel:  "claude-sonnet-4",
		RepoURL:   "https://github.com/org/repo",
		Prompt:    "Fix the login flow authentication issue",
	}

	Describe("MatchSession", func() {
		It("matches by name", func() {
			Expect(tui.MatchSession(session, "fix-login")).To(BeTrue())
		})

		It("matches by prompt", func() {
			Expect(tui.MatchSession(session, "authentication")).To(BeTrue())
		})

		It("matches by project ID", func() {
			Expect(tui.MatchSession(session, "my-project")).To(BeTrue())
		})

		It("matches by model", func() {
			Expect(tui.MatchSession(session, "sonnet")).To(BeTrue())
		})

		It("matches by repo URL", func() {
			Expect(tui.MatchSession(session, "org/repo")).To(BeTrue())
		})

		It("does not match unrelated text", func() {
			Expect(tui.MatchSession(session, "nonexistent")).To(BeFalse())
		})

		It("is case insensitive", func() {
			Expect(tui.MatchSession(session, "FIX-LOGIN")).To(BeTrue())
		})

		It("supports regex", func() {
			Expect(tui.MatchSession(session, "fix-.*-bug")).To(BeTrue())
		})
	})

	Describe("prefix filters", func() {
		It("matches project: prefix", func() {
			Expect(tui.MatchSession(session, "project:my-project")).To(BeTrue())
			Expect(tui.MatchSession(session, "project:other")).To(BeFalse())
		})

		It("matches phase: prefix", func() {
			Expect(tui.MatchSession(session, "phase:Running")).To(BeTrue())
			Expect(tui.MatchSession(session, "phase:Stopped")).To(BeFalse())
		})

		It("matches model: prefix", func() {
			Expect(tui.MatchSession(session, "model:sonnet")).To(BeTrue())
			Expect(tui.MatchSession(session, "model:opus")).To(BeFalse())
		})

		It("matches repo: prefix", func() {
			Expect(tui.MatchSession(session, "repo:github")).To(BeTrue())
			Expect(tui.MatchSession(session, "repo:gitlab")).To(BeFalse())
		})
	})

	Describe("combined filters", func() {
		It("ANDs multiple space-separated terms", func() {
			Expect(tui.MatchSession(session, "project:my-project phase:Running")).To(BeTrue())
			Expect(tui.MatchSession(session, "project:my-project phase:Stopped")).To(BeFalse())
		})

		It("ANDs bare text with prefix", func() {
			Expect(tui.MatchSession(session, "fix project:my-project")).To(BeTrue())
			Expect(tui.MatchSession(session, "nonexistent project:my-project")).To(BeFalse())
		})
	})

	Describe("MatchSessions", func() {
		sessions := []ambient.Session{
			{Name: "session-a", Phase: "Running", ProjectID: "proj-1"},
			{Name: "session-b", Phase: "Stopped", ProjectID: "proj-1"},
			{Name: "session-c", Phase: "Running", ProjectID: "proj-2"},
		}

		It("filters to matching sessions", func() {
			result := tui.MatchSessions(sessions, "phase:Running")
			Expect(result).To(HaveLen(2))
		})

		It("returns empty for no match", func() {
			result := tui.MatchSessions(sessions, "phase:Failed")
			Expect(result).To(BeEmpty())
		})

		It("returns all for empty filter", func() {
			result := tui.MatchSessions(sessions, "")
			Expect(result).To(HaveLen(3))
		})
	})
})
