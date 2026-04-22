package mcp

import (
	"context"

	hn "github.com/teslashibe/hn-go"
	"github.com/teslashibe/mcptool"
)

// SearchInput is the typed input for hn_search and hn_search_by_date.
//
// Tags are AND-ed together; pass TagsOr for an OR-grouped tag set (Algolia
// syntax: tags=tag1,(orTag1,orTag2)). Useful tag values: story, comment,
// ask_hn, show_hn, job, poll, front_page.
type SearchInput struct {
	Query       string   `json:"query" jsonschema:"description=full-text search query,required"`
	Tags        []string `json:"tags,omitempty" jsonschema:"description=AND-ed tag filters; allowed: story,comment,ask_hn,show_hn,job,poll,front_page"`
	TagsOr      []string `json:"tags_or,omitempty" jsonschema:"description=OR-grouped tag filters combined with tags via AND"`
	Page        int      `json:"page,omitempty" jsonschema:"description=zero-based page index,minimum=0,default=0"`
	HitsPerPage int      `json:"hits_per_page,omitempty" jsonschema:"description=hits per page,minimum=1,maximum=100,default=20"`
	MinPoints   int      `json:"min_points,omitempty" jsonschema:"description=filter results with points >= min_points (0 = off),minimum=0"`
	MinComments int      `json:"min_comments,omitempty" jsonschema:"description=filter results with num_comments >= min_comments (0 = off),minimum=0"`
}

func toSearchOptions(in SearchInput) hn.SearchOptions {
	opts := hn.SearchOptions{
		Page:        in.Page,
		HitsPerPage: in.HitsPerPage,
		MinPoints:   in.MinPoints,
		MinComments: in.MinComments,
	}
	for _, t := range in.Tags {
		opts.Tags = append(opts.Tags, hn.SearchTag(t))
	}
	for _, t := range in.TagsOr {
		opts.TagsOr = append(opts.TagsOr, hn.SearchTag(t))
	}
	return opts
}

func searchRelevance(ctx context.Context, c *hn.Client, in SearchInput) (any, error) {
	res, err := c.Search(ctx, in.Query, toSearchOptions(in))
	if err != nil {
		return nil, err
	}
	limit := in.HitsPerPage
	if limit <= 0 {
		limit = 20
	}
	return mcptool.PageOf(res.Hits, "", limit), nil
}

func searchByDate(ctx context.Context, c *hn.Client, in SearchInput) (any, error) {
	res, err := c.SearchByDate(ctx, in.Query, toSearchOptions(in))
	if err != nil {
		return nil, err
	}
	limit := in.HitsPerPage
	if limit <= 0 {
		limit = 20
	}
	return mcptool.PageOf(res.Hits, "", limit), nil
}

var searchTools = []mcptool.Tool{
	mcptool.Define[*hn.Client, SearchInput](
		"hn_search",
		"Full-text search Hacker News via the Algolia API, sorted by relevance",
		"Search",
		searchRelevance,
	),
	mcptool.Define[*hn.Client, SearchInput](
		"hn_search_by_date",
		"Full-text search Hacker News via the Algolia API, sorted newest-first",
		"SearchByDate",
		searchByDate,
	),
}
