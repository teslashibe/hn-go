// Package hn is a Go client for Hacker News (news.ycombinator.com).
//
// It covers the full programmatic surface of the site:
//   - All HTML feed pages (top, new, best, ask, show, jobs, …) via HTML scraping.
//   - The public HN Firebase REST API for structured item / user data.
//   - The Algolia HN search API for full-text and date-sorted search.
//   - All authenticated actions (upvote, unvote, hide, favourite, comment, submit).
//
// Authentication uses the "user" session cookie obtained from a logged-in
// browser session; no username/password is sent over the wire.
package hn

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	hnBase      = "https://news.ycombinator.com"
	firebaseBase = "https://hacker-news.firebaseio.com/v0"
	algoliaBase  = "https://hn.algolia.com/api/v1"

	defaultTimeout   = 15 * time.Second
	defaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
)

// Client is an authenticated Hacker News client.
type Client struct {
	userCookie    string // raw value of the "user" cookie
	http          *http.Client
	userAgent     string
	rateLimit     time.Duration // min delay between authenticated HN requests (0 = no limit)
	rateMu        sync.Mutex
	lastHNRequest time.Time
}

// Option customises a Client.
type Option func(*Client)

// WithHTTPClient swaps the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) {
		if h != nil {
			c.http = h
		}
	}
}

// WithTimeout overrides the default 15 s request timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.http.Timeout = d
	}
}

// WithUserAgent overrides the browser user-agent string.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithRateLimit enforces a minimum delay between authenticated requests to
// news.ycombinator.com. Recommended: 2–5 s for a GTM agent to avoid bans.
// Public Firebase / Algolia requests are not rate-limited.
func WithRateLimit(d time.Duration) Option {
	return func(c *Client) {
		c.rateLimit = d
	}
}

// New returns an authenticated Client.
//
// cookie is the raw value of the "user" session cookie from a logged-in
// browser, e.g. "teslashibe&Yvut3pZNKZDEpjkDenOxfHyhbyoSDShn".
func New(cookie string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(cookie) == "" {
		return nil, ErrUnauthenticated
	}
	c := &Client{
		userCookie: cookie,
		userAgent:  defaultUserAgent,
		http: &http.Client{
			Timeout: defaultTimeout,
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

// --- internal HTTP helpers ---

// hnGet performs an authenticated GET against news.ycombinator.com.
func (c *Client) hnGet(ctx context.Context, path string) ([]byte, error) {
	if err := c.waitRateLimit(ctx); err != nil {
		return nil, err
	}
	return c.doGet(ctx, hnBase+path, true)
}

// hnGetNoFollow performs a GET that does NOT follow redirects, returning
// (statusCode, Location header). Used for action endpoints that 302 on success.
func (c *Client) hnGetNoFollow(ctx context.Context, path string) (int, string, error) {
	if err := c.waitRateLimit(ctx); err != nil {
		return 0, "", err
	}
	req, err := c.newHNReq(ctx, http.MethodGet, hnBase+path, nil)
	if err != nil {
		return 0, "", err
	}
	noFollow := *c.http
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := noFollow.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("hn: request: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("Location"), nil
}

// hnPost performs an authenticated HTML form POST against news.ycombinator.com.
func (c *Client) hnPost(ctx context.Context, path string, form url.Values) ([]byte, error) {
	if err := c.waitRateLimit(ctx); err != nil {
		return nil, err
	}
	req, err := c.newHNReq(ctx, http.MethodPost, hnBase+path, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return c.execute(req)
}

// publicGet performs an unauthenticated GET (Firebase, Algolia).
func (c *Client) publicGet(ctx context.Context, rawURL string) ([]byte, error) {
	return c.doGet(ctx, rawURL, false)
}

func (c *Client) doGet(ctx context.Context, rawURL string, auth bool) ([]byte, error) {
	var req *http.Request
	var err error
	if auth {
		req, err = c.newHNReq(ctx, http.MethodGet, rawURL, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err == nil {
			req.Header.Set("User-Agent", c.userAgent)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("hn: build request: %w", err)
	}
	return c.execute(req)
}

// waitRateLimit serialises authenticated requests so that at most one fires
// per rateLimit interval. The lock is held across the sleep to prevent
// concurrent callers from reading a stale lastHNRequest and bypassing the gap.
func (c *Client) waitRateLimit(ctx context.Context) error {
	if c.rateLimit == 0 {
		return nil
	}
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	wait := time.Until(c.lastHNRequest.Add(c.rateLimit))
	if wait > 0 {
		c.rateMu.Unlock()
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			c.rateMu.Lock()
			return ctx.Err()
		}
		c.rateMu.Lock()
	}
	c.lastHNRequest = time.Now()
	return nil
}

func (c *Client) newHNReq(ctx context.Context, method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, rawURL, body)
	if err != nil {
		return nil, fmt.Errorf("hn: build request: %w", err)
	}
	req.Header.Set("Cookie", "user="+c.userCookie)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", hnBase+"/")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	return req, nil
}

const maxResponseBody = 10 << 20 // 10 MB

func (c *Client) execute(req *http.Request) ([]byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hn: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, fmt.Errorf("hn: read body: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusOK, http.StatusFound, http.StatusMovedPermanently:
		return body, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		return nil, ErrRateLimited
	default:
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}
}
