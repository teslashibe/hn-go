package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// --- Firebase REST API (public, no authentication required) ---

// TopStoryIDs returns up to 500 current top-story IDs.
func (c *Client) TopStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "topstories")
}

// NewStoryIDs returns up to 500 newest story IDs.
func (c *Client) NewStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "newstories")
}

// BestStoryIDs returns up to 200 best story IDs.
func (c *Client) BestStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "beststories")
}

// AskStoryIDs returns current Ask HN story IDs.
func (c *Client) AskStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "askstories")
}

// ShowStoryIDs returns current Show HN story IDs.
func (c *Client) ShowStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "showstories")
}

// JobStoryIDs returns current job-listing story IDs.
func (c *Client) JobStoryIDs(ctx context.Context) ([]int, error) {
	return c.firebaseFeed(ctx, "jobstories")
}

// MaxItemID returns the largest item ID currently in HN (useful for crawling
// new items without polling every feed).
func (c *Client) MaxItemID(ctx context.Context) (int, error) {
	body, err := c.publicGet(ctx, firebaseBase+"/maxitem.json")
	if err != nil {
		return 0, err
	}
	var id int
	return id, json.Unmarshal(body, &id)
}

// Item fetches a single item (story, comment, job, poll, pollopt) by ID.
func (c *Client) Item(ctx context.Context, id int) (*FirebaseItem, error) {
	body, err := c.publicGet(ctx, fmt.Sprintf("%s/item/%d.json", firebaseBase, id))
	if err != nil {
		return nil, err
	}
	var item FirebaseItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("hn: decode item %d: %w", id, err)
	}
	return &item, nil
}

// FirebaseUser fetches a user's public profile by username.
func (c *Client) FirebaseUser(ctx context.Context, username string) (*FirebaseUser, error) {
	body, err := c.publicGet(ctx, fmt.Sprintf("%s/user/%s.json", firebaseBase, username))
	if err != nil {
		return nil, err
	}
	var user FirebaseUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, fmt.Errorf("hn: decode user %s: %w", username, err)
	}
	return &user, nil
}

// ItemsBatch fetches multiple items concurrently up to concurrency goroutines.
// Dead or deleted items are silently omitted from the result.
func (c *Client) ItemsBatch(ctx context.Context, ids []int, concurrency int) []*FirebaseItem {
	if concurrency <= 0 {
		concurrency = 10
	}
	sem := make(chan struct{}, concurrency)
	var (
		mu      sync.Mutex
		results = make([]*FirebaseItem, 0, len(ids))
		wg      sync.WaitGroup
	)

	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item, err := c.Item(ctx, id)
			if err != nil || item == nil || item.Dead || item.Deleted {
				return
			}
			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return results
}

// TopStoriesExpanded hydrates the top story IDs via Firebase.
// limit caps the number of IDs to fetch (0 = all 500).
func (c *Client) TopStoriesExpanded(ctx context.Context, limit, concurrency int) ([]*FirebaseItem, error) {
	ids, err := c.TopStoryIDs(ctx)
	if err != nil {
		return nil, err
	}
	if limit > 0 && limit < len(ids) {
		ids = ids[:limit]
	}
	return c.ItemsBatch(ctx, ids, concurrency), nil
}

// Updates returns the most recently changed item IDs and user profiles.
// Poll this every 30–60 s for near-realtime monitoring of new/updated content.
func (c *Client) Updates(ctx context.Context) (*FirebaseUpdates, error) {
	body, err := c.publicGet(ctx, firebaseBase+"/updates.json")
	if err != nil {
		return nil, err
	}
	var u FirebaseUpdates
	return &u, json.Unmarshal(body, &u)
}

// WhoIsHiring returns the current month's "Ask HN: Who is hiring?" thread ID.
// The thread is posted on the 1st of each month by user `whoishiring`.
func (c *Client) WhoIsHiring(ctx context.Context) (*SearchHit, error) {
	return c.latestWhoIsHiringThread(ctx, "who is hiring")
}

// WhoWantsToBeHired returns the current "Ask HN: Who wants to be hired?" thread.
func (c *Client) WhoWantsToBeHired(ctx context.Context) (*SearchHit, error) {
	return c.latestWhoIsHiringThread(ctx, "who wants to be hired")
}

func (c *Client) latestWhoIsHiringThread(ctx context.Context, keyword string) (*SearchHit, error) {
	result, err := c.SearchByDate(ctx, keyword, SearchOptions{
		Tags:        []SearchTag{TagAskHN, "author_whoishiring"},
		HitsPerPage: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Hits) == 0 {
		return nil, ErrNotFound
	}
	return &result.Hits[0], nil
}

func (c *Client) firebaseFeed(ctx context.Context, name string) ([]int, error) {
	body, err := c.publicGet(ctx, fmt.Sprintf("%s/%s.json", firebaseBase, name))
	if err != nil {
		return nil, err
	}
	var ids []int
	return ids, json.Unmarshal(body, &ids)
}
