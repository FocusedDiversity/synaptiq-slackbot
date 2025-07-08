package main

import (
	"context"
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
	scheduler   *standup.Scheduler
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

	// Create service and scheduler
	service = standup.NewService(botCtx, dataStore, slackClient)
	scheduler = standup.NewScheduler(service, botCtx, dataStore)
}

func main() {
	lambda.Start(handler)
}

// handler processes EventBridge scheduled events.
func handler(ctx context.Context, event *events.CloudWatchEvent) error {
	logger := botCtx.Logger()

	// Add request ID from event
	ctx = botCtx.WithRequestID(ctx, event.ID)

	logger.Info(ctx, "Scheduler triggered",
		botcontext.Field{Key: "event_source", Value: event.Source},
		botcontext.Field{Key: "event_detail_type", Value: event.DetailType},
	)

	// Start tracer
	tracer := botCtx.Tracer()
	ctx, done := tracer.StartSpan(ctx, "scheduler_handler")
	defer done()

	// Process scheduled tasks
	if err := scheduler.ProcessScheduledTasks(ctx); err != nil {
		logger.Error(ctx, "Failed to process scheduled tasks", err)
		return err
	}

	logger.Info(ctx, "Scheduler completed successfully")
	return nil
}
