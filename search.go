package hn

import (
	"context"
	"encoding/json"
	"fmt"
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
	params := make([]string, 0, 8)
	params = append(params, "query="+encodeQuery(query))

	if len(opts.Tags) > 0 {
		tags := make([]string, len(opts.Tags))
		for i, t := range opts.Tags {
			tags[i] = string(t)
		}
		params = append(params, "tags="+strings.Join(tags, ","))
	}

	hitsPerPage := opts.HitsPerPage
	if hitsPerPage <= 0 {
		hitsPerPage = 20
	}
	params = append(params, fmt.Sprintf("hitsPerPage=%d", hitsPerPage))

	if opts.Page > 0 {
		params = append(params, fmt.Sprintf("page=%d", opts.Page))
	}

	var numeric []string
	if opts.MinPoints > 0 {
		numeric = append(numeric, fmt.Sprintf("points>=%d", opts.MinPoints))
	}
	if opts.MinComments > 0 {
		numeric = append(numeric, fmt.Sprintf("num_comments>=%d", opts.MinComments))
	}
	if len(numeric) > 0 {
		params = append(params, "numericFilters="+encodeQuery(strings.Join(numeric, ",")))
	}

	rawURL := fmt.Sprintf("%s/%s?%s", algoliaBase, endpoint, strings.Join(params, "&"))

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

// encodeQuery does minimal percent-encoding for Algolia query strings.
func encodeQuery(s string) string {
	return strings.NewReplacer(
		" ", "%20",
		"#", "%23",
		"&", "%26",
		"=", "%3D",
		"+", "%2B",
		",", "%2C",
	).Replace(s)
}
