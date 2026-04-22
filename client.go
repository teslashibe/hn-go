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
	"strconv"
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
	sessionOnce sync.Once
	sessionErr  error // sticky error from first validation attempt

	// Rate-limit tracking.
	rlMu      sync.Mutex
	rlState   RateLimitState
	gapMu     sync.Mutex
	lastReqAt time.Time
	minGap    time.Duration
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
		c.minGap = d
	}
}

// New returns an authenticated Client.
//
// cookie is the raw value of the "user" session cookie from a logged-in
// browser, e.g. "teslashibe&Yvut3pZNKZDEpjkDenOxfHyhbyoSDShn".
func New(cookie string, opts ...Option) (*Client, error) {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" {
		return nil, ErrUnauthenticated
	}
	if !strings.Contains(cookie, "&") {
		return nil, fmt.Errorf("%w: cookie must contain username&token (got %q)", ErrUnauthenticated, cookie)
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

// ValidateSession checks whether the user cookie is still valid by fetching the
// front page and inspecting the HTML for a logout link. Returns ErrUnauthenticated
// if the session has expired. This is called automatically on the first
// authenticated request. On transient failure the result is NOT cached — the
// next call will retry.
func (c *Client) ValidateSession(ctx context.Context) error {
	body, err := c.doGet(ctx, hnBase+"/", true)
	if err != nil {
		return fmt.Errorf("hn: validate session: %w", err)
	}
	if !isAuthenticated(body) {
		return ErrUnauthenticated
	}
	return nil
}

// ensureSession validates the session exactly once. On success, subsequent
// calls are no-ops. On failure the Once is reset so the next call retries
// (handles transient network errors without permanently poisoning the client).
func (c *Client) ensureSession(ctx context.Context) error {
	c.sessionOnce.Do(func() {
		c.sessionErr = c.ValidateSession(ctx)
	})
	if c.sessionErr != nil {
		err := c.sessionErr
		c.sessionOnce = sync.Once{} // reset so next call retries
		c.sessionErr = nil
		return err
	}
	return nil
}

// --- internal HTTP helpers ---

// hnGet performs an authenticated GET against news.ycombinator.com.
func (c *Client) hnGet(ctx context.Context, path string) ([]byte, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
	if err := c.waitRateLimit(ctx); err != nil {
		return nil, err
	}
	return c.doGet(ctx, hnBase+path, true)
}

// hnGetNoFollow performs a GET that does NOT follow redirects, returning
// (statusCode, Location header). Used for action endpoints that 302 on success.
func (c *Client) hnGetNoFollow(ctx context.Context, path string) (int, string, error) {
	if err := c.ensureSession(ctx); err != nil {
		return 0, "", err
	}
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
	c.updateRateLimit(resp.Header)
	// Drain body so the underlying TCP connection can be reused by keep-alive.
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBody))
	return resp.StatusCode, resp.Header.Get("Location"), nil
}

// hnPost performs an authenticated HTML form POST against news.ycombinator.com.
func (c *Client) hnPost(ctx context.Context, path string, form url.Values) ([]byte, error) {
	if err := c.ensureSession(ctx); err != nil {
		return nil, err
	}
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

// waitRateLimit serialises authenticated requests. Delegates to waitForGap.
func (c *Client) waitRateLimit(ctx context.Context) error {
	c.waitForGap(ctx)
	return ctx.Err()
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

// RateLimit returns a snapshot of the most recently observed rate-limit state.
func (c *Client) RateLimit() RateLimitState {
	c.rlMu.Lock()
	defer c.rlMu.Unlock()
	return c.rlState
}

// updateRateLimit reads standard rate-limit headers and updates tracked state.
func (c *Client) updateRateLimit(h http.Header) {
	c.rlMu.Lock()
	defer c.rlMu.Unlock()
	if v := rlHeader(h, "Limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rlState.Limit = n
		}
	}
	if v := rlHeader(h, "Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rlState.Remaining = n
		}
	}
	if v := rlHeader(h, "Reset"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			if ts > 1_000_000_000 {
				c.rlState.Reset = time.Unix(ts, 0)
			} else {
				c.rlState.Reset = time.Now().Add(time.Duration(ts) * time.Second)
			}
		}
	}
}

// rlHeader returns the value of a rate-limit header, checking four common prefix variants.
func rlHeader(h http.Header, suffix string) string {
	for _, p := range []string{"X-RateLimit-", "X-Rate-Limit-", "X-Ratelimit-", "RateLimit-"} {
		if v := strings.TrimSpace(h.Get(p + suffix)); v != "" {
			return v
		}
	}
	return ""
}

// adaptiveGap returns the delay before the next request based on rate-limit state.
func (c *Client) adaptiveGap() time.Duration {
	c.rlMu.Lock()
	rs := c.rlState
	c.rlMu.Unlock()

	if rs.Remaining == 0 && !rs.Reset.IsZero() {
		if d := time.Until(rs.Reset); d > 0 {
			return d + 50*time.Millisecond
		}
	}
	if rs.Remaining > 0 && !rs.Reset.IsZero() {
		if d := time.Until(rs.Reset); d > 0 {
			spread := d / time.Duration(float64(rs.Remaining)*0.9)
			if spread > c.minGap {
				return spread
			}
		}
	}
	return c.minGap
}

// waitForGap enforces the min request gap, honouring rate-limit state adaptively.
func (c *Client) waitForGap(ctx context.Context) {
	gap := c.adaptiveGap()
	c.gapMu.Lock()
	now := time.Now()
	next := c.lastReqAt.Add(gap)
	if now.After(next) {
		next = now
	}
	c.lastReqAt = next
	c.gapMu.Unlock()

	if wait := time.Until(next); wait > 0 {
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
	c.rlMu.Lock()
	c.rlState.RetryAfter = 0
	c.rlMu.Unlock()
}

// parseRetryAfter parses the Retry-After header (seconds integer, unix timestamp, or HTTP-date).
func parseRetryAfter(val string, fallback time.Duration) time.Duration {
	if val == "" {
		return fallback
	}
	trimmed := strings.TrimSpace(val)
	if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if n > 1_000_000_000 {
			if d := time.Until(time.Unix(n, 0)); d > 0 {
				return d
			}
			return fallback
		}
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(trimmed); err == nil {
		if d := time.Until(t); d > 0 {
			return d
		}
	}
	return fallback
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

	c.updateRateLimit(resp.Header)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusFound, http.StatusMovedPermanently:
		return body, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, ErrForbidden
	case http.StatusNotFound:
		return nil, ErrNotFound
	case http.StatusTooManyRequests:
		wait := parseRetryAfter(resp.Header.Get("Retry-After"), 60*time.Second)
		c.rlMu.Lock()
		c.rlState.Remaining = 0
		c.rlState.RetryAfter = wait
		if c.rlState.Reset.IsZero() || time.Until(c.rlState.Reset) < wait {
			c.rlState.Reset = time.Now().Add(wait)
		}
		c.rlMu.Unlock()
		c.gapMu.Lock()
		if earliest := time.Now().Add(wait); c.lastReqAt.Before(earliest) {
			c.lastReqAt = earliest
		}
		c.gapMu.Unlock()
		return nil, fmt.Errorf("%w: retry after %s", ErrRateLimited, wait)
	default:
		return nil, &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}
}
