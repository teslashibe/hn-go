package hn

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// --- Authenticated write actions ---
// Every action requires a per-item auth token embedded in the HTML page.
// These tokens are HMAC-SHA1 digests scoped to the session + item.
//
// Workflow:
//   1. Fetch the relevant page (via Page / Thread / GetReplyForm / GetSubmitForm).
//   2. Extract the auth token from the returned Story / Comment / form struct.
//   3. Call the action method.

// Upvote upvotes item itemID using the auth token from Story.VoteUpAuth.
// Returns ErrForbidden if the token is expired or the item was already voted.
func (c *Client) Upvote(ctx context.Context, itemID int, auth string) error {
	return c.voteAction(ctx, itemID, "up", auth)
}

// Unvote removes an existing upvote using Story.VoteUpAuth.
func (c *Client) Unvote(ctx context.Context, itemID int, auth string) error {
	return c.voteAction(ctx, itemID, "un", auth)
}

func (c *Client) voteAction(ctx context.Context, itemID int, how, auth string) error {
	if auth == "" {
		return fmt.Errorf("%w: vote auth token is empty (item already voted or not found on page)", ErrBadRequest)
	}
	path := fmt.Sprintf("/vote?id=%d&how=%s&auth=%s&goto=news", itemID, how, auth)
	status, _, err := c.hnGetNoFollow(ctx, path)
	if err != nil {
		return err
	}
	if status != http.StatusFound && status != http.StatusMovedPermanently {
		return fmt.Errorf("hn: vote: unexpected status %d", status)
	}
	return nil
}

// Hide hides itemID from the feed using Story.HideAuth.
func (c *Client) Hide(ctx context.Context, itemID int, auth string) error {
	if auth == "" {
		return fmt.Errorf("%w: hide auth token is empty", ErrBadRequest)
	}
	path := fmt.Sprintf("/hide?id=%d&auth=%s&goto=news", itemID, auth)
	status, _, err := c.hnGetNoFollow(ctx, path)
	if err != nil {
		return err
	}
	if status != http.StatusFound && status != http.StatusMovedPermanently {
		return fmt.Errorf("hn: hide: unexpected status %d", status)
	}
	return nil
}

// Fave adds itemID to the authenticated user's favourites using Story.FaveAuth.
func (c *Client) Fave(ctx context.Context, itemID int, auth string) error {
	if auth == "" {
		return fmt.Errorf("%w: fave auth token is empty", ErrBadRequest)
	}
	path := fmt.Sprintf("/fave?id=%d&auth=%s", itemID, auth)
	status, _, err := c.hnGetNoFollow(ctx, path)
	if err != nil {
		return err
	}
	if status != http.StatusFound && status != http.StatusMovedPermanently {
		return fmt.Errorf("hn: fave: unexpected status %d", status)
	}
	return nil
}

// Flag flags itemID as problematic. Requires ≥30 karma; returns ErrForbidden otherwise.
func (c *Client) Flag(ctx context.Context, itemID int, auth string) error {
	if auth == "" {
		return fmt.Errorf("%w: flag auth token is empty (need ≥30 karma)", ErrBadRequest)
	}
	path := fmt.Sprintf("/flag?id=%d&auth=%s", itemID, auth)
	status, _, err := c.hnGetNoFollow(ctx, path)
	if err != nil {
		return err
	}
	if status == http.StatusForbidden {
		return ErrForbidden
	}
	if status != http.StatusFound && status != http.StatusMovedPermanently {
		return fmt.Errorf("hn: flag: unexpected status %d", status)
	}
	return nil
}

// Comment posts a reply to parentID.
//
// Use GetReplyForm(ctx, parentID) to obtain the ReplyForm first.
func (c *Client) Comment(ctx context.Context, form *ReplyForm, text string) error {
	if form == nil {
		return fmt.Errorf("%w: reply form is nil — call GetReplyForm first", ErrBadRequest)
	}
	if form.HMAC == "" {
		return fmt.Errorf("%w: reply form has no hmac", ErrBadRequest)
	}
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("%w: comment text is empty", ErrBadRequest)
	}

	body, err := c.hnPost(ctx, "/comment", url.Values{
		"parent": {fmt.Sprintf("%d", form.ParentID)},
		"goto":   {form.Goto},
		"hmac":   {form.HMAC},
		"text":   {text},
	})
	if err != nil {
		return err
	}
	// HN returns the thread page on success, or an error page with "login" in URL.
	if strings.Contains(string(body), "login?") {
		return ErrUnauthenticated
	}
	return nil
}

// Submit submits a new story or Ask HN post.
//
// Use GetSubmitForm(ctx) to obtain the SubmitForm first.
// For a link post: set title + storyURL, leave text empty.
// For an Ask HN post: set title + text, leave storyURL empty.
func (c *Client) Submit(ctx context.Context, form *SubmitForm, title, storyURL, text string) error {
	if form == nil {
		return fmt.Errorf("%w: submit form is nil — call GetSubmitForm first", ErrBadRequest)
	}
	if form.FNID == "" {
		return fmt.Errorf("%w: submit form has no fnid (not logged in?)", ErrBadRequest)
	}
	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("%w: submit title is required", ErrBadRequest)
	}
	if storyURL != "" && text != "" {
		return fmt.Errorf("%w: submit — provide either url or text, not both", ErrBadRequest)
	}

	vals := url.Values{
		"fnid": {form.FNID},
		"fnop": {form.FNOP},
		"title": {title},
		"url":   {storyURL},
		"text":  {text},
	}
	body, err := c.hnPost(ctx, "/r", vals)
	if err != nil {
		return err
	}
	if strings.Contains(string(body), "login?") {
		return ErrUnauthenticated
	}
	// Re-using an expired fnid returns the submit form again.
	if strings.Contains(string(body), `name="fnid"`) {
		return fmt.Errorf("hn: submit: fnid expired — call GetSubmitForm again")
	}
	return nil
}

// SubmitShowHN submits a "Show HN:" post. title should NOT include the prefix —
// it is added automatically. storyURL is the link being shown.
// For a text-only Show HN, leave storyURL empty and pass text instead.
func (c *Client) SubmitShowHN(ctx context.Context, title, storyURL, text string) error {
	form, err := c.GetSubmitForm(ctx)
	if err != nil {
		return err
	}
	return c.Submit(ctx, form, "Show HN: "+title, storyURL, text)
}

// SubmitAskHN submits an "Ask HN:" post. title should NOT include the prefix —
// it is added automatically. text is the body of the question.
func (c *Client) SubmitAskHN(ctx context.Context, title, text string) error {
	form, err := c.GetSubmitForm(ctx)
	if err != nil {
		return err
	}
	return c.Submit(ctx, form, "Ask HN: "+title, "", text)
}

// UpdateUserSettings persists updated profile settings via /user.
// Fetch current values with GetUserSettings first to populate HMAC + ID.
func (c *Client) UpdateUserSettings(ctx context.Context, s *UserSettings) error {
	if s == nil {
		return fmt.Errorf("%w: settings is nil — call GetUserSettings first", ErrBadRequest)
	}
	if s.HMAC == "" || s.ID == "" {
		return fmt.Errorf("%w: settings id and hmac are required", ErrBadRequest)
	}
	showDead := "no"
	if s.ShowDead {
		showDead = "yes"
	}
	noPro := "no"
	if s.NoPro {
		noPro = "yes"
	}
	_, err := c.hnPost(ctx, "/user", url.Values{
		"id":     {s.ID},
		"hmac":   {s.HMAC},
		"email":  {s.Email},
		"maxv":   {s.MaxVis},
		"mina":   {s.MinAge},
		"delay":  {s.Delay},
		"showd":  {showDead},
		"nopro":  {noPro},
	})
	return err
}
