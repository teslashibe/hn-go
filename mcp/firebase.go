package mcp

import (
	"context"

	hn "github.com/teslashibe/hn-go"
	"github.com/teslashibe/mcptool"
)

// EmptyInput is used by tools that take no parameters.
type EmptyInput struct{}

// LimitInput caps a returned slice of IDs (does not change what the upstream
// API returns; just trims the response sent to the agent).
type LimitInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"description=cap on returned items,minimum=1,maximum=500,default=100"`
}

func defaultLimit(in, def int) int {
	if in <= 0 {
		return def
	}
	return in
}

func getTopStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.TopStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getNewStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.NewStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getBestStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.BestStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getAskStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.AskStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getShowStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.ShowStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getJobStoryIDs(ctx context.Context, c *hn.Client, in LimitInput) (any, error) {
	ids, err := c.JobStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	return mcptool.PageOf(ids, "", defaultLimit(in.Limit, 100)), nil
}

func getMaxItemID(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	id, err := c.MaxItemID(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"max_item_id": id}, nil
}

// GetItemInput is the typed input for hn_get_item.
type GetItemInput struct {
	ID int `json:"id" jsonschema:"description=Hacker News item ID (story, comment, ask, show, job, poll, or pollopt),required,minimum=1"`
}

func getItem(ctx context.Context, c *hn.Client, in GetItemInput) (any, error) {
	return c.Item(ctx, in.ID)
}

// GetUserInput is the typed input for hn_get_user.
type GetUserInput struct {
	Username string `json:"username" jsonschema:"description=Hacker News username (case-sensitive),required"`
}

func getUser(ctx context.Context, c *hn.Client, in GetUserInput) (any, error) {
	return c.FirebaseUser(ctx, in.Username)
}

// GetItemsBatchInput is the typed input for hn_get_items_batch.
type GetItemsBatchInput struct {
	IDs         []int `json:"ids" jsonschema:"description=item IDs to fetch concurrently; dead/deleted items are omitted,required"`
	Concurrency int   `json:"concurrency,omitempty" jsonschema:"description=parallel fetch goroutines,minimum=1,maximum=50,default=10"`
	Limit       int   `json:"limit,omitempty" jsonschema:"description=cap on returned items (post-fetch),minimum=1,maximum=500,default=100"`
}

func getItemsBatch(ctx context.Context, c *hn.Client, in GetItemsBatchInput) (any, error) {
	items := c.ItemsBatch(ctx, in.IDs, in.Concurrency)
	return mcptool.PageOf(items, "", defaultLimit(in.Limit, 100)), nil
}

// GetTopStoriesExpandedInput is the typed input for hn_get_top_stories_expanded.
type GetTopStoriesExpandedInput struct {
	Limit       int `json:"limit,omitempty" jsonschema:"description=number of top stories to hydrate (0 = all 500),minimum=0,maximum=500,default=30"`
	Concurrency int `json:"concurrency,omitempty" jsonschema:"description=parallel fetch goroutines,minimum=1,maximum=50,default=10"`
}

func getTopStoriesExpanded(ctx context.Context, c *hn.Client, in GetTopStoriesExpandedInput) (any, error) {
	limit := in.Limit
	if limit < 0 {
		limit = 0
	}
	items, err := c.TopStoriesExpanded(ctx, limit, in.Concurrency)
	if err != nil {
		return nil, err
	}
	maxItems := limit
	if maxItems <= 0 {
		maxItems = 500
	}
	return mcptool.PageOf(items, "", maxItems), nil
}

func getUpdates(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	return c.Updates(ctx)
}

func getWhoIsHiring(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	return c.WhoIsHiring(ctx)
}

func getWhoWantsToBeHired(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	return c.WhoWantsToBeHired(ctx)
}

var firebaseTools = []mcptool.Tool{
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_top_story_ids",
		"List current top-story item IDs from the public Firebase API (up to 500)",
		"TopStoryIDs",
		getTopStoryIDs,
	),
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_new_story_ids",
		"List newest story item IDs from the public Firebase API (up to 500)",
		"NewStoryIDs",
		getNewStoryIDs,
	),
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_best_story_ids",
		"List best story item IDs from the public Firebase API (up to 200)",
		"BestStoryIDs",
		getBestStoryIDs,
	),
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_ask_story_ids",
		"List current Ask HN story item IDs from the public Firebase API",
		"AskStoryIDs",
		getAskStoryIDs,
	),
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_show_story_ids",
		"List current Show HN story item IDs from the public Firebase API",
		"ShowStoryIDs",
		getShowStoryIDs,
	),
	mcptool.Define[*hn.Client, LimitInput](
		"hn_get_job_story_ids",
		"List current job-listing story item IDs from the public Firebase API",
		"JobStoryIDs",
		getJobStoryIDs,
	),
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_get_max_item_id",
		"Return the largest item ID currently on Hacker News (useful for crawling new items)",
		"MaxItemID",
		getMaxItemID,
	),
	mcptool.Define[*hn.Client, GetItemInput](
		"hn_get_item",
		"Fetch a single Hacker News item (story, comment, job, poll, pollopt) by ID",
		"Item",
		getItem,
	),
	mcptool.Define[*hn.Client, GetUserInput](
		"hn_get_user",
		"Fetch a Hacker News user's public profile (karma, created, about, submitted IDs)",
		"FirebaseUser",
		getUser,
	),
	mcptool.Define[*hn.Client, GetItemsBatchInput](
		"hn_get_items_batch",
		"Fetch multiple Hacker News items concurrently; dead and deleted items are omitted",
		"ItemsBatch",
		getItemsBatch,
	),
	mcptool.Define[*hn.Client, GetTopStoriesExpandedInput](
		"hn_get_top_stories_expanded",
		"Fetch top story IDs and hydrate each into a full Firebase item in one call",
		"TopStoriesExpanded",
		getTopStoriesExpanded,
	),
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_get_updates",
		"Return recently changed item IDs and user profiles for near-realtime monitoring",
		"Updates",
		getUpdates,
	),
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_get_who_is_hiring",
		"Return the current month's 'Ask HN: Who is hiring?' thread (posted by whoishiring)",
		"WhoIsHiring",
		getWhoIsHiring,
	),
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_get_who_wants_to_be_hired",
		"Return the current month's 'Ask HN: Who wants to be hired?' thread",
		"WhoWantsToBeHired",
		getWhoWantsToBeHired,
	),
}
