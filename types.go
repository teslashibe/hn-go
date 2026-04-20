package hn

import "time"

// Feed identifies which HN list page to read.
type Feed string

const (
	FeedTop      Feed = ""         // https://news.ycombinator.com/ (default front page)
	FeedNew      Feed = "newest"   // newest submissions across all types
	FeedBest     Feed = "best"     // best stories
	FeedAsk      Feed = "ask"      // Ask HN (best)
	FeedAskNew   Feed = "asknew"   // Ask HN (newest first) — great for early engagement
	FeedShow     Feed = "show"     // Show HN (best)
	FeedShowNew  Feed = "shownew"  // Show HN (newest first) — great for early engagement
	FeedJobs     Feed = "jobs"     // HN job listings
	FeedLaunches Feed = "launches" // YC Launch HN posts
	FeedFront    Feed = "front"    // past front pages (supports ?day=YYYY-MM-DD)
)

// FirebaseUpdates is returned by the /v0/updates.json endpoint.
// Items is a list of recently changed item IDs; Profiles is a list of recently
// changed usernames. Poll this every 30–60 s for near-realtime monitoring.
type FirebaseUpdates struct {
	Items    []int    `json:"items"`
	Profiles []string `json:"profiles"`
}

// Story is a single row scraped from an HN list page.
type Story struct {
	ID      int       `json:"id"`
	Rank    int       `json:"rank"`
	Title   string    `json:"title"`
	URL     string    `json:"url"`    // empty for text / Ask HN posts
	Domain  string    `json:"domain"` // e.g. "github.com"
	Author  string    `json:"author"`
	Points  int       `json:"points"`
	Comments int      `json:"comments"`
	TimeAgo string    `json:"time_ago"`
	Age     time.Time `json:"age"`
	Type    string    `json:"type"` // "story" | "ask" | "show" | "job"

	// Per-item HMAC tokens extracted from the page. These are session-scoped
	// and required for action calls. They are empty for already-voted items.
	VoteUpAuth string `json:"vote_up_auth,omitempty"`
	HideAuth   string `json:"hide_auth,omitempty"`
	FaveAuth   string `json:"fave_auth,omitempty"`
	Voted      bool   `json:"voted"` // true when the session has already upvoted
}

// Comment is a single HN comment node; children are populated when
// fetching a full thread via Thread().
type Comment struct {
	ID      int       `json:"id"`
	Author  string    `json:"author"`
	Text    string    `json:"text"` // inner HTML of the comment body
	TimeAgo string    `json:"time_ago"`
	Age     time.Time `json:"age"`
	Level   int       `json:"level"` // indentation depth, 0 = top-level

	// Action tokens
	VoteUpAuth string    `json:"vote_up_auth,omitempty"`
	ReplyGoto  string    `json:"reply_goto,omitempty"`
	Children   []Comment `json:"children,omitempty"`

	// Populated only when fetched from /newcomments — the story this comment belongs to.
	ParentStoryID    int    `json:"parent_story_id,omitempty"`
	ParentStoryTitle string `json:"parent_story_title,omitempty"`
}

// Thread is a story with its full comment tree.
type Thread struct {
	Story    Story     `json:"story"`
	Comments []Comment `json:"comments"`
}

// FirebaseItem mirrors the HN Firebase v0 item schema.
type FirebaseItem struct {
	ID          int    `json:"id"`
	By          string `json:"by"`
	Title       string `json:"title"`
	URL         string `json:"url"`
	Text        string `json:"text"`
	Score       int    `json:"score"`
	Descendants int    `json:"descendants"`
	Kids        []int  `json:"kids"`
	Time        int64  `json:"time"`
	Type        string `json:"type"` // "story" | "comment" | "ask" | "job" | "poll" | "pollopt"
	Dead        bool   `json:"dead"`
	Deleted     bool   `json:"deleted"`
	Parent      int    `json:"parent"`
	Poll        int    `json:"poll"`
	Parts       []int  `json:"parts"`
}

// FirebaseUser mirrors the HN Firebase v0 user schema.
type FirebaseUser struct {
	ID        string `json:"id"`
	Karma     int    `json:"karma"`
	Created   int64  `json:"created"`
	About     string `json:"about"`
	Submitted []int  `json:"submitted"`
}

// SearchHit is one result from the Algolia HN search API.
type SearchHit struct {
	ObjectID    string   `json:"objectID"`
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Author      string   `json:"author"`
	Points      int      `json:"points"`
	NumComments int      `json:"num_comments"`
	CreatedAt   string   `json:"created_at"`
	CreatedAtI  int64    `json:"created_at_i"`
	StoryID     int      `json:"story_id"`
	ParentID    int      `json:"parent_id"`
	StoryText   string   `json:"story_text,omitempty"`
	Tags        []string `json:"_tags"`
}

// SearchResult is the top-level Algolia response envelope.
type SearchResult struct {
	Hits        []SearchHit `json:"hits"`
	NbHits      int         `json:"nbHits"`
	Page        int         `json:"page"`
	NbPages     int         `json:"nbPages"`
	HitsPerPage int         `json:"hitsPerPage"`
	Query       string      `json:"query"`
}

// SearchTag is an Algolia tag filter. Multiple tags are AND-ed.
type SearchTag string

const (
	TagStory     SearchTag = "story"
	TagComment   SearchTag = "comment"
	TagAskHN     SearchTag = "ask_hn"
	TagShowHN    SearchTag = "show_hn"
	TagJob       SearchTag = "job"
	TagPoll      SearchTag = "poll"
	TagFrontPage SearchTag = "front_page"
)

// SearchOptions configures an Algolia search request.
type SearchOptions struct {
	// Tags restricts results to specific content types (default: any).
	// Multiple tags are AND-ed. For OR, use TagsOr instead.
	Tags []SearchTag
	// TagsOr groups tags with OR logic — Algolia syntax: (tag1,tag2).
	// This is combined with Tags via AND: tags=Tags,(TagsOr)
	TagsOr []SearchTag
	// Page is zero-based.
	Page int
	// HitsPerPage defaults to 20; max 100.
	HitsPerPage int
	// MinPoints filters results with points >= MinPoints (0 = no filter).
	MinPoints int
	// MinComments filters results with num_comments >= MinComments (0 = no filter).
	MinComments int
}

// SubmitForm holds the one-time FNID nonce fetched from /submit.
// It is consumed by Submit() automatically; you rarely need this directly.
type SubmitForm struct {
	FNID string
	FNOP string
}

// ReplyForm holds the per-comment HMAC required to post a reply.
// Fetch it via ReplyForm(); pass it to Comment().
type ReplyForm struct {
	ParentID int
	HMAC     string
	Goto     string
}

// UserSettings represents the editable profile fields from /user?id=<name>.
type UserSettings struct {
	ID      string
	HMAC    string
	Email   string
	MaxVis  string // maxv – min visibility threshold
	MinAge  string // mina – minimum age in minutes
	Delay   string // comment delay in minutes
	ShowDead bool
	NoPro    bool
}
