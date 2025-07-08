package standup

import (
	"context"
	"fmt"
	"time"

	botcontext "github.com/synaptiq/standup-bot/context"
	"github.com/synaptiq/standup-bot/internal/store"
)

// Scheduler handles scheduled standup tasks.
type Scheduler struct {
	service *Service
	botCtx  botcontext.BotContext
	store   store.Store
}

// NewScheduler creates a new scheduler.
func NewScheduler(service *Service, botCtx botcontext.BotContext, store store.Store) *Scheduler {
	return &Scheduler{
		service: service,
		botCtx:  botCtx,
		store:   store,
	}
}

// ProcessScheduledTasks processes tasks that need to run at the current time.
func (s *Scheduler) ProcessScheduledTasks(ctx context.Context) error {
	logger := s.botCtx.Logger()
	now := time.Now()

	// Get all active channel configurations
	configs, err := s.store.ListActiveChannelConfigs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active configs: %w", err)
	}

	logger.Info(ctx, "Processing scheduled tasks",
		botcontext.Field{Key: "config_count", Value: len(configs)},
		botcontext.Field{Key: "time", Value: now.Format("15:04")},
	)

	for _, config := range configs {
		// Skip if not an active day
		if !s.isActiveDay(config, now) {
			continue
		}

		// Get channel's local time
		channelTime := s.getChannelTime(config, now)

		// Process reminders
		if err := s.processReminders(ctx, config, channelTime); err != nil {
			logger.Error(ctx, "Failed to process reminders", err,
				botcontext.Field{Key: "channel_id", Value: config.ChannelID},
			)
		}

		// Process daily summary
		if err := s.processDailySummary(ctx, config, channelTime); err != nil {
			logger.Error(ctx, "Failed to process summary", err,
				botcontext.Field{Key: "channel_id", Value: config.ChannelID},
			)
		}
	}

	return nil
}

// isActiveDay checks if today is an active day for the channel.
func (s *Scheduler) isActiveDay(config *store.ChannelConfig, now time.Time) bool {
	// Convert to channel's timezone
	loc, err := time.LoadLocation(config.Schedule.Timezone)
	if err != nil {
		// Default to checking in UTC if timezone is invalid
		loc = time.UTC
	}

	channelTime := now.In(loc)
	weekday := channelTime.Weekday()

	// Check if today is in active days
	dayMap := map[string]time.Weekday{
		"Sun": time.Sunday, "Mon": time.Monday, "Tue": time.Tuesday,
		"Wed": time.Wednesday, "Thu": time.Thursday, "Fri": time.Friday,
		"Sat": time.Saturday,
	}

	for _, activeDay := range config.Schedule.ActiveDays {
		if day, ok := dayMap[activeDay]; ok && day == weekday {
			return true
		}
	}

	return false
}

// getChannelTime converts current time to channel's timezone.
func (s *Scheduler) getChannelTime(config *store.ChannelConfig, now time.Time) time.Time {
	loc, err := time.LoadLocation(config.Schedule.Timezone)
	if err != nil {
		// Default to UTC if timezone is invalid
		return now.UTC()
	}
	return now.In(loc)
}

// processReminders checks and sends reminders if it's time.
func (s *Scheduler) processReminders(ctx context.Context, config *store.ChannelConfig, channelTime time.Time) error {
	currentTimeStr := channelTime.Format("15:04")

	for _, reminderTime := range config.Schedule.ReminderTimes {
		if s.isTimeMatch(currentTimeStr, reminderTime) {
			// Check if we've already sent reminders for this time today
			today := channelTime.Format("2006-01-02")
			reminders, err := s.store.ListReminders(ctx, config.ChannelID, today)
			if err != nil {
				return fmt.Errorf("failed to list reminders: %w", err)
			}

			// Check if we've already sent reminders for this time
			alreadySent := false
			for _, reminder := range reminders {
				if reminder.Time == reminderTime {
					alreadySent = true
					break
				}
			}

			if !alreadySent {
				if err := s.service.SendReminders(ctx, config.ChannelID, reminderTime); err != nil {
					return fmt.Errorf("failed to send reminders: %w", err)
				}
			}
		}
	}

	return nil
}

// processDailySummary checks and posts summary if it's time.
func (s *Scheduler) processDailySummary(ctx context.Context, config *store.ChannelConfig, channelTime time.Time) error {
	currentTimeStr := channelTime.Format("15:04")

	if s.isTimeMatch(currentTimeStr, config.Schedule.SummaryTime) {
		// Check if summary already posted today
		today := channelTime.Format("2006-01-02")
		session, err := s.store.GetSession(ctx, config.ChannelID, today)
		if err != nil && err != store.ErrNotFound {
			return fmt.Errorf("failed to get session: %w", err)
		}

		// Post summary if not already posted
		if session == nil || !session.SummaryPosted {
			if err := s.service.PostDailySummary(ctx, config.ChannelID); err != nil {
				return fmt.Errorf("failed to post summary: %w", err)
			}
		}
	}

	return nil
}

// This allows for a 1-minute window to handle timing variations.
func (s *Scheduler) isTimeMatch(currentTime, scheduledTime string) bool {
	// Parse times
	current, err1 := time.Parse("15:04", currentTime)
	scheduled, err2 := time.Parse("15:04", scheduledTime)

	if err1 != nil || err2 != nil {
		return false
	}

	// Check if within 1-minute window
	diff := current.Sub(scheduled).Minutes()
	return diff >= 0 && diff < 1
}

// This can be called at the beginning of each day.
func (s *Scheduler) StartDailyStandups(ctx context.Context) error {
	logger := s.botCtx.Logger()

	// Get all active channel configurations
	configs, err := s.store.ListActiveChannelConfigs(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active configs: %w", err)
	}

	startedCount := 0
	for _, config := range configs {
		// Check if today is an active day
		if !s.isActiveDay(config, time.Now()) {
			continue
		}

		// Start session
		_, err := s.service.StartStandupSession(ctx, config.ChannelID)
		if err != nil {
			logger.Error(ctx, "Failed to start standup session", err,
				botcontext.Field{Key: "channel_id", Value: config.ChannelID},
			)
			continue
		}

		startedCount++
	}

	logger.Info(ctx, "Started daily standup sessions",
		botcontext.Field{Key: "started_count", Value: startedCount},
		botcontext.Field{Key: "total_configs", Value: len(configs)},
	)

	return nil
}
