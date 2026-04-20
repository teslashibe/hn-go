package hn

import (
	"os"
	"testing"
)

func loadTestData(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("failed to load testdata/%s: %v", name, err)
	}
	return data
}

// --- parseStories ---

func TestParseStories_FrontPage(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	if len(stories) == 0 {
		t.Fatal("expected stories, got 0")
	}
	if len(stories) < 25 {
		t.Errorf("expected ≥25 stories, got %d", len(stories))
	}

	first := stories[0]
	if first.ID == 0 {
		t.Error("first story has ID 0")
	}
	if first.Title == "" {
		t.Errorf("first story has empty title, ID=%d", first.ID)
	}
	if first.Rank != 1 {
		t.Errorf("first story rank = %d, want 1", first.Rank)
	}
	if first.Author == "" {
		t.Errorf("first story has empty author, ID=%d", first.ID)
	}

	t.Logf("parsed %d stories; first: [%d] %s by %s (%d pts)",
		len(stories), first.ID, first.Title, first.Author, first.Points)
}

func TestParseStories_AuthTokensPresent(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	hasVoteAuth := false
	hasHideAuth := false
	hasFaveAuth := false
	for _, s := range stories {
		if s.VoteUpAuth != "" {
			hasVoteAuth = true
		}
		if s.HideAuth != "" {
			hasHideAuth = true
		}
		if s.FaveAuth != "" {
			hasFaveAuth = true
		}
	}

	if !hasVoteAuth {
		t.Error("no story has VoteUpAuth — page might be unauthenticated")
	}
	if !hasHideAuth {
		t.Error("no story has HideAuth")
	}
	// FaveAuth is only on item pages, not feed pages — but HideAuth proves auth is working
	_ = hasFaveAuth
}

func TestParseStories_NoAuthPage(t *testing.T) {
	body := loadTestData(t, "frontpage_noauth.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	if len(stories) == 0 {
		t.Fatal("expected stories even without auth")
	}
	if stories[0].Title == "" {
		t.Error("story title should still be parseable without auth")
	}
}

func TestParseStories_RanksAreSequential(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	for i, s := range stories {
		if s.Rank != i+1 {
			t.Errorf("story %d: rank = %d, want %d", i, s.Rank, i+1)
		}
	}
}

func TestParseStories_TypeDetection(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	types := map[string]int{}
	for _, s := range stories {
		types[s.Type]++
	}
	t.Logf("type distribution: %v", types)

	if types["story"] == 0 {
		t.Error("expected at least one story-type item")
	}
}

func TestParseStories_AgeIsParsed(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	stories, err := parseStories(body)
	if err != nil {
		t.Fatalf("parseStories: %v", err)
	}

	for _, s := range stories {
		if s.Age.IsZero() {
			t.Errorf("story %d (%s): Age is zero", s.ID, s.Title)
			break
		}
	}
}

// --- parseThread ---

func TestParseThread(t *testing.T) {
	body := loadTestData(t, "thread.html")
	thread, err := parseThread(body, 47834565)
	if err != nil {
		t.Fatalf("parseThread: %v", err)
	}

	if thread.Story.ID == 0 {
		t.Error("thread story has ID 0")
	}
	if thread.Story.Title == "" {
		t.Error("thread story has empty title")
	}

	if len(thread.Comments) == 0 {
		t.Fatal("expected comments, got 0")
	}
	if len(thread.Comments) < 10 {
		t.Errorf("expected ≥10 comments, got %d", len(thread.Comments))
	}

	first := thread.Comments[0]
	if first.ID == 0 {
		t.Error("first comment has ID 0")
	}
	if first.Author == "" {
		t.Error("first comment has empty author")
	}
	if first.Text == "" {
		t.Error("first comment has empty text")
	}

	t.Logf("parsed thread: %d comments, first by %s (level %d)",
		len(thread.Comments), first.Author, first.Level)
}

func TestParseThread_CommentLevels(t *testing.T) {
	body := loadTestData(t, "thread.html")
	thread, err := parseThread(body, 47834565)
	if err != nil {
		t.Fatalf("parseThread: %v", err)
	}

	hasLevel0 := false
	hasLevel1 := false
	for _, c := range thread.Comments {
		if c.Level == 0 {
			hasLevel0 = true
		}
		if c.Level == 1 {
			hasLevel1 = true
		}
	}
	if !hasLevel0 {
		t.Error("no top-level (level 0) comments found")
	}
	if !hasLevel1 {
		t.Error("no reply (level 1) comments found")
	}
}

func TestParseThread_CommentVoteAuth(t *testing.T) {
	body := loadTestData(t, "thread.html")
	thread, err := parseThread(body, 47834565)
	if err != nil {
		t.Fatalf("parseThread: %v", err)
	}

	hasVoteAuth := false
	for _, c := range thread.Comments {
		if c.VoteUpAuth != "" {
			hasVoteAuth = true
			break
		}
	}
	if !hasVoteAuth {
		t.Error("no comment has VoteUpAuth — should have auth tokens on authenticated page")
	}
}

// --- parseCommentPage ---

func TestParseCommentPage(t *testing.T) {
	body := loadTestData(t, "thread.html")
	comments := parseCommentPage(body)

	if len(comments) == 0 {
		t.Fatal("expected comments, got 0")
	}

	for i, c := range comments[:min(5, len(comments))] {
		if c.ID == 0 {
			t.Errorf("comment %d: ID is 0", i)
		}
		if c.Author == "" {
			t.Errorf("comment %d (ID %d): empty author", i, c.ID)
		}
		if c.Text == "" {
			t.Errorf("comment %d (ID %d): empty text", i, c.ID)
		}
	}
}

// --- parseNewComments ---

func TestParseNewComments(t *testing.T) {
	body := loadTestData(t, "newcomments.html")
	comments := parseNewComments(body)

	if len(comments) == 0 {
		t.Fatal("expected comments, got 0")
	}
	if len(comments) < 20 {
		t.Errorf("expected ≥20 comments, got %d", len(comments))
	}

	first := comments[0]
	if first.ID == 0 {
		t.Error("first comment ID is 0")
	}
	if first.Author == "" {
		t.Error("first comment has empty author")
	}
	if first.Text == "" {
		t.Error("first comment has empty text")
	}

	t.Logf("parsed %d newcomments; first: [%d] by %s", len(comments), first.ID, first.Author)
}

func TestParseNewComments_ParentStory(t *testing.T) {
	body := loadTestData(t, "newcomments.html")
	comments := parseNewComments(body)

	hasParent := false
	for _, c := range comments {
		if c.ParentStoryID != 0 && c.ParentStoryTitle != "" {
			hasParent = true
			t.Logf("comment %d on story [%d] '%s'", c.ID, c.ParentStoryID, c.ParentStoryTitle)
			break
		}
	}
	if !hasParent {
		t.Error("no comment has ParentStoryID + ParentStoryTitle populated")
	}
}

func TestParseNewComments_VoteAuth(t *testing.T) {
	body := loadTestData(t, "newcomments.html")
	comments := parseNewComments(body)

	hasAuth := false
	for _, c := range comments {
		if c.VoteUpAuth != "" {
			hasAuth = true
			break
		}
	}
	if !hasAuth {
		t.Error("no newcomment has VoteUpAuth")
	}
}

// --- parseReplyForm ---

func TestParseReplyForm(t *testing.T) {
	body := loadTestData(t, "reply.html")
	form, err := parseReplyForm(body, 47836251)
	if err != nil {
		t.Fatalf("parseReplyForm: %v", err)
	}

	if form.ParentID != 47836251 {
		t.Errorf("ParentID = %d, want 47836251", form.ParentID)
	}
	if form.HMAC == "" {
		t.Error("HMAC is empty")
	}
	if form.Goto == "" {
		t.Error("Goto is empty")
	}

	t.Logf("reply form: parent=%d hmac=%s goto=%s", form.ParentID, form.HMAC[:8]+"…", form.Goto)
}

// --- parseSubmitForm ---

func TestParseSubmitForm(t *testing.T) {
	body := loadTestData(t, "submit.html")
	form, err := parseSubmitForm(body)
	if err != nil {
		t.Fatalf("parseSubmitForm: %v", err)
	}

	if form.FNID == "" {
		t.Error("FNID is empty")
	}
	if form.FNOP == "" {
		t.Error("FNOP is empty")
	}

	t.Logf("submit form: fnid=%s fnop=%s", form.FNID, form.FNOP)
}

// --- parseUserSettings ---

func TestParseUserSettings(t *testing.T) {
	body := loadTestData(t, "usersettings.html")
	settings, err := parseUserSettings(body, "teslashibe")
	if err != nil {
		t.Fatalf("parseUserSettings: %v", err)
	}

	if settings.ID != "teslashibe" {
		t.Errorf("ID = %q, want teslashibe", settings.ID)
	}
	if settings.HMAC == "" {
		t.Error("HMAC is empty")
	}
	if settings.Email == "" {
		t.Error("Email is empty")
	}

	t.Logf("settings: id=%s email=%s maxv=%s mina=%s delay=%s showDead=%v noPro=%v",
		settings.ID, settings.Email, settings.MaxVis, settings.MinAge,
		settings.Delay, settings.ShowDead, settings.NoPro)
}

func TestParseUserSettings_ShowDeadFalse(t *testing.T) {
	body := loadTestData(t, "usersettings.html")
	settings, err := parseUserSettings(body, "teslashibe")
	if err != nil {
		t.Fatalf("parseUserSettings: %v", err)
	}

	// The test account has ShowDead = false (selected option is "no").
	if settings.ShowDead {
		t.Error("ShowDead should be false for teslashibe (default is no)")
	}
}

// --- parseSelectBool ---

func TestParseSelectBool(t *testing.T) {
	tests := []struct {
		name string
		html string
		field string
		want bool
	}{
		{
			name:  "yes selected",
			html:  `<select name="showd"><option selected="t">yes</option><option>no</option></select>`,
			field: "showd",
			want:  true,
		},
		{
			name:  "no selected",
			html:  `<select name="showd"><option>yes</option><option selected="t">no</option></select>`,
			field: "showd",
			want:  false,
		},
		{
			name:  "field not present",
			html:  `<select name="other"><option selected="t">yes</option></select>`,
			field: "showd",
			want:  false,
		},
		{
			name:  "nopro yes",
			html:  `<select name="nopro"><option selected="t">yes</option><option>no</option></select>`,
			field: "nopro",
			want:  true,
		},
		{
			name:  "nopro no",
			html:  `<select name="nopro"><option>yes</option><option selected="t">no</option></select>`,
			field: "nopro",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseSelectBool(tt.html, tt.field)
			if got != tt.want {
				t.Errorf("parseSelectBool(%q) = %v, want %v", tt.field, got, tt.want)
			}
		})
	}
}

// --- storyType ---

func TestStoryType(t *testing.T) {
	tests := []struct {
		title string
		url   string
		want  string
	}{
		{"Ask HN: How do you do X?", "", "ask"},
		{"Show HN: My project", "https://example.com", "show"},
		{"Launch HN: Acme (YC W26)", "https://acme.com", "launch"},
		{"Regular article", "https://example.com", "story"},
		{"Text post with no URL", "", "ask"},
	}

	for _, tt := range tests {
		label := tt.title
		if len(label) > 20 {
			label = label[:20]
		}
		t.Run(tt.want+"/"+label, func(t *testing.T) {
			got := storyType(tt.title, tt.url)
			if got != tt.want {
				t.Errorf("storyType(%q, %q) = %q, want %q", tt.title, tt.url, got, tt.want)
			}
		})
	}
}

// --- isAuthenticated ---

func TestIsAuthenticated_AuthPage(t *testing.T) {
	body := loadTestData(t, "frontpage.html")
	if !isAuthenticated(body) {
		t.Error("expected authenticated page to return true")
	}
}

func TestIsAuthenticated_NoAuthPage(t *testing.T) {
	body := loadTestData(t, "frontpage_noauth.html")
	if isAuthenticated(body) {
		t.Error("expected unauthenticated page to return false")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
