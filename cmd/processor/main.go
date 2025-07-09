package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	botcontext "github.com/synaptiq/standup-bot/context"
	lambdautil "github.com/synaptiq/standup-bot/internal/lambda"
	"github.com/synaptiq/standup-bot/internal/slack"
	"github.com/synaptiq/standup-bot/internal/standup"
	"github.com/synaptiq/standup-bot/internal/store"
)

var (
	// Global instances initialized in init().
	botCtx      botcontext.BotContext
	dataStore   store.Store
	slackClient slack.Client
	service     *standup.Service
)

func init() {
	// Initialize components
	ctx := context.Background()
	initConfig := lambdautil.DefaultInitConfig()

	var err error
	botCtx, dataStore, slackClient, err = lambdautil.Initialize(ctx, initConfig)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Create service
	service = standup.NewService(botCtx, dataStore, slackClient)
}

func main() {
	lambda.Start(handler)
}

// TaskMessage represents an async task to process.
type TaskMessage struct {
	Type      string                 `json:"type"`
	ChannelID string                 `json:"channel_id"`
	UserID    string                 `json:"user_id"`
	Payload   map[string]interface{} `json:"payload"`
}

// handler processes SQS messages for async tasks.
func handler(ctx context.Context, event events.SQSEvent) error {
	logger := botCtx.Logger()

	// Process each message
	for i := range event.Records {
		record := &event.Records[i]
		// Add message ID as request ID
		ctx := botCtx.WithRequestID(ctx, record.MessageId)

		logger.Info(ctx, "Processing message",
			botcontext.Field{Key: "message_id", Value: record.MessageId},
			botcontext.Field{Key: "event_source", Value: record.EventSource},
		)

		// Start tracer
		tracer := botCtx.Tracer()
		msgCtx, done := tracer.StartSpan(ctx, "process_message")

		// Process the message
		if err := processMessage(msgCtx, record.Body); err != nil {
			logger.Error(msgCtx, "Failed to process message", err)
			done()
			// Return error to retry the message
			return err
		}

		done()
	}

	return nil
}

func processMessage(ctx context.Context, body string) error {
	var task TaskMessage
	if err := json.Unmarshal([]byte(body), &task); err != nil {
		// Bad message format - don't retry
		botCtx.Logger().Error(ctx, "Invalid message format", err,
			botcontext.Field{Key: "body", Value: lambdautil.SanitizeLogValue(body)},
		)
		return nil
	}

	// Add context
	if task.UserID != "" {
		ctx = botCtx.WithUserID(ctx, task.UserID)
	}
	if task.ChannelID != "" {
		ctx = botCtx.WithChannelID(ctx, task.ChannelID)
	}

	logger := botCtx.Logger()
	logger.Info(ctx, "Processing task",
		botcontext.Field{Key: "task_type", Value: lambdautil.SanitizeLogValue(task.Type)},
	)

	switch task.Type {
	case "send_welcome":
		return processSendWelcome(ctx, task)
	case "generate_report":
		return processGenerateReport(ctx, task)
	case "bulk_reminder":
		return processBulkReminder(ctx, task)
	default:
		logger.Warn(ctx, "Unknown task type",
			botcontext.Field{Key: "task_type", Value: lambdautil.SanitizeLogValue(task.Type)},
		)
		// Don't retry unknown task types
		return nil
	}
}

func processSendWelcome(ctx context.Context, task TaskMessage) error {
	userID := task.UserID
	channelID := task.ChannelID

	if userID == "" || channelID == "" {
		return fmt.Errorf("missing required fields for welcome message")
	}

	// Get channel configuration
	cfg := botCtx.Config()
	channel, found := cfg.ChannelByID(channelID)
	if !found {
		return fmt.Errorf("channel %s not configured", lambdautil.SanitizeLogValue(channelID))
	}

	// Build welcome message
	blocks := slack.NewMessageBuilder().
		AddHeader("Welcome to Daily Standups! 👋").
		AddSection(fmt.Sprintf("You've been added to the daily standup for <#%s>.", channelID)).
		AddSection("*How it works:*\n" +
			"• You'll receive a DM reminder each morning\n" +
			"• Use `/standup` to submit your update\n" +
			"• A summary is posted to the channel at the end").
		AddSection(fmt.Sprintf("*Schedule:*\n"+
			"• Reminders: %v\n"+
			"• Summary: %s",
			channel.ReminderTimes(),
			channel.SummaryTime().Format("3:04 PM"))).
		Build()

	// Send welcome DM
	dmChannel, err := slackClient.OpenDM(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to open DM: %w", err)
	}

	_, err = slackClient.PostMessage(ctx, dmChannel, slack.WithBlocks(blocks...))
	if err != nil {
		return fmt.Errorf("failed to send welcome message: %w", err)
	}

	botCtx.Logger().Info(ctx, "Sent welcome message",
		botcontext.Field{Key: "user_id", Value: lambdautil.SanitizeLogValue(userID)},
		botcontext.Field{Key: "channel_id", Value: lambdautil.SanitizeLogValue(channelID)},
	)

	return nil
}

func processGenerateReport(ctx context.Context, task TaskMessage) error {
	channelID := task.ChannelID
	if channelID == "" {
		return fmt.Errorf("missing channel ID for report")
	}

	// Get report parameters
	startDate, ok := task.Payload["start_date"].(string)
	if !ok {
		startDate = ""
	}
	endDate, _ := task.Payload["end_date"].(string)       //nolint:errcheck // optional parameter
	reportType, _ := task.Payload["report_type"].(string) //nolint:errcheck // optional parameter

	// TODO: Implement report generation
	// This would:
	// 1. Query historical data from DynamoDB
	// 2. Generate report (CSV, charts, etc.)
	// 3. Upload to S3 or send via Slack

	botCtx.Logger().Info(ctx, "Report generation not yet implemented",
		botcontext.Field{Key: "channel_id", Value: lambdautil.SanitizeLogValue(channelID)},
		botcontext.Field{Key: "report_type", Value: lambdautil.SanitizeLogValue(reportType)},
		botcontext.Field{Key: "start_date", Value: lambdautil.SanitizeLogValue(startDate)},
		botcontext.Field{Key: "end_date", Value: lambdautil.SanitizeLogValue(endDate)},
	)

	return nil
}

func processBulkReminder(ctx context.Context, task TaskMessage) error {
	channelID := task.ChannelID
	if channelID == "" {
		return fmt.Errorf("missing channel ID for bulk reminder")
	}

	// Get reminder time
	reminderTime, _ := task.Payload["reminder_time"].(string) //nolint:errcheck // optional parameter

	// Send reminders
	if err := service.SendReminders(ctx, channelID, reminderTime); err != nil {
		return fmt.Errorf("failed to send bulk reminders: %w", err)
	}

	return nil
}

// SendAsyncTask sends a task to the processor queue.
//
//nolint:unparam // ctx will be used when SQS implementation is added
func SendAsyncTask(ctx context.Context, taskType, channelID, userID string, payload map[string]interface{}) error {
	task := TaskMessage{
		Type:      taskType,
		ChannelID: channelID,
		UserID:    userID,
		Payload:   payload,
	}

	// TODO: Send to SQS queue
	// This would use AWS SDK to send the message to the processor queue
	_ = task // Temporarily suppress unused variable warning

	return nil
}
