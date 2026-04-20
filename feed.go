package hn

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// Page fetches one page of a named HN feed.
//
// page is 1-based. Use FeedTop / FeedNew / … constants for feed.
// For FeedFront pass the day as a query string yourself: Feed("front?day=2026-04-20").
func (c *Client) Page(ctx context.Context, feed Feed, page int) ([]Story, error) {
	path := "/"
	if feed != FeedTop {
		path = "/" + string(feed)
	}
	if page > 1 {
		if strings.Contains(path, "?") {
			path += fmt.Sprintf("&p=%d", page)
		} else {
			path += fmt.Sprintf("?p=%d", page)
		}
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseStories(body)
}

// Thread fetches a story and its full (flat) comment list.
func (c *Client) Thread(ctx context.Context, itemID int) (*Thread, error) {
	body, err := c.hnGet(ctx, fmt.Sprintf("/item?id=%d", itemID))
	if err != nil {
		return nil, err
	}
	return parseThread(body, itemID)
}

// UserThreads fetches the comment threads for username (their public "threads" page).
func (c *Client) UserThreads(ctx context.Context, username string, page int) ([]Comment, error) {
	path := fmt.Sprintf("/threads?id=%s", username)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseCommentPage(body), nil
}

// Favorites fetches the public favourites list for username.
func (c *Client) Favorites(ctx context.Context, username string, page int) ([]Story, error) {
	path := fmt.Sprintf("/favorites?id=%s", username)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseStories(body)
}

// Upvoted fetches the upvoted list for the authenticated user.
// This endpoint is only accessible for the cookie owner.
func (c *Client) Upvoted(ctx context.Context, username string, page int) ([]Story, error) {
	path := fmt.Sprintf("/upvoted?id=%s", username)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseStories(body)
}

// FromDomain fetches stories submitted from a specific domain.
func (c *Client) FromDomain(ctx context.Context, domain string, page int) ([]Story, error) {
	path := fmt.Sprintf("/from?site=%s", domain)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseStories(body)
}

// NewComments fetches the site-wide recent-comments feed (/newcomments).
// Each returned Comment includes the parent story title/ID via ParentStoryID / ParentStoryTitle.
func (c *Client) NewComments(ctx context.Context, page int) ([]Comment, error) {
	path := "/newcomments"
	if page > 1 {
		path += fmt.Sprintf("?p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseNewComments(body), nil
}

// UserSubmissions fetches the public list of stories submitted by username.
func (c *Client) UserSubmissions(ctx context.Context, username string, page int) ([]Story, error) {
	path := fmt.Sprintf("/submitted?id=%s", username)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseStories(body)
}

// FavoriteComments fetches the comments (not stories) that username has favourited.
// This is the ?comments=t variant of /favorites.
func (c *Client) FavoriteComments(ctx context.Context, username string, page int) ([]Comment, error) {
	path := fmt.Sprintf("/favorites?id=%s&comments=t", username)
	if page > 1 {
		path += fmt.Sprintf("&p=%d", page)
	}
	body, err := c.hnGet(ctx, path)
	if err != nil {
		return nil, err
	}
	return parseCommentPage(body), nil
}

// ShowNew fetches the newest Show HN posts — engage early before a post gains traction.
func (c *Client) ShowNew(ctx context.Context, page int) ([]Story, error) {
	return c.Page(ctx, FeedShowNew, page)
}

// AskNew fetches the newest Ask HN posts — respond early to gather/give feedback.
func (c *Client) AskNew(ctx context.Context, page int) ([]Story, error) {
	return c.Page(ctx, FeedAskNew, page)
}

// Launches fetches the YC Launch HN feed (/launches).
// This is the highest-signal feed for finding newly launched YC-backed products.
func (c *Client) Launches(ctx context.Context, page int) ([]Story, error) {
	return c.Page(ctx, FeedLaunches, page)
}

// GetReplyForm fetches the HMAC + parent data needed to post a reply to commentID.
func (c *Client) GetReplyForm(ctx context.Context, commentID int) (*ReplyForm, error) {
	body, err := c.hnGet(ctx, fmt.Sprintf("/reply?id=%d", commentID))
	if err != nil {
		return nil, err
	}
	return parseReplyForm(body, commentID)
}

// GetSubmitForm fetches the one-time FNID nonce from /submit.
func (c *Client) GetSubmitForm(ctx context.Context) (*SubmitForm, error) {
	body, err := c.hnGet(ctx, "/submit")
	if err != nil {
		return nil, err
	}
	return parseSubmitForm(body)
}

// GetUserSettings fetches the editable profile settings for username.
func (c *Client) GetUserSettings(ctx context.Context, username string) (*UserSettings, error) {
	body, err := c.hnGet(ctx, fmt.Sprintf("/user?id=%s", username))
	if err != nil {
		return nil, err
	}
	return parseUserSettings(body, username)
}

// ---------- HTML parsers ----------

var (
	reItemID    = regexp.MustCompile(`(?:class="athing|class="athing submission") id="(\d+)"`)
	reRank      = regexp.MustCompile(`<span class="rank">(\d+)\.</span>`)
	reTitleLink = regexp.MustCompile(`class="titleline"><a href="([^"]+)"[^>]*>([^<]+)`)
	reDomain    = regexp.MustCompile(`class="sitestr">([^<]+)<`)
	rePoints    = regexp.MustCompile(`id="score_\d+">(\d+) points?`)
	reAuthor    = regexp.MustCompile(`class="hnuser">([^<]+)<`)
	reAge       = regexp.MustCompile(`title="([^"]+)"\s*><a href="item\?id=\d+">([^<]+)</a></span>`)
	reComments  = regexp.MustCompile(`href="item\?id=\d+"[^>]*>(\d+)&nbsp;comment`)
	reVoteUp    = regexp.MustCompile(`vote\?id=(\d+)&amp;how=up&amp;auth=([a-f0-9]+)`)
	reHide      = regexp.MustCompile(`hide\?id=(\d+)&amp;auth=([a-f0-9]+)`)
	reFave      = regexp.MustCompile(`fave\?id=(\d+)&amp;auth=([a-f0-9]+)`)
	reUnvote    = regexp.MustCompile(`vote\?id=(\d+)&amp;how=un&amp;auth=([a-f0-9]+)`)
	reAge2      = regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})`)

	reCommentLevel  = regexp.MustCompile(`indent.*?width="(\d+)"`)
	reCommentAuthor = regexp.MustCompile(`class="hnuser">([^<]+)<`)
	reCommentAge    = regexp.MustCompile(`class="age"[^>]*title="([^"]+)"`)
	reCommentText   = regexp.MustCompile(`(?s)class="commtext[^"]*"[^>]*>(.*?)</span>`)
	reCommentID     = regexp.MustCompile(`id="(\d+)"`)
	reReplyGoto     = regexp.MustCompile(`reply\?id=(\d+)&amp;goto=([^"]+)`)

	reInputVal  = regexp.MustCompile(`name="([^"]+)"[^>]+value="([^"]*)"`)
	reFNID      = regexp.MustCompile(`name="fnid"\s+value="([^"]+)"`)
	reFNOP      = regexp.MustCompile(`name="fnop"\s+value="([^"]+)"`)
	reHMAC      = regexp.MustCompile(`name="hmac"\s+value="([^"]+)"`)
	reParent    = regexp.MustCompile(`name="parent"\s+value="([^"]+)"`)
	reGoto      = regexp.MustCompile(`name="goto"\s+value="([^"]+)"`)

	// newcomments-specific
	reOnStory = regexp.MustCompile(`on: <a href="item\?id=(\d+)"[^>]*>([^<]+)<`)
	reContext  = regexp.MustCompile(`context.*?item\?id=(\d+)#(\d+)`)
)

func parseStories(body []byte) ([]Story, error) {
	// Validate that the body is parseable HTML.
	if _, err := html.Parse(bytes.NewReader(body)); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailure, err)
	}

	// HN's table layout is easier to parse with targeted regex than with a full
	// DOM walk, so we operate on the raw string after validating the HTML.
	raw := string(body)

	// Extract all per-item auth maps first (keyed by item ID).
	voteUpMap := extractIDAuthMap(reVoteUp, raw)
	unvoteMap := extractIDAuthMap(reUnvote, raw)
	hideMap := extractIDAuthMap(reHide, raw)
	faveMap := extractIDAuthMap(reFave, raw)

	// HN renders each story as two consecutive <tr> rows. We walk the raw HTML
	// and slice by item ID anchor.
	var stories []Story
	chunks := splitByAthing(raw)

	for rank, chunk := range chunks {
		idMatch := reItemID.FindStringSubmatch(chunk)
		if len(idMatch) < 2 {
			continue
		}
		id, _ := strconv.Atoi(idMatch[1])
		if id == 0 {
			continue
		}

		s := Story{
			ID:         id,
			Rank:       rank + 1,
			VoteUpAuth: voteUpMap[id],
			HideAuth:   hideMap[id],
			FaveAuth:   faveMap[id],
			Voted:      unvoteMap[id] != "",
		}

		if m := reTitleLink.FindStringSubmatch(chunk); len(m) >= 3 {
			s.URL = m[1]
			s.Title = html.UnescapeString(m[2])
		}
		if m := reDomain.FindStringSubmatch(chunk); len(m) >= 2 {
			s.Domain = m[1]
		}
		if m := rePoints.FindStringSubmatch(chunk); len(m) >= 2 {
			s.Points, _ = strconv.Atoi(m[1])
		}
		if m := reAuthor.FindStringSubmatch(chunk); len(m) >= 2 {
			s.Author = m[1]
		}
		if m := reAge.FindStringSubmatch(chunk); len(m) >= 3 {
			s.TimeAgo = m[2]
			if ts := reAge2.FindString(m[1]); ts != "" {
				if t, err := time.Parse("2006-01-02T15:04:05", ts); err == nil {
					s.Age = t
				}
			}
		}
		if m := reComments.FindStringSubmatch(chunk); len(m) >= 2 {
			s.Comments, _ = strconv.Atoi(m[1])
		}

		// Determine type from URL / title prefix.
		s.Type = storyType(s.Title, s.URL)

		stories = append(stories, s)
	}
	return stories, nil
}

func parseThread(body []byte, itemID int) (*Thread, error) {
	stories, err := parseStories(body)
	if err != nil {
		return nil, err
	}
	var story Story
	for _, s := range stories {
		if s.ID == itemID {
			story = s
			break
		}
	}
	if story.ID == 0 && len(stories) > 0 {
		story = stories[0]
	}

	comments := parseCommentPage(body)
	return &Thread{Story: story, Comments: comments}, nil
}

func parseCommentPage(body []byte) []Comment {
	raw := string(body)
	var comments []Comment

	// Each comment is anchored by its id attribute on the <tr class="athing comtr"> row.
	parts := strings.Split(raw, `class="athing comtr"`)
	for _, part := range parts[1:] {
		var c Comment

		if m := reCommentID.FindStringSubmatch(part); len(m) >= 2 {
			c.ID, _ = strconv.Atoi(m[1])
		}
		if m := reCommentLevel.FindStringSubmatch(part); len(m) >= 2 {
			indent, _ := strconv.Atoi(m[1])
			c.Level = indent / 40
		}
		if m := reCommentAuthor.FindStringSubmatch(part); len(m) >= 2 {
			c.Author = m[1]
		}
		if m := reCommentAge.FindStringSubmatch(part); len(m) >= 2 {
			if ts := reAge2.FindString(m[1]); ts != "" {
				if t, err := time.Parse("2006-01-02T15:04:05", ts); err == nil {
					c.Age = t
					c.TimeAgo = time.Since(t).Round(time.Minute).String()
				}
			}
		}
		if m := reCommentText.FindStringSubmatch(part); len(m) >= 2 {
			c.Text = strings.TrimSpace(m[1])
		}
		if m := reReplyGoto.FindStringSubmatch(part); len(m) >= 3 {
			c.ReplyGoto = m[2]
		}
		if m := reVoteUp.FindStringSubmatch(part); len(m) >= 3 {
			c.VoteUpAuth = m[2]
		}
		if c.ID > 0 {
			comments = append(comments, c)
		}
	}
	return comments
}

// parseNewComments parses the /newcomments feed.
// Each entry is a flat comment block (class="athing") with an "on: <story>" nav link.
func parseNewComments(body []byte) []Comment {
	raw := string(body)
	var comments []Comment

	// Split on the athing anchor — newcomments uses plain "athing", not "athing comtr".
	parts := strings.Split(raw, `<tr class="athing"`)
	for _, part := range parts[1:] {
		var c Comment

		if m := reCommentID.FindStringSubmatch(part); len(m) >= 2 {
			c.ID, _ = strconv.Atoi(m[1])
		}
		if m := reCommentAuthor.FindStringSubmatch(part); len(m) >= 2 {
			c.Author = m[1]
		}
		if m := reCommentAge.FindStringSubmatch(part); len(m) >= 2 {
			if ts := reAge2.FindString(m[1]); ts != "" {
				if t, err := time.Parse("2006-01-02T15:04:05", ts); err == nil {
					c.Age = t
					c.TimeAgo = time.Since(t).Round(time.Minute).String()
				}
			}
		}
		if m := reCommentText.FindStringSubmatch(part); len(m) >= 2 {
			c.Text = strings.TrimSpace(m[1])
		}
		if m := reVoteUp.FindStringSubmatch(part); len(m) >= 3 {
			c.VoteUpAuth = m[2]
		}
		if m := reOnStory.FindStringSubmatch(part); len(m) >= 3 {
			c.ParentStoryID, _ = strconv.Atoi(m[1])
			c.ParentStoryTitle = html.UnescapeString(m[2])
		}
		if m := reContext.FindStringSubmatch(part); len(m) >= 3 {
			c.ReplyGoto = fmt.Sprintf("item?id=%s#%s", m[1], m[2])
		}

		if c.ID > 0 {
			comments = append(comments, c)
		}
	}
	return comments
}

func parseReplyForm(body []byte, parentID int) (*ReplyForm, error) {
	raw := string(body)
	f := &ReplyForm{ParentID: parentID}

	if m := reHMAC.FindStringSubmatch(raw); len(m) >= 2 {
		f.HMAC = m[1]
	}
	if m := reGoto.FindStringSubmatch(raw); len(m) >= 2 {
		f.Goto = m[1]
	}
	if f.HMAC == "" {
		return nil, fmt.Errorf("%w: hmac not found in reply form", ErrParseFailure)
	}
	return f, nil
}

func parseSubmitForm(body []byte) (*SubmitForm, error) {
	raw := string(body)
	f := &SubmitForm{}

	if m := reFNID.FindStringSubmatch(raw); len(m) >= 2 {
		f.FNID = m[1]
	}
	if m := reFNOP.FindStringSubmatch(raw); len(m) >= 2 {
		f.FNOP = m[1]
	}
	if f.FNID == "" {
		return nil, fmt.Errorf("%w: fnid not found (not logged in?)", ErrParseFailure)
	}
	return f, nil
}

func parseUserSettings(body []byte, username string) (*UserSettings, error) {
	raw := string(body)
	s := &UserSettings{ID: username}

	fields := map[string]*string{
		"hmac":  &s.HMAC,
		"email": &s.Email,
		"maxv":  &s.MaxVis,
		"mina":  &s.MinAge,
		"delay": &s.Delay,
	}
	for _, m := range reInputVal.FindAllStringSubmatch(raw, -1) {
		if len(m) < 3 {
			continue
		}
		if dst, ok := fields[m[1]]; ok {
			*dst = m[2]
		}
	}
	s.ShowDead = strings.Contains(raw, `name="showd"`) &&
		strings.Contains(raw, `<option selected`)
	s.NoPro = strings.Contains(raw, `name="nopro"`) &&
		strings.Contains(raw, `<option selected`)

	if s.HMAC == "" {
		return nil, fmt.Errorf("%w: hmac not found in user settings", ErrParseFailure)
	}
	return s, nil
}

// ---------- helpers ----------

// splitByAthing breaks the raw HTML into per-story chunks.
func splitByAthing(raw string) []string {
	const marker = `class="athing`
	parts := strings.Split(raw, marker)
	chunks := make([]string, 0, len(parts)-1)
	for _, p := range parts[1:] {
		chunks = append(chunks, marker+p)
	}
	return chunks
}

// extractIDAuthMap builds a map[itemID]authToken from a regex with two capture groups.
func extractIDAuthMap(re *regexp.Regexp, raw string) map[int]string {
	m := make(map[int]string)
	for _, match := range re.FindAllStringSubmatch(raw, -1) {
		if len(match) < 3 {
			continue
		}
		id, _ := strconv.Atoi(match[1])
		if id > 0 {
			m[id] = match[2]
		}
	}
	return m
}

func storyType(title, url string) string {
	tl := strings.ToLower(title)
	switch {
	case strings.HasPrefix(tl, "ask hn:"):
		return "ask"
	case strings.HasPrefix(tl, "show hn:"):
		return "show"
	case url == "":
		return "ask"
	default:
		return "story"
	}
}
