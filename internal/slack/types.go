package slack

import (
	"time"
)

// Modal represents a Slack modal view.
type Modal struct {
	Type            string     `json:"type"`
	Title           *TextBlock `json:"title"`
	Submit          *TextBlock `json:"submit,omitempty"`
	Close           *TextBlock `json:"close,omitempty"`
	Blocks          []Block    `json:"blocks"`
	PrivateMetadata string     `json:"private_metadata,omitempty"`
	CallbackID      string     `json:"callback_id,omitempty"`
	ClearOnClose    bool       `json:"clear_on_close,omitempty"`
	NotifyOnClose   bool       `json:"notify_on_close,omitempty"`
}

// Block is an interface for Slack blocks.
type Block interface {
	BlockType() string
}

// TextBlock represents a text object.
type TextBlock struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji,omitempty"`
}

// SectionBlock represents a section block.
type SectionBlock struct {
	Type      string      `json:"type"`
	Text      *TextBlock  `json:"text,omitempty"`
	BlockID   string      `json:"block_id,omitempty"`
	Fields    []TextBlock `json:"fields,omitempty"`
	Accessory interface{} `json:"accessory,omitempty"`
}

func (s SectionBlock) BlockType() string { return "section" }

// HeaderBlock represents a header block.
type HeaderBlock struct {
	Type    string     `json:"type"`
	Text    *TextBlock `json:"text"`
	BlockID string     `json:"block_id,omitempty"`
}

func (h HeaderBlock) BlockType() string { return "header" }

// InputBlock represents an input block.
type InputBlock struct {
	Type     string      `json:"type"`
	BlockID  string      `json:"block_id"`
	Label    *TextBlock  `json:"label"`
	Element  interface{} `json:"element"`
	Optional bool        `json:"optional,omitempty"`
	Hint     *TextBlock  `json:"hint,omitempty"`
}

func (i InputBlock) BlockType() string { return "input" }

// DividerBlock represents a divider block.
type DividerBlock struct {
	Type string `json:"type"`
}

func (d DividerBlock) BlockType() string { return "divider" }

// PlainTextInputElement represents a plain text input.
type PlainTextInputElement struct {
	Type         string     `json:"type"`
	ActionID     string     `json:"action_id"`
	Placeholder  *TextBlock `json:"placeholder,omitempty"`
	InitialValue string     `json:"initial_value,omitempty"`
	Multiline    bool       `json:"multiline,omitempty"`
	MinLength    int        `json:"min_length,omitempty"`
	MaxLength    int        `json:"max_length,omitempty"`
}

// Message represents a Slack message.
type Message struct {
	Channel     string       `json:"channel"`
	Text        string       `json:"text,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
	ThreadTS    string       `json:"thread_ts,omitempty"`
	Mrkdwn      bool         `json:"mrkdwn,omitempty"`
	UnfurlLinks bool         `json:"unfurl_links,omitempty"`
	UnfurlMedia bool         `json:"unfurl_media,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	IconURL     string       `json:"icon_url,omitempty"`
	Username    string       `json:"username,omitempty"`
	AsUser      bool         `json:"as_user,omitempty"`
	LinkNames   bool         `json:"link_names,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Metadata    *Metadata    `json:"metadata,omitempty"`
}

// Attachment represents a message attachment.
type Attachment struct {
	Color      string   `json:"color,omitempty"`
	Fallback   string   `json:"fallback,omitempty"`
	Title      string   `json:"title,omitempty"`
	TitleLink  string   `json:"title_link,omitempty"`
	Text       string   `json:"text,omitempty"`
	Fields     []Field  `json:"fields,omitempty"`
	Footer     string   `json:"footer,omitempty"`
	FooterIcon string   `json:"footer_icon,omitempty"`
	Timestamp  int64    `json:"ts,omitempty"`
	Mrkdwn     bool     `json:"mrkdwn,omitempty"`
	MrkdwnIn   []string `json:"mrkdwn_in,omitempty"`
}

// Field represents an attachment field.
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short,omitempty"`
}

// Metadata represents message metadata.
type Metadata struct {
	EventType    string                 `json:"event_type"`
	EventPayload map[string]interface{} `json:"event_payload"`
}

// InteractionCallback represents a Slack interaction payload.
type InteractionCallback struct {
	Type        string                 `json:"type"`
	Token       string                 `json:"token"`
	CallbackID  string                 `json:"callback_id"`
	TriggerID   string                 `json:"trigger_id"`
	User        User                   `json:"user"`
	Team        Team                   `json:"team"`
	Channel     Channel                `json:"channel"`
	ResponseURL string                 `json:"response_url"`
	View        *View                  `json:"view,omitempty"`
	Actions     []Action               `json:"actions,omitempty"`
	Submission  map[string]interface{} `json:"submission,omitempty"`
}

// User represents a Slack user.
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	TeamID   string `json:"team_id"`
}

// Team represents a Slack team.
type Team struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Name   string `json:"name"`
}

// Channel represents a Slack channel.
type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// View represents a modal view in interactions.
type View struct {
	ID              string     `json:"id"`
	TeamID          string     `json:"team_id"`
	Type            string     `json:"type"`
	PrivateMetadata string     `json:"private_metadata"`
	CallbackID      string     `json:"callback_id"`
	State           *ViewState `json:"state"`
	Hash            string     `json:"hash"`
	Title           *TextBlock `json:"title"`
	Close           *TextBlock `json:"close"`
	Submit          *TextBlock `json:"submit"`
	Blocks          []Block    `json:"blocks"`
}

// ViewState represents the state of a view.
type ViewState struct {
	Values map[string]map[string]ViewStateValue `json:"values"`
}

// ViewStateValue represents a value in view state.
type ViewStateValue struct {
	Type            string   `json:"type"`
	Value           string   `json:"value,omitempty"`
	SelectedOption  *Option  `json:"selected_option,omitempty"`
	SelectedOptions []Option `json:"selected_options,omitempty"`
	SelectedDate    string   `json:"selected_date,omitempty"`
	SelectedTime    string   `json:"selected_time,omitempty"`
}

// Option represents a select option.
type Option struct {
	Text  *TextBlock `json:"text"`
	Value string     `json:"value"`
}

// Action represents an interactive action.
type Action struct {
	Type     string     `json:"type"`
	ActionID string     `json:"action_id"`
	BlockID  string     `json:"block_id"`
	Text     *TextBlock `json:"text,omitempty"`
	Value    string     `json:"value,omitempty"`
	Style    string     `json:"style,omitempty"`
	ActionTS string     `json:"action_ts"`
}

// SlashCommand represents a Slack slash command.
type SlashCommand struct {
	Token       string `json:"token"`
	TeamID      string `json:"team_id"`
	TeamDomain  string `json:"team_domain"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	UserID      string `json:"user_id"`
	UserName    string `json:"user_name"`
	Command     string `json:"command"`
	Text        string `json:"text"`
	ResponseURL string `json:"response_url"`
	TriggerID   string `json:"trigger_id"`
}

// Event represents a Slack event.
type Event struct {
	Type     string `json:"type"`
	EventTS  string `json:"event_ts"`
	User     string `json:"user,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Text     string `json:"text,omitempty"`
	TS       string `json:"ts,omitempty"`
	ThreadTS string `json:"thread_ts,omitempty"`
	Subtype  string `json:"subtype,omitempty"`
	BotID    string `json:"bot_id,omitempty"`
}

// EventWrapper wraps Slack events.
type EventWrapper struct {
	Token     string `json:"token"`
	TeamID    string `json:"team_id"`
	APIAppID  string `json:"api_app_id"`
	Event     Event  `json:"event"`
	Type      string `json:"type"`
	EventID   string `json:"event_id"`
	EventTime int64  `json:"event_time"`
	Challenge string `json:"challenge,omitempty"`
}

// ConversationInfo represents channel information.
type ConversationInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	IsChannel      bool   `json:"is_channel"`
	IsGroup        bool   `json:"is_group"`
	IsIM           bool   `json:"is_im"`
	IsMPIM         bool   `json:"is_mpim"`
	IsPrivate      bool   `json:"is_private"`
	Created        int64  `json:"created"`
	Creator        string `json:"creator"`
	IsArchived     bool   `json:"is_archived"`
	IsGeneral      bool   `json:"is_general"`
	Unlinked       int    `json:"unlinked"`
	NameNormalized string `json:"name_normalized"`
	IsShared       bool   `json:"is_shared"`
	IsMember       bool   `json:"is_member"`
	NumMembers     int    `json:"num_members"`
}

// UserInfo represents user information.
type UserInfo struct {
	ID       string      `json:"id"`
	TeamID   string      `json:"team_id"`
	Name     string      `json:"name"`
	Deleted  bool        `json:"deleted"`
	Color    string      `json:"color"`
	RealName string      `json:"real_name"`
	TZ       string      `json:"tz"`
	TZLabel  string      `json:"tz_label"`
	TZOffset int         `json:"tz_offset"`
	Profile  UserProfile `json:"profile"`
	IsAdmin  bool        `json:"is_admin"`
	IsOwner  bool        `json:"is_owner"`
	IsBot    bool        `json:"is_bot"`
	Updated  int64       `json:"updated"`
}

// UserProfile represents user profile information.
type UserProfile struct {
	Title                 string `json:"title"`
	Phone                 string `json:"phone"`
	Skype                 string `json:"skype"`
	RealName              string `json:"real_name"`
	RealNameNormalized    string `json:"real_name_normalized"`
	DisplayName           string `json:"display_name"`
	DisplayNameNormalized string `json:"display_name_normalized"`
	Email                 string `json:"email"`
	Image24               string `json:"image_24"`
	Image32               string `json:"image_32"`
	Image48               string `json:"image_48"`
	Image72               string `json:"image_72"`
	Image192              string `json:"image_192"`
	Image512              string `json:"image_512"`
	StatusText            string `json:"status_text"`
	StatusEmoji           string `json:"status_emoji"`
	StatusExpiration      int64  `json:"status_expiration"`
}

// StandupModalMetadata contains metadata for standup modals.
type StandupModalMetadata struct {
	ChannelID string    `json:"channel_id"`
	Date      string    `json:"date"`
	SessionID string    `json:"session_id"`
	UserID    string    `json:"user_id"`
	Timestamp time.Time `json:"timestamp"`
}
