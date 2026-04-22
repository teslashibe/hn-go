# hn-go

Go client for [Hacker News](https://news.ycombinator.com) — full programmatic access to every feed, action, and API endpoint using a session cookie.

```bash
go get github.com/teslashibe/hn-go
```

## Quick start

```go
client, _ := hn.New("username&token", hn.WithRateLimit(3*time.Second))
ctx := context.Background()

stories, _ := client.Page(ctx, hn.FeedTop, 1)
for _, s := range stories {
    fmt.Printf("#%d %s (%d pts)\n", s.Rank, s.Title, s.Points)
}
```

## Authentication

Pass the raw `user` cookie value from a logged-in browser session. The client validates the session on first use and returns `ErrUnauthenticated` if expired.

```go
client, err := hn.New("teslashibe&Yvut3pZNKZDEpjkDenOxfHyhbyoSDShn",
    hn.WithRateLimit(3*time.Second),
)
```

## Feeds

```go
stories, _ := client.Page(ctx, hn.FeedTop, 1)      // front page
stories, _ = client.Page(ctx, hn.FeedNew, 1)        // newest
stories, _ = client.Page(ctx, hn.FeedBest, 1)       // best
stories, _ = client.Page(ctx, hn.FeedAsk, 1)        // Ask HN
stories, _ = client.Page(ctx, hn.FeedShow, 1)       // Show HN
stories, _ = client.Page(ctx, hn.FeedJobs, 1)       // jobs
stories, _ = client.Page(ctx, hn.FeedFront, 1)      // past front pages

stories, _ = client.ShowNew(ctx, 1)                  // newest Show HN
stories, _ = client.AskNew(ctx, 1)                   // newest Ask HN
stories, _ = client.Launches(ctx, 1)                 // YC Launch HN

stories, _ = client.FromDomain(ctx, "github.com", 1) // by domain
stories, _ = client.UserSubmissions(ctx, "pg", 1)    // user's posts
stories, _ = client.Favorites(ctx, "pg", 1)          // user's favourites
stories, _ = client.Upvoted(ctx, "teslashibe", 1)    // own upvotes
```

## Threads and comments

```go
thread, _ := client.Thread(ctx, 47834565)
fmt.Println(thread.Story.Title, len(thread.Comments), "comments")

for _, c := range thread.Comments {
    fmt.Printf("[L%d] %s: %s\n", c.Level, c.Author, c.Text)
}

// Site-wide recent comments (includes parent story context)
comments, _ := client.NewComments(ctx, 1)
for _, c := range comments {
    fmt.Printf("%s on '%s'\n", c.Author, c.ParentStoryTitle)
}

// User comment history
comments, _ = client.UserThreads(ctx, "pg", 1)
comments, _ = client.FavoriteComments(ctx, "pg", 1)
```

## Actions

Every action token is extracted automatically from scraped pages.

```go
// Vote
_ = client.Upvote(ctx, story.ID, story.VoteUpAuth)
_ = client.Unvote(ctx, story.ID, story.VoteUpAuth)

// Favourite
_ = client.Fave(ctx, story.ID, story.FaveAuth)

// Hide
_ = client.Hide(ctx, story.ID, story.HideAuth)

// Reply to a comment
form, _ := client.GetReplyForm(ctx, commentID)
_ = client.Comment(ctx, form, "Great post.")

// Submit a Show HN
_ = client.SubmitShowHN(ctx, "My Product", "https://example.com", "")

// Submit an Ask HN
_ = client.SubmitAskHN(ctx, "How do you handle X?", "Context here.")

// Generic submit
sf, _ := client.GetSubmitForm(ctx)
_ = client.Submit(ctx, sf, "Title", "https://url.com", "")

// Update profile
settings, _ := client.GetUserSettings(ctx, "teslashibe")
settings.Email = "new@example.com"
_ = client.UpdateUserSettings(ctx, settings)
```

## Firebase API

Public structured data — no authentication required.

```go
ids, _ := client.TopStoryIDs(ctx)    // 500 IDs
ids, _ = client.NewStoryIDs(ctx)     // 500 IDs
ids, _ = client.BestStoryIDs(ctx)    // 200 IDs
ids, _ = client.AskStoryIDs(ctx)
ids, _ = client.ShowStoryIDs(ctx)
ids, _ = client.JobStoryIDs(ctx)

item, _ := client.Item(ctx, 47834565)
user, _ := client.FirebaseUser(ctx, "pg")
max, _  := client.MaxItemID(ctx)

// Concurrent batch fetch (preserves input order)
items := client.ItemsBatch(ctx, ids[:50], 10)

// Hydrate top stories
items, _ = client.TopStoriesExpanded(ctx, 50, 10)

// Near-realtime polling
updates, _ := client.Updates(ctx)
```

## Search

Algolia HN search API with relevance and date sorting.

```go
result, _ := client.Search(ctx, "AI agent", hn.SearchOptions{
    Tags:        []hn.SearchTag{hn.TagStory},
    HitsPerPage: 30,
    MinPoints:   100,
})

// Newest first
result, _ = client.SearchByDate(ctx, "golang", hn.SearchOptions{
    Tags: []hn.SearchTag{hn.TagShowHN},
})

// OR search (stories OR comments)
result, _ = client.Search(ctx, "your product", hn.SearchOptions{
    TagsOr: []hn.SearchTag{hn.TagStory, hn.TagComment},
})

// Monthly hiring threads
hiring, _ := client.WhoIsHiring(ctx)
wanted, _ := client.WhoWantsToBeHired(ctx)
```

Tags: `TagStory`, `TagComment`, `TagAskHN`, `TagShowHN`, `TagJob`, `TagPoll`, `TagFrontPage`

## Error handling

All errors are sentinel-wrapped for programmatic handling:

```go
if errors.Is(err, hn.ErrUnauthenticated) { /* session expired */ }
if errors.Is(err, hn.ErrBadRequest)      { /* missing/invalid parameter */ }
if errors.Is(err, hn.ErrForbidden)       { /* insufficient karma */ }
if errors.Is(err, hn.ErrRateLimited)     { /* slow down */ }
if errors.Is(err, hn.ErrNotFound)        { /* item doesn't exist */ }
if errors.Is(err, hn.ErrParseFailure)    { /* HTML structure changed */ }
```

## Options

```go
client, _ := hn.New(cookie,
    hn.WithRateLimit(3*time.Second),   // min gap between HN requests
    hn.WithTimeout(30*time.Second),    // HTTP timeout
    hn.WithHTTPClient(customClient),   // custom transport
    hn.WithUserAgent("my-agent/1.0"),  // custom user-agent
)
```

## Live-tested endpoints

Every endpoint below has been verified against the live HN site:

| Endpoint | Method | Verified |
|---|---|---|
| Front page + all feeds | `Page()` | Yes |
| Show HN / Ask HN / Launches (newest) | `ShowNew()` `AskNew()` `Launches()` | Yes |
| Thread + comments | `Thread()` | Yes |
| Site-wide new comments | `NewComments()` | Yes |
| Upvote | `Upvote()` | Yes |
| Unvote | `Unvote()` | Yes |
| Favourite | `Fave()` | Yes |
| Reply form | `GetReplyForm()` | Yes |
| Comment | `Comment()` | Yes |
| Submit form | `GetSubmitForm()` | Yes |
| Submit (Show HN / Ask HN) | `Submit()` `SubmitShowHN()` `SubmitAskHN()` | Yes |
| User settings | `GetUserSettings()` `UpdateUserSettings()` | Yes |
| Session validation | `ValidateSession()` | Yes |
| Firebase: items, users, feeds | `Item()` `FirebaseUser()` `TopStoryIDs()` etc. | Yes |
| Firebase: batch + updates | `ItemsBatch()` `Updates()` | Yes |
| Algolia: search + date sort | `Search()` `SearchByDate()` | Yes |
| Algolia: OR tags | `TagsOr` | Yes |
| Algolia: Who is Hiring | `WhoIsHiring()` | Yes |
| Domain filter | `FromDomain()` | Yes |
| User activity | `UserSubmissions()` `UserThreads()` `Favorites()` `Upvoted()` | Yes |

## MCP support

This package ships an [MCP](https://modelcontextprotocol.io/) tool surface in `./mcp` for use with [`teslashibe/mcptool`](https://github.com/teslashibe/mcptool)-compatible hosts (e.g. [`teslashibe/agent-setup`](https://github.com/teslashibe/agent-setup)). 42 tools cover the full client API: every HTML feed (top, new, best, ask, show, jobs, launches, front, by-domain, by-user), full thread + comment fetch, the public Firebase REST API (item, user, top/new/best/ask/show/job IDs, batch hydration, updates, hiring threads, max item ID), Algolia search (relevance + by-date with tag/numeric filters), and every authenticated action (upvote, unvote, hide, favourite, flag, post comment, submit story / Show HN / Ask HN, update user settings, validate session).

```go
import (
    "github.com/teslashibe/mcptool"
    hn "github.com/teslashibe/hn-go"
    hnmcp "github.com/teslashibe/hn-go/mcp"
)

client, _ := hn.New("user&token")
provider := hnmcp.Provider{}
for _, tool := range provider.Tools() {
    // register tool with your MCP server, passing client as the
    // opaque client argument when invoking
}
```

A coverage test in `mcp/mcp_test.go` fails if a new exported method is added to `*Client` without either being wrapped by an MCP tool or being added to `mcp.Excluded` with a reason — keeping the MCP surface in lockstep with the package API is enforced by CI rather than convention.

## Limitations

- **No private messaging** — HN has no DM system. Reply publicly or read `FirebaseUser.About` for contact info.
- **No post deletion** — HN does not expose a delete endpoint via the web UI for stories (only within a short window, and only via the edit page which is not consistently available).
- **Session cookies expire** — the client detects this automatically on first use. Handle `ErrUnauthenticated` to prompt for a fresh cookie.
