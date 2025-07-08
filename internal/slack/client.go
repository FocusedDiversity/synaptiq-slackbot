package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client interface defines Slack API operations.
type Client interface {
	// Message operations
	PostMessage(ctx context.Context, channel string, opts ...MessageOption) (string, error)
	PostEphemeral(ctx context.Context, channel, userID string, opts ...MessageOption) (string, error)
	UpdateMessage(ctx context.Context, channel, timestamp string, opts ...MessageOption) error
	DeleteMessage(ctx context.Context, channel, timestamp string) error

	// Modal operations
	OpenModal(ctx context.Context, triggerID string, modal *Modal) error
	UpdateModal(ctx context.Context, viewID string, modal *Modal) error
	PushModal(ctx context.Context, triggerID string, modal *Modal) error

	// User operations
	GetUserInfo(ctx context.Context, userID string) (*UserInfo, error)
	GetUserByEmail(ctx context.Context, email string) (*UserInfo, error)

	// Channel operations
	GetChannelInfo(ctx context.Context, channelID string) (*ConversationInfo, error)
	ListChannelMembers(ctx context.Context, channelID string) ([]string, error)

	// DM operations
	OpenDM(ctx context.Context, userID string) (string, error)
}

// client implements the Client interface.
type client struct {
	token      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Slack client.
func NewClient(token string) Client {
	return &client{
		token: token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://slack.com/api",
	}
}

// MessageOption is a function that modifies a message.
type MessageOption func(*Message)

// WithText sets the message text.
func WithText(text string) MessageOption {
	return func(m *Message) {
		m.Text = text
	}
}

// WithBlocks sets the message blocks.
func WithBlocks(blocks ...Block) MessageOption {
	return func(m *Message) {
		m.Blocks = blocks
	}
}

// WithThreadTS sets the thread timestamp.
func WithThreadTS(threadTS string) MessageOption {
	return func(m *Message) {
		m.ThreadTS = threadTS
	}
}

// WithMetadata sets the message metadata.
func WithMetadata(eventType string, payload map[string]interface{}) MessageOption {
	return func(m *Message) {
		m.Metadata = &Metadata{
			EventType:    eventType,
			EventPayload: payload,
		}
	}
}

// PostMessage posts a message to a channel.
func (c *client) PostMessage(ctx context.Context, channel string, opts ...MessageOption) (string, error) {
	msg := &Message{
		Channel: channel,
		AsUser:  true,
	}

	for _, opt := range opts {
		opt(msg)
	}

	resp, err := c.callAPI(ctx, "chat.postMessage", msg)
	if err != nil {
		return "", err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
		TS    string `json:"ts"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.TS, nil
}

// PostEphemeral posts an ephemeral message.
func (c *client) PostEphemeral(ctx context.Context, channel, userID string, opts ...MessageOption) (string, error) {
	msg := &struct {
		*Message
		User string `json:"user"`
	}{
		Message: &Message{Channel: channel},
		User:    userID,
	}

	for _, opt := range opts {
		opt(msg.Message)
	}

	resp, err := c.callAPI(ctx, "chat.postEphemeral", msg)
	if err != nil {
		return "", err
	}

	var result struct {
		OK        bool   `json:"ok"`
		Error     string `json:"error,omitempty"`
		MessageTS string `json:"message_ts"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.MessageTS, nil
}

// UpdateMessage updates an existing message.
func (c *client) UpdateMessage(ctx context.Context, channel, timestamp string, opts ...MessageOption) error {
	msg := &struct {
		*Message
		TS string `json:"ts"`
	}{
		Message: &Message{Channel: channel},
		TS:      timestamp,
	}

	for _, opt := range opts {
		opt(msg.Message)
	}

	resp, err := c.callAPI(ctx, "chat.update", msg)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// DeleteMessage deletes a message.
func (c *client) DeleteMessage(ctx context.Context, channel, timestamp string) error {
	params := map[string]interface{}{
		"channel": channel,
		"ts":      timestamp,
	}

	resp, err := c.callAPI(ctx, "chat.delete", params)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// OpenModal opens a modal dialog.
func (c *client) OpenModal(ctx context.Context, triggerID string, modal *Modal) error {
	params := map[string]interface{}{
		"trigger_id": triggerID,
		"view":       modal,
	}

	resp, err := c.callAPI(ctx, "views.open", params)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// UpdateModal updates an existing modal.
func (c *client) UpdateModal(ctx context.Context, viewID string, modal *Modal) error {
	params := map[string]interface{}{
		"view_id": viewID,
		"view":    modal,
	}

	resp, err := c.callAPI(ctx, "views.update", params)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// PushModal pushes a new modal onto the stack.
func (c *client) PushModal(ctx context.Context, triggerID string, modal *Modal) error {
	params := map[string]interface{}{
		"trigger_id": triggerID,
		"view":       modal,
	}

	resp, err := c.callAPI(ctx, "views.push", params)
	if err != nil {
		return err
	}

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("slack API error: %s", result.Error)
	}

	return nil
}

// GetUserInfo gets information about a user.
func (c *client) GetUserInfo(ctx context.Context, userID string) (*UserInfo, error) {
	params := map[string]string{
		"user": userID,
	}

	resp, err := c.callAPIWithParams(ctx, "users.info", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK    bool     `json:"ok"`
		Error string   `json:"error,omitempty"`
		User  UserInfo `json:"user"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result.User, nil
}

// GetUserByEmail gets user info by email.
func (c *client) GetUserByEmail(ctx context.Context, email string) (*UserInfo, error) {
	params := map[string]string{
		"email": email,
	}

	resp, err := c.callAPIWithParams(ctx, "users.lookupByEmail", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK    bool     `json:"ok"`
		Error string   `json:"error,omitempty"`
		User  UserInfo `json:"user"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result.User, nil
}

// GetChannelInfo gets information about a channel.
func (c *client) GetChannelInfo(ctx context.Context, channelID string) (*ConversationInfo, error) {
	params := map[string]string{
		"channel": channelID,
	}

	resp, err := c.callAPIWithParams(ctx, "conversations.info", params)
	if err != nil {
		return nil, err
	}

	var result struct {
		OK      bool             `json:"ok"`
		Error   string           `json:"error,omitempty"`
		Channel ConversationInfo `json:"channel"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("slack API error: %s", result.Error)
	}

	return &result.Channel, nil
}

// ListChannelMembers lists members of a channel.
func (c *client) ListChannelMembers(ctx context.Context, channelID string) ([]string, error) {
	var members []string
	cursor := ""

	for {
		params := map[string]string{
			"channel": channelID,
			"limit":   "200",
		}

		if cursor != "" {
			params["cursor"] = cursor
		}

		resp, err := c.callAPIWithParams(ctx, "conversations.members", params)
		if err != nil {
			return nil, err
		}

		var result struct {
			OK               bool     `json:"ok"`
			Error            string   `json:"error,omitempty"`
			Members          []string `json:"members"`
			ResponseMetadata struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		if err := json.Unmarshal(resp, &result); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if !result.OK {
			return nil, fmt.Errorf("slack API error: %s", result.Error)
		}

		members = append(members, result.Members...)

		if result.ResponseMetadata.NextCursor == "" {
			break
		}

		cursor = result.ResponseMetadata.NextCursor
	}

	return members, nil
}

// OpenDM opens a direct message channel with a user.
func (c *client) OpenDM(ctx context.Context, userID string) (string, error) {
	params := map[string]interface{}{
		"users": userID,
	}

	resp, err := c.callAPI(ctx, "conversations.open", params)
	if err != nil {
		return "", err
	}

	var result struct {
		OK      bool   `json:"ok"`
		Error   string `json:"error,omitempty"`
		Channel struct {
			ID string `json:"id"`
		} `json:"channel"`
	}

	if err := json.Unmarshal(resp, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.OK {
		return "", fmt.Errorf("slack API error: %s", result.Error)
	}

	return result.Channel.ID, nil
}

// callAPI makes an API call with JSON body.
func (c *client) callAPI(ctx context.Context, method string, params interface{}) ([]byte, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/"+method, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// callAPIWithParams makes an API call with URL parameters.
func (c *client) callAPIWithParams(ctx context.Context, method string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(c.baseURL + "/" + method)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}
