# hn-go

Go client for [Hacker News](https://news.ycombinator.com) — full programmatic access to all feeds, actions, search and the Firebase API using a session cookie.

## Install

```bash
go get github.com/teslashibe/hn-go
```

## Authentication

Export the raw value of the `user` cookie from a logged-in browser session and pass it to `hn.New`:

```go
client, err := hn.New("teslashibe&Yvut3pZNKZDEpjkDenOxfHyhbyoSDShn",
    hn.WithRateLimit(3 * time.Second), // recommended for GTM agents
)
```

---

## GTM surface — complete coverage map

| GTM use case | HN supports it | Method |
|---|---|---|
| Monitor top stories | ✅ | `Page(ctx, FeedTop, 1)` |
| Monitor newest posts in real-time | ✅ | `Page(ctx, FeedNew, 1)` |
| Engage with Show HN early (before viral) | ✅ | `ShowNew(ctx, 1)` |
| Engage with Ask HN early | ✅ | `AskNew(ctx, 1)` |
| Monitor YC launches | ✅ | `Launches(ctx, 1)` |
| Read a full thread + all comments | ✅ | `Thread(ctx, itemID)` |
| Reply to a comment | ✅ | `GetReplyForm()` + `Comment()` |
| Monitor site-wide recent comments | ✅ | `NewComments(ctx, 1)` |
| Post a Show HN (product launch) | ✅ | `SubmitShowHN(ctx, title, url, "")` |
| Post an Ask HN (user research) | ✅ | `SubmitAskHN(ctx, title, body)` |
| Find the monthly "Who is Hiring" thread | ✅ | `WhoIsHiring(ctx)` |
| Full-text search mentions of your product | ✅ | `Search(ctx, "your product", opts)` |
| Search newest discussions by keyword | ✅ | `SearchByDate(ctx, q, opts)` |
| Near-realtime changed items (polling) | ✅ | `Updates(ctx)` — Firebase |
| Bulk fetch story metadata concurrently | ✅ | `ItemsBatch(ctx, ids, 10)` |
| Read a user's submitted stories | ✅ | `UserSubmissions(ctx, username, 1)` |
| Read a user's comment history | ✅ | `UserThreads(ctx, username, 1)` |
| Read a user's about / contact info | ✅ | `FirebaseUser(ctx, username)` |
| Upvote a story / comment | ✅ | `Upvote(ctx, id, story.VoteUpAuth)` |
| Favourite a story | ✅ | `Fave(ctx, id, story.FaveAuth)` |
| Private messages / DMs | ❌ HN has no DM system | N/A |
| Email / contact a user | ❌ HN has no messaging | Reply publicly or read `.About` for contact info |

---

## API reference

### Feed scraping

```go
ctx := context.Background()

// ── Engagement feeds ──────────────────────────────────────────────────
stories, _ := client.Page(ctx, hn.FeedTop, 1)       // front page
stories, _ := client.Page(ctx, hn.FeedNew, 1)       // newest (all types)
stories, _ := client.ShowNew(ctx, 1)                 // newest Show HN — engage early
stories, _ := client.AskNew(ctx, 1)                  // newest Ask HN — engage early
stories, _ := client.Launches(ctx, 1)                // YC Launch HN posts
stories, _ := client.Page(ctx, hn.FeedBest, 1)      // all-time best
stories, _ := client.Page(ctx, hn.FeedAsk, 1)       // best Ask HN
stories, _ := client.Page(ctx, hn.FeedShow, 1)      // best Show HN
stories, _ := client.Page(ctx, hn.FeedJobs, 1)      // job listings
stories, _ := client.Page(ctx, hn.FeedFront, 1)     // past front pages

// ── Thread + comments ─────────────────────────────────────────────────
thread, _ := client.Thread(ctx, 47834565)
fmt.Println(thread.Story.Title, thread.Story.Points, "pts")
for _, c := range thread.Comments {
    // Level = indent depth; 0 = top-level, 1 = reply, etc.
    fmt.Printf("[L%d] %s: %s\n", c.Level, c.Author, c.Text)
}

// ── Site-wide recent comments (with parent story context) ─────────────
comments, _ := client.NewComments(ctx, 1)
for _, c := range comments {
    fmt.Printf("%s on '%s': %s\n", c.Author, c.ParentStoryTitle, c.Text)
}

// ── Domain-scoped stories ─────────────────────────────────────────────
stories, _ = client.FromDomain(ctx, "github.com", 1)

// ── User activity ─────────────────────────────────────────────────────
stories,  _ = client.UserSubmissions(ctx, "teslashibe", 1)  // their story history
comments, _ = client.UserThreads(ctx, "teslashibe", 1)       // their comment history
stories,  _ = client.Favorites(ctx, "teslashibe", 1)         // their favourited stories
comments, _ = client.FavoriteComments(ctx, "teslashibe", 1)  // their favourited comments
stories,  _ = client.Upvoted(ctx, "teslashibe", 1)           // their upvotes (own only)
```

Each `Story` carries per-item auth tokens ready for action calls:

```go
story.VoteUpAuth  // upvote token
story.HideAuth    // hide token
story.FaveAuth    // favourite token
story.Voted       // true if already upvoted this session
```

---

### Authenticated actions

```go
// ── Vote ──────────────────────────────────────────────────────────────
_ = client.Upvote(ctx, story.ID, story.VoteUpAuth)
_ = client.Unvote(ctx, story.ID, story.VoteUpAuth)

// ── Hide / Favourite / Flag ───────────────────────────────────────────
_ = client.Hide(ctx, story.ID, story.HideAuth)
_ = client.Fave(ctx, story.ID, story.FaveAuth)
_ = client.Flag(ctx, story.ID, flagAuth)  // requires ≥30 karma

// ── Reply to a comment ───────────────────────────────────────────────
form, _ := client.GetReplyForm(ctx, commentID)
_ = client.Comment(ctx, form, "Great post — happy to share feedback.")

// ── Submit posts ──────────────────────────────────────────────────────

// Show HN — product launch (auto-prefixes "Show HN:")
_ = client.SubmitShowHN(ctx, "My Product – do X without Y", "https://myproduct.com", "")

// Ask HN — user research (auto-prefixes "Ask HN:")
_ = client.SubmitAskHN(ctx, "How do you handle X today?", "Context about why I'm asking...")

// Generic submit (full control)
sf, _  := client.GetSubmitForm(ctx)
_ = client.Submit(ctx, sf, "Title", "https://url.com", "")

// ── Update profile ────────────────────────────────────────────────────
settings, _ := client.GetUserSettings(ctx, "teslashibe")
settings.Email = "new@example.com"
_ = client.UpdateUserSettings(ctx, settings)
```

---

### Firebase API (structured + concurrent, no auth)

```go
// ── ID feeds ──────────────────────────────────────────────────────────
ids, _ := client.TopStoryIDs(ctx)    // 500 IDs
ids, _ = client.NewStoryIDs(ctx)     // 500 IDs
ids, _ = client.BestStoryIDs(ctx)    // 200 IDs
ids, _ = client.AskStoryIDs(ctx)
ids, _ = client.ShowStoryIDs(ctx)
ids, _ = client.JobStoryIDs(ctx)

// ── Near-realtime polling ─────────────────────────────────────────────
// Poll every 30–60 s; items = recently changed item IDs.
updates, _ := client.Updates(ctx)
items := client.ItemsBatch(ctx, updates.Items, 10)

// ── Item + user data ──────────────────────────────────────────────────
item, _ := client.Item(ctx, 47834565)
user, _ := client.FirebaseUser(ctx, "teslashibe")
fmt.Println(user.Karma, user.About) // About often contains contact info

// ── Bulk concurrent fetch ─────────────────────────────────────────────
items, _ = client.TopStoriesExpanded(ctx, 50, 10) // top 50, 10 goroutines

max, _ := client.MaxItemID(ctx) // useful for crawling new content
```

---

### Algolia search

```go
// ── By relevance ──────────────────────────────────────────────────────
result, _ := client.Search(ctx, "your product name", hn.SearchOptions{
    Tags:        []hn.SearchTag{hn.TagStory, hn.TagComment},
    HitsPerPage: 30,
    MinPoints:   50,
})

// ── Newest-first (real-time monitoring) ───────────────────────────────
result, _ = client.SearchByDate(ctx, "your product name", hn.SearchOptions{
    Tags:        []hn.SearchTag{hn.TagComment},
    HitsPerPage: 20,
})

// ── Monthly recurring threads ─────────────────────────────────────────
hiring, _  := client.WhoIsHiring(ctx)        // "Ask HN: Who is hiring? (April 2026)"
wanted, _  := client.WhoWantsToBeHired(ctx)  // "Ask HN: Who wants to be hired?"
// Then fetch the full thread:
thread, _ := client.Thread(ctx, hiringID)

fmt.Printf("Total: %d results across %d pages\n", result.NbHits, result.NbPages)
```

Available tags: `TagStory`, `TagComment`, `TagAskHN`, `TagShowHN`, `TagJob`, `TagPoll`, `TagFrontPage`

---

### Rate limiting (important for GTM agents)

```go
// Recommended: 2–5 s between authenticated HN requests to stay below radar.
// Public Firebase + Algolia calls are NOT rate-limited.
client, _ := hn.New(cookie, hn.WithRateLimit(3 * time.Second))
```

---

### Private messaging

**HN has no private messaging system.** `/inbox` and `/messages` return 404. The only way to reach a user is to reply publicly to one of their comments. Their `.About` field (via `FirebaseUser`) sometimes contains a contact email.

---

## Options

```go
client, err := hn.New(cookie,
    hn.WithRateLimit(3 * time.Second),  // GTM agents: always set this
    hn.WithTimeout(30 * time.Second),
    hn.WithHTTPClient(myHTTPClient),
    hn.WithUserAgent("my-bot/1.0"),
)
```
