package lifecycle

import (
	"strings"
	"testing"

	"github.com/aoagents/agent-orchestrator/backend/internal/domain"
	"github.com/aoagents/agent-orchestrator/backend/internal/ports"
)

// The AO reviewer posts with the PR author's account. A finding it adds to an
// existing thread looks like an author reply, but it belongs to an AO review
// run and must still reach the worker; the worker's own reply must not.
func TestPRObservation_AOReviewCommentInExistingThreadIsForwarded(t *testing.T) {
	m, st, msg := newNoticeManager(t)
	st.prs["w-1"] = []domain.PullRequest{{URL: donePR, Author: "me"}}
	st.runs = []domain.ReviewRun{doneRun("run-1", domain.ReviewRunCompleted, domain.VerdictChangesRequested, "777", "Summary")}
	st.reviews[donePR] = []domain.PullRequestReview{observedReview("777")}
	st.comments[donePR] = []domain.PullRequestComment{
		{ID: "c1", ThreadID: "T1", ReviewID: "100", Author: "someone", File: "a.go", Line: 3, Body: "nil check", AutoInjectReview: true},
		{ID: "r1", ThreadID: "T1", ReviewID: "101", Author: "me", File: "a.go", Line: 3, Body: "Fixed in abc", AutoInjectReview: true},
		{ID: "c2", ThreadID: "T1", ReviewID: "777", Author: "me", File: "a.go", Line: 3, Body: "still wrong after fix", AutoInjectReview: true},
	}
	if err := m.ApplyPRObservation(ctx, "w-1", ports.PRObservation{Fetched: true, URL: donePR}); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(msg.msgs, "\n---\n")
	if !strings.Contains(all, "still wrong after fix") {
		t.Fatalf("AO review comment in an existing thread was dropped:\n%s", all)
	}
	if strings.Contains(all, "Fixed in abc") {
		t.Fatalf("the PR author's own reply was injected:\n%s", all)
	}
}

func TestWithoutPRAuthorRepliesKeepsComments(t *testing.T) {
	comments := []domain.PullRequestComment{
		{ID: "1", ThreadID: "T1", Author: "alice"},
		{ID: "2", ThreadID: "T1", Author: "me", ReviewID: "9"},
		{ID: "3", ThreadID: "T1", Author: "me", ReviewID: "10"},
	}
	keep := func(c domain.PullRequestComment) bool { return c.ReviewID == "9" }
	got := withoutPRAuthorReplies(comments, "me", keep)
	if len(got) != 2 || got[1].ID != "2" {
		t.Fatalf("got %v, want comments 1 and 2", got)
	}
}
