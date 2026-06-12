package hn

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Credential login for Hacker News.
//
// HN authenticates browser sessions with a single `user` cookie of the form
// "username&token". A logged-in cookie can be pasted directly (see New), but
// the Social OS flow logs in with a username + password: we POST the login
// form, capture the freshly-minted `user` cookie, and self-heal by
// re-logging-in when the session expires. This mirrors the TitlePro247
// pattern (form login, redirects disabled, captured session, relogin + a
// GetMe liveness check).

// Credentials is a Hacker News username/password pair.
type Credentials struct {
	Username string
	Password string
}

// LoginWith logs in to Hacker News with a username + password and returns an
// authenticated Client. The minted `user` cookie is captured for subsequent
// requests, and the credentials are retained so the client can transparently
// re-login when the session expires.
func LoginWith(ctx context.Context, creds Credentials, opts ...Option) (*Client, error) {
	creds.Username = strings.TrimSpace(creds.Username)
	if creds.Username == "" || creds.Password == "" {
		return nil, fmt.Errorf("%w: username and password are required", ErrBadRequest)
	}
	c := &Client{
		userAgent: defaultUserAgent,
		http:      &http.Client{Timeout: defaultTimeout},
		username:  creds.Username,
		password:  creds.Password,
	}
	for _, o := range opts {
		o(c)
	}
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	return c, nil
}

// WithCredentials attaches a username/password to a cookie-constructed client
// so it can self-heal (re-login) when the pasted session expires. Use with
// New when you have both a cookie and the credentials.
func WithCredentials(creds Credentials) Option {
	return func(c *Client) {
		c.username = strings.TrimSpace(creds.Username)
		c.password = creds.Password
	}
}

// login posts the HN login form and captures the `user` session cookie.
// Redirects are disabled so the Set-Cookie on the 302 is observed directly;
// a 200 response means HN re-rendered the login page (bad credentials).
func (c *Client) login(ctx context.Context) error {
	if c.username == "" || c.password == "" {
		return fmt.Errorf("%w: no credentials to log in with", ErrUnauthenticated)
	}
	form := url.Values{
		"acct": {c.username},
		"pw":   {c.password},
		"goto": {"news"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hnBase+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("hn: build login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Origin", hnBase)
	req.Header.Set("Referer", hnBase+"/login")

	noFollow := *c.http
	noFollow.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := noFollow.Do(req)
	if err != nil {
		return fmt.Errorf("hn: login request: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))

	if cookie := userCookieFromResponse(resp); cookie != "" {
		c.setUserCookie(cookie)
		return nil
	}
	// No cookie set: HN re-rendered the login form. Distinguish the common
	// failure modes for a clearer error.
	raw := string(body)
	switch {
	case strings.Contains(raw, "Bad login"):
		return fmt.Errorf("%w: bad login (wrong username or password)", ErrUnauthenticated)
	case strings.Contains(raw, "Validation required") || strings.Contains(raw, "recaptcha"):
		return fmt.Errorf("%w: HN demanded a captcha/validation challenge", ErrForbidden)
	default:
		return fmt.Errorf("%w: login did not set a session cookie (status %d)", ErrUnauthenticated, resp.StatusCode)
	}
}

// userCookieFromResponse extracts the raw value of the `user` cookie from a
// login response's Set-Cookie headers.
func userCookieFromResponse(resp *http.Response) string {
	for _, ck := range resp.Cookies() {
		if ck.Name == "user" && strings.TrimSpace(ck.Value) != "" {
			return ck.Value
		}
	}
	return ""
}

func (c *Client) setUserCookie(cookie string) {
	c.cookieMu.Lock()
	c.userCookie = cookie
	c.cookieMu.Unlock()
}

func (c *Client) currentCookie() string {
	c.cookieMu.Lock()
	defer c.cookieMu.Unlock()
	return c.userCookie
}

// canRelogin reports whether the client holds credentials it can use to mint
// a fresh session after the current one expires.
func (c *Client) canRelogin() bool {
	return c.username != "" && c.password != ""
}

// relogin mints a fresh session, serialized so concurrent callers don't
// stampede the login endpoint.
func (c *Client) relogin(ctx context.Context) error {
	c.loginMu.Lock()
	defer c.loginMu.Unlock()
	return c.login(ctx)
}

// AuthSnapshot returns the current session cookie so a host can persist a
// freshly minted session (mirrors titlepro247-go's AuthSnapshot). Empty when
// the client has never logged in.
func (c *Client) AuthSnapshot() string { return c.currentCookie() }

// GetMe verifies the session is live and returns the authenticated username.
// It self-heals: on an expired session with stored credentials it re-logs-in
// once and retries. Used by the host liveness prober.
func (c *Client) GetMe(ctx context.Context) (string, error) {
	err := c.ValidateSession(ctx)
	if err != nil && errors.Is(err, ErrUnauthenticated) && c.canRelogin() {
		if lerr := c.relogin(ctx); lerr != nil {
			return "", lerr
		}
		err = c.ValidateSession(ctx)
	}
	if err != nil {
		return "", err
	}
	return c.username, nil
}
