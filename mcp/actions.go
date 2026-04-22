package mcp

import (
	"context"

	hn "github.com/teslashibe/hn-go"
	"github.com/teslashibe/mcptool"
)

func validateSession(ctx context.Context, c *hn.Client, _ EmptyInput) (any, error) {
	if err := c.ValidateSession(ctx); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "valid": true}, nil
}

// ItemAuthInput is shared by per-item action tools (upvote, unvote, hide, fave, flag).
//
// Auth tokens are extracted from the corresponding feed page (Story.VoteUpAuth,
// Story.HideAuth, Story.FaveAuth, etc.) and are session+item scoped.
type ItemAuthInput struct {
	ItemID int    `json:"item_id" jsonschema:"description=item ID to act on,required,minimum=1"`
	Auth   string `json:"auth" jsonschema:"description=per-item HMAC auth token from the corresponding feed page,required"`
}

func upvote(ctx context.Context, c *hn.Client, in ItemAuthInput) (any, error) {
	if err := c.Upvote(ctx, in.ItemID, in.Auth); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "item_id": in.ItemID}, nil
}

func unvote(ctx context.Context, c *hn.Client, in ItemAuthInput) (any, error) {
	if err := c.Unvote(ctx, in.ItemID, in.Auth); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "item_id": in.ItemID}, nil
}

func hide(ctx context.Context, c *hn.Client, in ItemAuthInput) (any, error) {
	if err := c.Hide(ctx, in.ItemID, in.Auth); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "item_id": in.ItemID}, nil
}

func fave(ctx context.Context, c *hn.Client, in ItemAuthInput) (any, error) {
	if err := c.Fave(ctx, in.ItemID, in.Auth); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "item_id": in.ItemID}, nil
}

func flag(ctx context.Context, c *hn.Client, in ItemAuthInput) (any, error) {
	if err := c.Flag(ctx, in.ItemID, in.Auth); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "item_id": in.ItemID}, nil
}

// PostCommentInput is the typed input for hn_post_comment.
//
// Fetch ParentID, HMAC, and Goto via hn_get_reply_form first.
type PostCommentInput struct {
	ParentID int    `json:"parent_id" jsonschema:"description=parent comment or story ID (from hn_get_reply_form),required,minimum=1"`
	HMAC     string `json:"hmac" jsonschema:"description=HMAC nonce from hn_get_reply_form,required"`
	Goto     string `json:"goto,omitempty" jsonschema:"description=goto value from hn_get_reply_form (echoed back to HN)"`
	Text     string `json:"text" jsonschema:"description=plain-text reply body,required"`
}

func postComment(ctx context.Context, c *hn.Client, in PostCommentInput) (any, error) {
	form := &hn.ReplyForm{ParentID: in.ParentID, HMAC: in.HMAC, Goto: in.Goto}
	if err := c.Comment(ctx, form, in.Text); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "parent_id": in.ParentID}, nil
}

// SubmitStoryInput is the typed input for hn_submit_story.
//
// Fetch FNID + FNOP via hn_get_submit_form first. Provide either URL (link
// post) or Text (text post), not both.
type SubmitStoryInput struct {
	FNID  string `json:"fnid" jsonschema:"description=one-time fnid nonce from hn_get_submit_form,required"`
	FNOP  string `json:"fnop,omitempty" jsonschema:"description=fnop value from hn_get_submit_form"`
	Title string `json:"title" jsonschema:"description=story title,required"`
	URL   string `json:"url,omitempty" jsonschema:"description=story URL (mutually exclusive with text)"`
	Text  string `json:"text,omitempty" jsonschema:"description=story body for text/Ask posts (mutually exclusive with url)"`
}

func submitStory(ctx context.Context, c *hn.Client, in SubmitStoryInput) (any, error) {
	form := &hn.SubmitForm{FNID: in.FNID, FNOP: in.FNOP}
	if err := c.Submit(ctx, form, in.Title, in.URL, in.Text); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// SubmitShowHNInput is the typed input for hn_submit_show_hn. The 'Show HN: '
// prefix is added automatically by the underlying client.
type SubmitShowHNInput struct {
	Title string `json:"title" jsonschema:"description=title (without 'Show HN:' prefix; added automatically),required"`
	URL   string `json:"url,omitempty" jsonschema:"description=link being shown (mutually exclusive with text)"`
	Text  string `json:"text,omitempty" jsonschema:"description=description for a text-only Show HN (mutually exclusive with url)"`
}

func submitShowHN(ctx context.Context, c *hn.Client, in SubmitShowHNInput) (any, error) {
	if err := c.SubmitShowHN(ctx, in.Title, in.URL, in.Text); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// SubmitAskHNInput is the typed input for hn_submit_ask_hn. The 'Ask HN: '
// prefix is added automatically by the underlying client.
type SubmitAskHNInput struct {
	Title string `json:"title" jsonschema:"description=title (without 'Ask HN:' prefix; added automatically),required"`
	Text  string `json:"text" jsonschema:"description=question body,required"`
}

func submitAskHN(ctx context.Context, c *hn.Client, in SubmitAskHNInput) (any, error) {
	if err := c.SubmitAskHN(ctx, in.Title, in.Text); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true}, nil
}

// UpdateUserSettingsInput is the typed input for hn_update_user_settings.
//
// Fetch the current settings via hn_get_user_settings first to populate ID +
// HMAC; modify the fields you want to change, then send the result.
type UpdateUserSettingsInput struct {
	ID       string `json:"id" jsonschema:"description=username (from hn_get_user_settings),required"`
	HMAC     string `json:"hmac" jsonschema:"description=HMAC nonce from hn_get_user_settings,required"`
	Email    string `json:"email,omitempty" jsonschema:"description=contact email"`
	MaxVis   string `json:"max_vis,omitempty" jsonschema:"description=maxv: minimum visibility threshold"`
	MinAge   string `json:"min_age,omitempty" jsonschema:"description=mina: minimum age in minutes"`
	Delay    string `json:"delay,omitempty" jsonschema:"description=comment delay in minutes"`
	ShowDead bool   `json:"show_dead,omitempty" jsonschema:"description=showd: show [dead] items in feeds"`
	NoPro    bool   `json:"no_pro,omitempty" jsonschema:"description=nopro: opt out of profile metadata"`
}

func updateUserSettings(ctx context.Context, c *hn.Client, in UpdateUserSettingsInput) (any, error) {
	settings := &hn.UserSettings{
		ID:       in.ID,
		HMAC:     in.HMAC,
		Email:    in.Email,
		MaxVis:   in.MaxVis,
		MinAge:   in.MinAge,
		Delay:    in.Delay,
		ShowDead: in.ShowDead,
		NoPro:    in.NoPro,
	}
	if err := c.UpdateUserSettings(ctx, settings); err != nil {
		return nil, err
	}
	return map[string]any{"ok": true, "id": in.ID}, nil
}

var actionTools = []mcptool.Tool{
	mcptool.Define[*hn.Client, EmptyInput](
		"hn_validate_session",
		"Validate that the configured user cookie is still authenticated against news.ycombinator.com",
		"ValidateSession",
		validateSession,
	),
	mcptool.Define[*hn.Client, ItemAuthInput](
		"hn_upvote_item",
		"Upvote an item; auth token must come from the same session's feed page (Story.VoteUpAuth)",
		"Upvote",
		upvote,
	),
	mcptool.Define[*hn.Client, ItemAuthInput](
		"hn_unvote_item",
		"Remove an existing upvote; auth token from Story.VoteUpAuth",
		"Unvote",
		unvote,
	),
	mcptool.Define[*hn.Client, ItemAuthInput](
		"hn_hide_item",
		"Hide an item from your feed; auth token from Story.HideAuth",
		"Hide",
		hide,
	),
	mcptool.Define[*hn.Client, ItemAuthInput](
		"hn_favorite_item",
		"Add an item to the authenticated user's favourites; auth token from Story.FaveAuth",
		"Fave",
		fave,
	),
	mcptool.Define[*hn.Client, ItemAuthInput](
		"hn_flag_item",
		"Flag an item as problematic (requires >=30 karma); auth token from Story flag link",
		"Flag",
		flag,
	),
	mcptool.Define[*hn.Client, PostCommentInput](
		"hn_post_comment",
		"Post a reply to a comment or story; call hn_get_reply_form first to obtain hmac and parent_id",
		"Comment",
		postComment,
	),
	mcptool.Define[*hn.Client, SubmitStoryInput](
		"hn_submit_story",
		"Submit a new story or text post; call hn_get_submit_form first to obtain fnid",
		"Submit",
		submitStory,
	),
	mcptool.Define[*hn.Client, SubmitShowHNInput](
		"hn_submit_show_hn",
		"Submit a Show HN post (the 'Show HN:' prefix is added automatically)",
		"SubmitShowHN",
		submitShowHN,
	),
	mcptool.Define[*hn.Client, SubmitAskHNInput](
		"hn_submit_ask_hn",
		"Submit an Ask HN post (the 'Ask HN:' prefix is added automatically)",
		"SubmitAskHN",
		submitAskHN,
	),
	mcptool.Define[*hn.Client, UpdateUserSettingsInput](
		"hn_update_user_settings",
		"Persist updated profile settings; fetch current values via hn_get_user_settings first",
		"UpdateUserSettings",
		updateUserSettings,
	),
}
