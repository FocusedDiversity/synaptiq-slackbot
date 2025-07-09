package standup

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	botcontext "github.com/synaptiq/standup-bot/context"
	lambdautil "github.com/synaptiq/standup-bot/internal/lambda"
	"github.com/synaptiq/standup-bot/internal/slack"
	"github.com/synaptiq/standup-bot/internal/store"
)

// Service handles standup business logic.
type Service struct {
	botCtx      botcontext.BotContext
	store       store.Store
	slackClient slack.Client
}

// NewService creates a new standup service.
func NewService(botCtx botcontext.BotContext, store store.Store, slackClient slack.Client) *Service {
	return &Service{
		botCtx:      botCtx,
		store:       store,
		slackClient: slackClient,
	}
}

// StartStandupSession starts a new standup session for a channel.
func (s *Service) StartStandupSession(ctx context.Context, channelID string) (*store.Session, error) {
	logger := s.botCtx.Logger()
	today := time.Now().Format("2006-01-02")

	// Check if session already exists
	existingSession, err := s.store.GetSession(ctx, channelID, today)
	if err == nil && existingSession != nil {
		logger.Info(ctx, "Session already exists",
			botcontext.Field{Key: "channel_id", Value: channelID},
			botcontext.Field{Key: "date", Value: today},
		)
		return existingSession, nil
	}

	// Create new session
	session := &store.Session{
		SessionID:     uuid.New().String(),
		ChannelID:     channelID,
		Date:          today,
		Status:        store.SessionPending,
		SummaryPosted: false,
		CreatedAt:     time.Now(),
	}

	if err := s.store.CreateSession(ctx, session); err != nil {
		if err == store.ErrAlreadyExists {
			// Race condition - another process created the session
			return s.store.GetSession(ctx, channelID, today)
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	logger.Info(ctx, "Started new standup session",
		botcontext.Field{Key: "session_id", Value: session.SessionID},
		botcontext.Field{Key: "channel_id", Value: channelID},
	)

	return session, nil
}

// OpenStandupModal opens the standup submission modal for a user.
func (s *Service) OpenStandupModal(ctx context.Context, triggerID, channelID, userID string) error {
	cfg := s.botCtx.Config()
	channel, found := cfg.ChannelByID(channelID)
	if !found {
		return fmt.Errorf("channel not configured: %s", lambdautil.SanitizeLogValue(channelID))
	}

	if !channel.IsEnabled() {
		return fmt.Errorf("standups not enabled for channel %s", channelID)
	}

	// Ensure session exists
	session, err := s.StartStandupSession(ctx, channelID)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Build and open modal
	modal := slack.BuildStandupModal(channelID, session.SessionID, channel.Questions())
	if err := s.slackClient.OpenModal(ctx, triggerID, modal); err != nil {
		return fmt.Errorf("failed to open modal: %w", err)
	}

	return nil
}

// SubmitStandupResponse processes a standup submission from a user.
func (s *Service) SubmitStandupResponse(ctx context.Context, submission *Submission) error {
	logger := s.botCtx.Logger()

	// Create user response
	response := &store.UserResponse{
		SessionID:     submission.SessionID,
		ChannelID:     submission.ChannelID,
		Date:          submission.Date,
		UserID:        submission.UserID,
		UserName:      submission.UserName,
		Responses:     submission.Responses,
		SubmittedAt:   time.Now(),
		ReminderCount: 0,
	}

	if err := s.store.SaveUserResponse(ctx, response); err != nil {
		return fmt.Errorf("failed to save response: %w", err)
	}

	logger.Info(ctx, "Saved standup response",
		botcontext.Field{Key: "user_id", Value: submission.UserID},
		botcontext.Field{Key: "channel_id", Value: submission.ChannelID},
	)

	// Post to channel in thread if threading is enabled
	if s.botCtx.Config().IsFeatureEnabled("threading_enabled") {
		if err := s.postResponseToChannel(ctx, submission); err != nil {
			logger.Error(ctx, "Failed to post response to channel", err)
			// Don't fail the submission if posting fails
		}
	}

	return nil
}

// SendReminders sends reminders to users who haven't submitted.
func (s *Service) SendReminders(ctx context.Context, channelID, reminderTime string) error {
	logger := s.botCtx.Logger()
	today := time.Now().Format("2006-01-02")

	// Get channel configuration
	teamID := "" // TODO: Get from channel config or workspace lookup

	channelConfig, err := s.store.GetChannelConfig(ctx, teamID, channelID)
	if err != nil {
		return fmt.Errorf("failed to get channel config: %w", err)
	}

	if !channelConfig.Enabled {
		return nil // Skip disabled channels
	}

	// Get users without responses
	missingUsers, err := s.store.GetUsersWithoutResponse(ctx, channelID, today, channelConfig.Users)
	if err != nil {
		return fmt.Errorf("failed to get missing users: %w", err)
	}

	// Send reminders
	for _, userID := range missingUsers {
		if err := s.sendReminderToUser(ctx, userID, channelID, channelConfig.ChannelName, reminderTime); err != nil {
			logger.Error(ctx, "Failed to send reminder", err,
				botcontext.Field{Key: "user_id", Value: userID},
			)
			continue
		}
	}

	logger.Info(ctx, "Sent reminders",
		botcontext.Field{Key: "channel_id", Value: channelID},
		botcontext.Field{Key: "reminder_count", Value: len(missingUsers)},
	)

	return nil
}

// PostDailySummary posts the daily standup summary.
func (s *Service) PostDailySummary(ctx context.Context, channelID string) error {
	logger := s.botCtx.Logger()
	today := time.Now().Format("2006-01-02")

	// Get session
	session, err := s.store.GetSession(ctx, channelID, today)
	if err != nil && err != store.ErrNotFound {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if err == store.ErrNotFound {
		// No session today - create one
		session, err = s.StartStandupSession(ctx, channelID)
		if err != nil {
			return fmt.Errorf("failed to create session: %w", err)
		}
	}

	// Check if summary already posted
	if session.SummaryPosted {
		logger.Info(ctx, "Summary already posted",
			botcontext.Field{Key: "channel_id", Value: channelID},
		)
		return nil
	}

	// Get all responses
	responses, err := s.store.ListUserResponses(ctx, channelID, today)
	if err != nil {
		return fmt.Errorf("failed to list responses: %w", err)
	}

	// Get channel configuration
	cfg := s.botCtx.Config()
	channel, found := cfg.ChannelByID(channelID)
	if !found {
		return fmt.Errorf("channel not configured: %s", lambdautil.SanitizeLogValue(channelID))
	}

	// Build summary
	summaries := make([]*slack.UserResponseSummary, 0, len(channel.Users()))
	respondedUsers := make(map[string]bool)

	for _, resp := range responses {
		summaries = append(summaries, &slack.UserResponseSummary{
			UserID:    resp.UserID,
			UserName:  resp.UserName,
			Submitted: true,
			Time:      resp.SubmittedAt.Format("3:04 PM"),
		})
		respondedUsers[resp.UserID] = true
	}

	// Add missing users
	for _, user := range channel.Users() {
		if !respondedUsers[user.ID()] {
			summaries = append(summaries, &slack.UserResponseSummary{
				UserID:    user.ID(),
				UserName:  user.Name(),
				Submitted: false,
			})
		}
	}

	// Post summary
	blocks := slack.BuildSummaryMessage(today, channel.Templates().SummaryHeader(), summaries)
	_, err = s.slackClient.PostMessage(ctx, channelID, slack.WithBlocks(blocks...))
	if err != nil {
		return fmt.Errorf("failed to post summary: %w", err)
	}

	// Mark summary as posted
	if err := s.store.MarkSummaryPosted(ctx, channelID, today); err != nil {
		logger.Error(ctx, "Failed to mark summary posted", err)
		// Don't fail if we can't update the flag
	}

	// Update session status
	if err := s.store.UpdateSessionStatus(ctx, channelID, today, store.SessionCompleted); err != nil {
		logger.Error(ctx, "Failed to update session status", err)
	}

	logger.Info(ctx, "Posted daily summary",
		botcontext.Field{Key: "channel_id", Value: channelID},
		botcontext.Field{Key: "total_users", Value: len(summaries)},
		botcontext.Field{Key: "responded", Value: len(responses)},
	)

	return nil
}

// postResponseToChannel posts a user's response to the channel.
func (s *Service) postResponseToChannel(ctx context.Context, submission *Submission) error {
	cfg := s.botCtx.Config()
	channel, found := cfg.ChannelByID(submission.ChannelID)
	if !found {
		return fmt.Errorf("channel not configured")
	}

	// Build message
	builder := slack.NewMessageBuilder()
	builder.AddSection(fmt.Sprintf("*Standup Update from <@%s>*", submission.UserID))

	questions := channel.Questions()
	for i, question := range questions {
		answer := submission.Responses[fmt.Sprintf("question_%d", i)]
		if answer != "" {
			builder.AddSection(fmt.Sprintf("*%s*\n%s", question, answer))
		}
	}

	blocks := builder.Build()

	// Post to channel
	// TODO: Post in thread if there's a daily thread
	_, err := s.slackClient.PostMessage(ctx, submission.ChannelID, slack.WithBlocks(blocks...))
	return err
}

// sendReminderToUser sends a reminder DM to a user.
func (s *Service) sendReminderToUser(ctx context.Context, userID, channelID, channelName, reminderTime string) error {
	cfg := s.botCtx.Config()
	channel, found := cfg.ChannelByID(channelID)
	if !found {
		return fmt.Errorf("channel not configured")
	}

	// Get user info
	userInfo, err := s.slackClient.GetUserInfo(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	// Build reminder message
	blocks := slack.BuildReminderMessage(userInfo.Name, channelName, channel.Templates().Reminder())

	// Open DM and send message
	dmChannel, err := s.slackClient.OpenDM(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to open DM: %w", err)
	}

	msgTS, err := s.slackClient.PostMessage(ctx, dmChannel, slack.WithBlocks(blocks...))
	if err != nil {
		return fmt.Errorf("failed to send reminder: %w", err)
	}

	// Save reminder record
	reminder := &store.Reminder{
		ChannelID: channelID,
		Date:      time.Now().Format("2006-01-02"),
		UserID:    userID,
		Time:      reminderTime,
		SentAt:    time.Now(),
		MessageTS: msgTS,
	}

	if err := s.store.SaveReminder(ctx, reminder); err != nil {
		// Log but don't fail
		s.botCtx.Logger().Error(ctx, "Failed to save reminder record", err)
	}

	// Increment reminder count
	today := time.Now().Format("2006-01-02")
	if err := s.store.IncrementReminderCount(ctx, channelID, today, userID); err != nil {
		// Log but don't fail
		s.botCtx.Logger().Error(ctx, "Failed to increment reminder count", err)
	}

	return nil
}

// Submission represents a standup submission.
type Submission struct {
	SessionID string
	ChannelID string
	Date      string
	UserID    string
	UserName  string
	Responses map[string]string
}
