package main

import (
	"context"
	"fmt"
	"log"
	"os"

	botconfig "github.com/synaptiq/standup-bot/config"
	botcontext "github.com/synaptiq/standup-bot/context"
)

func main() {
	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	provider := botconfig.NewYAMLProvider(configPath)
	cfg, err := provider.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Validate configuration
	validator := botconfig.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Create bot context
	botCtx, err := botcontext.New(botcontext.Options{
		Config:         cfg,
		ConfigProvider: provider,
		// In a real application, you would provide actual AWS clients here
		// DynamoDB:       dynamodb.New(session),
		// SecretsManager: secretsmanager.New(session),
		// SlackClient:    slack.New(cfg.BotToken()),
	})
	if err != nil {
		log.Fatalf("Failed to create bot context: %v", err)
	}

	// Example: Using the context in a request handler
	ctx := context.Background()
	handleStandupRequest(ctx, botCtx, "U1234567890", "C1234567890") // pragma: allowlist secret
}

func handleStandupRequest(ctx context.Context, botCtx botcontext.BotContext, userID, channelID string) {
	// Add request context
	ctx = botCtx.WithRequestID(ctx, generateRequestID())
	ctx = botCtx.WithUserID(ctx, userID)
	ctx = botCtx.WithChannelID(ctx, channelID)

	logger := botCtx.Logger()
	// Note: In production code, always sanitize user input before logging
	// Even though these are hardcoded here, we demonstrate best practices
	logger.Info(ctx, "Handling standup request",
		botcontext.Field{Key: "user_id", Value: userID},
		botcontext.Field{Key: "channel_id", Value: channelID},
	)

	// Get channel configuration
	cfg := botCtx.Config()
	channel, found := cfg.ChannelByID(channelID)
	if !found {
		logger.Warn(ctx, "Channel not configured",
			botcontext.Field{Key: "channel_id", Value: channelID},
		)
		return
	}

	if !channel.IsEnabled() {
		logger.Info(ctx, "Channel is disabled")
		return
	}

	// Check if user is required to submit standup
	if !channel.IsUserRequired(userID) {
		logger.Info(ctx, "User not required for standup")
		return
	}

	// Get user configuration
	user, found := channel.UserByID(userID)
	if !found {
		logger.Error(ctx, "User not found in channel config", nil)
		return
	}

	// Display standup questions
	fmt.Printf("\nStandup for %s in #%s:\n", user.Name(), channel.Name())
	fmt.Println("Questions:")
	for i, question := range channel.Questions() {
		fmt.Printf("%d. %s\n", i+1, question)
	}

	// Check if threading is enabled
	if cfg.IsFeatureEnabled("threading_enabled") {
		fmt.Println("\n(Responses will be posted in a thread)")
	}

	// In a real application, you would:
	// 1. Open a Slack modal with the questions
	// 2. Collect user responses
	// 3. Store in DynamoDB
	// 4. Post to channel if needed

	logger.Info(ctx, "Standup request handled successfully")
}

func generateRequestID() string {
	// In production, use a proper UUID generator
	return fmt.Sprintf("req-%d", os.Getpid())
}
