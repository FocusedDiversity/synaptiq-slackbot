package slack

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/synaptiq/standup-bot/internal/security"
)

// ModalBuilder helps build Slack modals.
type ModalBuilder struct {
	modal *Modal
}

// NewModalBuilder creates a new modal builder.
func NewModalBuilder(title, callbackID string) *ModalBuilder {
	return &ModalBuilder{
		modal: &Modal{
			Type:       "modal",
			CallbackID: callbackID,
			Title: &TextBlock{
				Type: "plain_text",
				Text: title,
			},
			Blocks: []Block{},
		},
	}
}

// SetSubmit sets the submit button text.
func (b *ModalBuilder) SetSubmit(text string) *ModalBuilder {
	b.modal.Submit = &TextBlock{
		Type: "plain_text",
		Text: text,
	}
	return b
}

// SetClose sets the close button text.
func (b *ModalBuilder) SetClose(text string) *ModalBuilder {
	b.modal.Close = &TextBlock{
		Type: "plain_text",
		Text: text,
	}
	return b
}

// SetPrivateMetadata sets private metadata.
func (b *ModalBuilder) SetPrivateMetadata(metadata interface{}) *ModalBuilder {
	data, err := json.Marshal(metadata)
	if err != nil {
		return b
	}
	b.modal.PrivateMetadata = string(data)
	return b
}

// AddHeader adds a header block.
func (b *ModalBuilder) AddHeader(text string) *ModalBuilder {
	b.modal.Blocks = append(b.modal.Blocks, HeaderBlock{
		Type: "header",
		Text: &TextBlock{
			Type: "plain_text",
			Text: text,
		},
	})
	return b
}

// AddSection adds a section block.
func (b *ModalBuilder) AddSection(text string) *ModalBuilder {
	b.modal.Blocks = append(b.modal.Blocks, &SectionBlock{
		Type: "section",
		Text: &TextBlock{
			Type: "mrkdwn",
			Text: text,
		},
	})
	return b
}

// AddTextInput adds a text input block.
func (b *ModalBuilder) AddTextInput(blockID, actionID, label, placeholder string, multiline bool) *ModalBuilder {
	input := InputBlock{
		Type:    "input",
		BlockID: blockID,
		Label: &TextBlock{
			Type: "plain_text",
			Text: label,
		},
		Element: PlainTextInputElement{
			Type:      "plain_text_input",
			ActionID:  actionID,
			Multiline: multiline,
		},
	}

	if placeholder != "" {
		if element, ok := input.Element.(PlainTextInputElement); ok {
			element.Placeholder = &TextBlock{
				Type: "plain_text",
				Text: placeholder,
			}
			input.Element = element
		}
	}

	b.modal.Blocks = append(b.modal.Blocks, input)
	return b
}

// Build returns the built modal.
func (b *ModalBuilder) Build() *Modal {
	return b.modal
}

// MessageBuilder helps build Slack messages.
type MessageBuilder struct {
	blocks []Block
}

// NewMessageBuilder creates a new message builder.
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		blocks: []Block{},
	}
}

// AddHeader adds a header to the message.
func (b *MessageBuilder) AddHeader(text string) *MessageBuilder {
	b.blocks = append(b.blocks, HeaderBlock{
		Type: "header",
		Text: &TextBlock{
			Type:  "plain_text",
			Text:  text,
			Emoji: true,
		},
	})
	return b
}

// AddSection adds a section to the message.
func (b *MessageBuilder) AddSection(text string) *MessageBuilder {
	b.blocks = append(b.blocks, &SectionBlock{
		Type: "section",
		Text: &TextBlock{
			Type: "mrkdwn",
			Text: text,
		},
	})
	return b
}

// AddFields adds fields to the last section.
func (b *MessageBuilder) AddFields(fields ...string) *MessageBuilder {
	if len(b.blocks) == 0 || len(fields)%2 != 0 {
		return b
	}

	// Find the last section block
	for i := len(b.blocks) - 1; i >= 0; i-- {
		if section, ok := b.blocks[i].(*SectionBlock); ok {
			for j := 0; j < len(fields); j += 2 {
				section.Fields = append(section.Fields, TextBlock{
					Type: "mrkdwn",
					Text: fmt.Sprintf("*%s*\n%s", fields[j], fields[j+1]),
				})
			}
			// No need to reassign as we're modifying the pointer
			break
		}
	}

	return b
}

// AddDivider adds a divider block.
func (b *MessageBuilder) AddDivider() *MessageBuilder {
	b.blocks = append(b.blocks, DividerBlock{Type: "divider"})
	return b
}

// Build returns the built blocks.
func (b *MessageBuilder) Build() []Block {
	return b.blocks
}

// BuildStandupModal builds a standup submission modal.
func BuildStandupModal(channelID, sessionID string, questions []string) *Modal {
	metadata := StandupModalMetadata{
		ChannelID: channelID,
		SessionID: sessionID,
		Date:      time.Now().Format("2006-01-02"),
		Timestamp: time.Now(),
	}

	builder := NewModalBuilder("Daily Standup", "standup_submission").
		SetSubmit("Submit").
		SetPrivateMetadata(metadata).
		AddHeader("📝 Daily Standup Update").
		AddSection("Please answer the following questions:")

	// Add input for each question
	for i, question := range questions {
		blockID := fmt.Sprintf("question_%d", i)
		actionID := fmt.Sprintf("answer_%d", i)
		builder.AddTextInput(blockID, actionID, question, "Type your answer here...", true)
	}

	return builder.Build()
}

// BuildReminderMessage builds a reminder message.
func BuildReminderMessage(userName, channelName, template string) []Block {
	// Replace template variables
	text := strings.ReplaceAll(template, "{{.UserName}}", userName)
	text = strings.ReplaceAll(text, "{{.ChannelName}}", channelName)

	return NewMessageBuilder().
		AddSection(text).
		Build()
}

// BuildSummaryMessage builds a daily summary message.
func BuildSummaryMessage(date, headerTemplate string, responses []*UserResponseSummary) []Block {
	// Replace template variables
	header := strings.ReplaceAll(headerTemplate, "{{.Date}}", date)

	builder := NewMessageBuilder().
		AddHeader(header)

	// Add submitted users
	if len(responses) == 0 {
		builder.AddSection("No responses yet today.")
		return builder.Build()
	}

	var submitted []string
	var missing []string

	for _, resp := range responses {
		if resp.Submitted {
			userID := security.SanitizeLogValue(resp.UserID)
			submitted = append(submitted, fmt.Sprintf("• <@%s> - %s", userID, resp.Time))
		} else {
			missing = append(missing, fmt.Sprintf("• <@%s>", security.SanitizeLogValue(resp.UserID)))
		}
	}

	if len(submitted) > 0 {
		builder.AddSection("✅ *Submitted:*\n" + strings.Join(submitted, "\n"))
	}

	if len(missing) > 0 {
		builder.AddDivider()
		builder.AddSection("⏳ *Pending:*\n" + strings.Join(missing, "\n"))
	}

	return builder.Build()
}

// UserResponseSummary contains summary info for a user's response.
type UserResponseSummary struct {
	UserID    string
	UserName  string
	Submitted bool
	Time      string
}

// ParseModalSubmission parses the submission data from a modal.
func ParseModalSubmission(view *View) (map[string]string, error) {
	if view == nil || view.State == nil {
		return nil, fmt.Errorf("invalid view state")
	}

	responses := make(map[string]string)

	for blockID, actions := range view.State.Values {
		for _, value := range actions {
			if value.Type == "plain_text_input" {
				// Extract question number from block ID
				if strings.HasPrefix(blockID, "question_") {
					responses[blockID] = value.Value
				}
			}
		}
	}

	return responses, nil
}

// ParseModalMetadata parses the private metadata from a modal.
func ParseModalMetadata(privateMetadata string) (*StandupModalMetadata, error) {
	var metadata StandupModalMetadata
	if err := json.Unmarshal([]byte(privateMetadata), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return &metadata, nil
}
