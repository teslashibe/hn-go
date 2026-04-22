package mcp

import (
	"context"

	hn "github.com/teslashibe/hn-go"
	"github.com/teslashibe/mcptool"
)

// GetFeedPageInput is the typed input for hn_get_feed_page.
type GetFeedPageInput struct {
	Feed  string `json:"feed,omitempty" jsonschema:"description=feed name; allowed values: top,newest,best,ask,asknew,show,shownew,jobs,launches,front,default=top"`
	Page  int    `json:"page,omitempty" jsonschema:"description=1-based page number,minimum=1,default=1"`
	Day   string `json:"day,omitempty" jsonschema:"description=for feed=front only; date in YYYY-MM-DD format selecting that day's archived front page"`
	Limit int    `json:"limit,omitempty" jsonschema:"description=cap on returned stories,minimum=1,maximum=100,default=30"`
}

func feedFromName(name, day string) hn.Feed {
	switch name {
	case "", "top":
		return hn.FeedTop
	case "newest", "new":
		return hn.FeedNew
	case "best":
		return hn.FeedBest
	case "ask":
		return hn.FeedAsk
	case "asknew":
		return hn.FeedAskNew
	case "show":
		return hn.FeedShow
	case "shownew":
		return hn.FeedShowNew
	case "jobs":
		return hn.FeedJobs
	case "launches":
		return hn.FeedLaunches
	case "front":
		if day != "" {
			return hn.Feed("front?day=" + day)
		}
		return hn.FeedFront
	default:
		return hn.Feed(name)
	}
}

func getFeedPage(ctx context.Context, c *hn.Client, in GetFeedPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.Page(ctx, feedFromName(in.Feed, in.Day), page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

// GetThreadInput is the typed input for hn_get_thread.
type GetThreadInput struct {
	ItemID int `json:"item_id" jsonschema:"description=story or comment ID to fetch full thread for,required,minimum=1"`
}

func getThread(ctx context.Context, c *hn.Client, in GetThreadInput) (any, error) {
	return c.Thread(ctx, in.ItemID)
}

// UserPageInput is shared by tools paging over a user's content.
type UserPageInput struct {
	Username string `json:"username" jsonschema:"description=Hacker News username,required"`
	Page     int    `json:"page,omitempty" jsonschema:"description=1-based page number,minimum=1,default=1"`
	Limit    int    `json:"limit,omitempty" jsonschema:"description=cap on returned items,minimum=1,maximum=100,default=30"`
}

func getUserThreads(ctx context.Context, c *hn.Client, in UserPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	comments, err := c.UserThreads(ctx, in.Username, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(comments, "", defaultLimit(in.Limit, 30)), nil
}

func getUserFavorites(ctx context.Context, c *hn.Client, in UserPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.Favorites(ctx, in.Username, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

func getUserUpvoted(ctx context.Context, c *hn.Client, in UserPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.Upvoted(ctx, in.Username, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

// GetStoriesFromDomainInput is the typed input for hn_get_stories_from_domain.
type GetStoriesFromDomainInput struct {
	Domain string `json:"domain" jsonschema:"description=domain to filter by (e.g. github.com),required"`
	Page   int    `json:"page,omitempty" jsonschema:"description=1-based page number,minimum=1,default=1"`
	Limit  int    `json:"limit,omitempty" jsonschema:"description=cap on returned stories,minimum=1,maximum=100,default=30"`
}

func getStoriesFromDomain(ctx context.Context, c *hn.Client, in GetStoriesFromDomainInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.FromDomain(ctx, in.Domain, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

// PageOnlyInput is shared by tools that take a page number and limit.
type PageOnlyInput struct {
	Page  int `json:"page,omitempty" jsonschema:"description=1-based page number,minimum=1,default=1"`
	Limit int `json:"limit,omitempty" jsonschema:"description=cap on returned items,minimum=1,maximum=100,default=30"`
}

func getNewComments(ctx context.Context, c *hn.Client, in PageOnlyInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	comments, err := c.NewComments(ctx, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(comments, "", defaultLimit(in.Limit, 30)), nil
}

func getUserSubmissions(ctx context.Context, c *hn.Client, in UserPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.UserSubmissions(ctx, in.Username, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

func getUserFavoriteComments(ctx context.Context, c *hn.Client, in UserPageInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	comments, err := c.FavoriteComments(ctx, in.Username, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(comments, "", defaultLimit(in.Limit, 30)), nil
}

func getShowNew(ctx context.Context, c *hn.Client, in PageOnlyInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.ShowNew(ctx, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

func getAskNew(ctx context.Context, c *hn.Client, in PageOnlyInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.AskNew(ctx, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

func getLaunches(ctx context.Context, c *hn.Client, in PageOnlyInput) (any, error) {
	page := in.Page
	if page <= 0 {
		page = 1
	}
	stories, err := c.Launches(ctx, page)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(stories, "", defaultLimit(in.Limit, 30)), nil
}

// GetReplyFormInput is the typed input for hn_get_reply_form.
type GetReplyFormInput struct {
	CommentID int `json:"comment_id" jsonschema:"description=comment or story ID to reply to,required,minimum=1"`
}

func getReplyForm(ctx context.Context, c *hn.Client, in GetReplyFormInput) (any, error) {
	form, err := c.GetReplyForm(ctx, in.CommentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"parent_id": form.ParentID,
		"hmac":      form.HMAC,
		"goto":      form.Goto,
	}, nil
}

func getSubmitForm(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	form, err := c.GetSubmitForm(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"fnid": form.FNID,
		"fnop": form.FNOP,
	}, nil
}

// GetUserSettingsInput is the typed input for hn_get_user_settings.
type GetUserSettingsInput struct {
	Username string `json:"username" jsonschema:"description=Hacker News username (must match the authenticated session for editable fields),required"`
}

func getUserSettings(ctx context.Context, c *hn.Client, in GetUserSettingsInput) (any, error) {
	return c.GetUserSettings(ctx, in.Username)
}

var feedTools = []mcptool.Tool{
	mcptool.Define[*hn.Client, GetFeedPageInput](
		"hn_get_feed_page",
		"Fetch one page of an HN HTML feed (top/new/best/ask/show/jobs/launches/front) with per-item action tokens",
		"Page",
		getFeedPage,
	),
	mcptool.Define[*hn.Client, GetThreadInput](
		"hn_get_thread",
		"Fetch a story and its full flat comment tree by item ID",
		"Thread",
		getThread,
	),
	mcptool.Define[*hn.Client, UserPageInput](
		"hn_get_user_threads",
		"Fetch the comment threads page for a user (their public /threads page)",
		"UserThreads",
		getUserThreads,
	),
	mcptool.Define[*hn.Client, UserPageInput](
		"hn_get_user_favorites",
		"Fetch a user's public favourite stories list",
		"Favorites",
		getUserFavorites,
	),
	mcptool.Define[*hn.Client, UserPageInput](
		"hn_get_user_upvoted",
		"Fetch the upvoted stories list (only accessible for the cookie owner)",
		"Upvoted",
		getUserUpvoted,
	),
	mcptool.Define[*hn.Client, GetStoriesFromDomainInput](
		"hn_get_stories_from_domain",
		"Fetch stories submitted from a specific domain (e.g. github.com)",
		"FromDomain",
		getStoriesFromDomain,
	),
	mcptool.Define[*hn.Client, PageOnlyInput](
		"hn_get_new_comments",
		"Fetch the site-wide /newcomments feed; each entry includes parent story title and ID",
		"NewComments",
		getNewComments,
	),
	mcptool.Define[*hn.Client, UserPageInput](
		"hn_get_user_submissions",
		"Fetch the public list of stories submitted by a user",
		"UserSubmissions",
		getUserSubmissions,
	),
	mcptool.Define[*hn.Client, UserPageInput](
		"hn_get_user_favorite_comments",
		"Fetch the comments (not stories) a user has favourited",
		"FavoriteComments",
		getUserFavoriteComments,
	),
	mcptool.Define[*hn.Client, PageOnlyInput](
		"hn_get_show_new",
		"Fetch the newest Show HN posts — useful for early engagement before a post gains traction",
		"ShowNew",
		getShowNew,
	),
	mcptool.Define[*hn.Client, PageOnlyInput](
		"hn_get_ask_new",
		"Fetch the newest Ask HN posts — useful for responding early to gather or give feedback",
		"AskNew",
		getAskNew,
	),
	mcptool.Define[*hn.Client, PageOnlyInput](
		"hn_get_launches",
		"Fetch the YC Launch HN feed — high-signal feed for newly launched YC-backed products",
		"Launches",
		getLaunches,
	),
	mcptool.Define[*hn.Client, GetReplyFormInput](
		"hn_get_reply_form",
		"Fetch the HMAC nonce required to post a reply to a comment or story; pass result to hn_post_comment",
		"GetReplyForm",
		getReplyForm,
	),
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_get_submit_form",
		"Fetch the one-time fnid nonce required by hn_submit_story (or use hn_submit_show_hn / hn_submit_ask_hn)",
		"GetSubmitForm",
		getSubmitForm,
	),
	mcptool.Define[*hn.Client, GetUserSettingsInput](
		"hn_get_user_settings",
		"Fetch the editable profile settings for a user; required before calling hn_update_user_settings",
		"GetUserSettings",
		getUserSettings,
	),
}
