package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Search queries the Algolia HN search API by relevance.
//
// query is a full-text query string. Pass zero-value SearchOptions{} for defaults.
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	return c.algoliaSearch(ctx, "search", query, opts)
}

// SearchByDate queries the Algolia HN search API sorted by newest-first.
func (c *Client) SearchByDate(ctx context.Context, query string, opts SearchOptions) (*SearchResult, error) {
	return c.algoliaSearch(ctx, "search_by_date", query, opts)
}

func (c *Client) algoliaSearch(ctx context.Context, endpoint, query string, opts SearchOptions) (*SearchResult, error) {
	v := url.Values{}
	v.Set("query", query)

	if len(opts.Tags) > 0 || len(opts.TagsOr) > 0 {
		var parts []string
		for _, t := range opts.Tags {
			parts = append(parts, string(t))
		}
		if len(opts.TagsOr) > 0 {
			orTags := make([]string, len(opts.TagsOr))
			for i, t := range opts.TagsOr {
				orTags[i] = string(t)
			}
			parts = append(parts, "("+strings.Join(orTags, ",")+")")
		}
		v.Set("tags", strings.Join(parts, ","))
	}

	hitsPerPage := opts.HitsPerPage
	if hitsPerPage <= 0 {
		hitsPerPage = 20
	}
	v.Set("hitsPerPage", fmt.Sprintf("%d", hitsPerPage))

	if opts.Page > 0 {
		v.Set("page", fmt.Sprintf("%d", opts.Page))
	}

	var numeric []string
	if opts.MinPoints > 0 {
		numeric = append(numeric, fmt.Sprintf("points>=%d", opts.MinPoints))
	}
	if opts.MinComments > 0 {
		numeric = append(numeric, fmt.Sprintf("num_comments>=%d", opts.MinComments))
	}
	if len(numeric) > 0 {
		v.Set("numericFilters", strings.Join(numeric, ","))
	}

	rawURL := fmt.Sprintf("%s/%s?%s", algoliaBase, endpoint, v.Encode())

	body, err := c.publicGet(ctx, rawURL)
	if err != nil {
		return nil, err
	}

	var result SearchResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("hn: decode search result: %w", err)
	}
	return &result, nil
}
