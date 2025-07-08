package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/aws/aws-lambda-go/events"
	awslambda "github.com/aws/aws-lambda-go/lambda"

	botcontext "github.com/synaptiq/standup-bot/context"
	"github.com/synaptiq/standup-bot/internal/lambda"
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
	verifier    *slack.RequestVerifier
	handlerFunc lambda.Handler
)

func init() {
	// Initialize components
	ctx := context.Background()
	initConfig := lambda.DefaultInitConfig()

	var err error
	botCtx, dataStore, slackClient, err = lambda.Initialize(ctx, initConfig)
	if err != nil {
		log.Fatalf("Failed to initialize: %v", err)
	}

	// Create service
	service = standup.NewService(botCtx, dataStore, slackClient)

	// Create request verifier
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	if signingSecret == "" {
		log.Fatal("SLACK_SIGNING_SECRET not set")
	}
	verifier = slack.NewRequestVerifier(signingSecret)

	// Create handler with middleware
	handlerFunc = lambda.StandardMiddleware(botCtx)(handler)
}

func main() {
	awslambda.Start(handlerFunc)
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Verify Slack request
	timestamp := request.Headers["X-Slack-Request-Timestamp"]
	signature := request.Headers["X-Slack-Signature"]

	if err := verifier.VerifyRequest(timestamp, signature, request.Body); err != nil {
		return lambda.Unauthorized("Invalid request signature"), nil
	}

	// Handle URL verification challenge
	if request.Headers["Content-Type"] == "application/json" {
		var challenge struct {
			Type      string `json:"type"`
			Challenge string `json:"challenge"`
		}
		if err := json.Unmarshal([]byte(request.Body), &challenge); err == nil {
			if challenge.Type == "url_verification" {
				return lambda.OK(challenge.Challenge), nil
			}
		}
	}

	// Route based on content type
	contentType := request.Headers["Content-Type"]

	switch {
	case contentType == "application/x-www-form-urlencoded":
		// Slash command or interactive component
		values, err := url.ParseQuery(request.Body)
		if err != nil {
			return lambda.BadRequest("Invalid form data"), nil
		}

		if values.Get("command") != "" {
			return handleSlashCommand(ctx, values)
		} else if values.Get("payload") != "" {
			return handleInteraction(ctx, values.Get("payload"))
		}

	case contentType == "application/json":
		// Event subscription
		return handleEvent(ctx, request.Body)
	}

	return lambda.BadRequest("Unsupported request type"), nil
}

func handleSlashCommand(ctx context.Context, values url.Values) (events.APIGatewayProxyResponse, error) {
	cmd := slack.SlashCommand{
		Token:       values.Get("token"),
		TeamID:      values.Get("team_id"),
		TeamDomain:  values.Get("team_domain"),
		ChannelID:   values.Get("channel_id"),
		ChannelName: values.Get("channel_name"),
		UserID:      values.Get("user_id"),
		UserName:    values.Get("user_name"),
		Command:     values.Get("command"),
		Text:        values.Get("text"),
		ResponseURL: values.Get("response_url"),
		TriggerID:   values.Get("trigger_id"),
	}

	// Add user context
	ctx = botCtx.WithUserID(ctx, cmd.UserID)
	ctx = botCtx.WithChannelID(ctx, cmd.ChannelID)

	logger := botCtx.Logger()
	logger.Info(ctx, "Slash command received",
		botcontext.Field{Key: "command", Value: cmd.Command},
		botcontext.Field{Key: "text", Value: cmd.Text},
	)

	switch cmd.Command {
	case "/standup":
		return handleStandupCommand(ctx, cmd)
	case "/standup-config":
		return handleConfigCommand(ctx, cmd)
	case "/standup-report":
		return handleReportCommand(ctx, cmd)
	default:
		return lambda.SlackEphemeralResponse("Unknown command"), nil
	}
}

func handleStandupCommand(ctx context.Context, cmd slack.SlashCommand) (events.APIGatewayProxyResponse, error) {
	// Open standup modal
	if err := service.OpenStandupModal(ctx, cmd.TriggerID, cmd.ChannelID, cmd.UserID); err != nil {
		botCtx.Logger().Error(ctx, "Failed to open standup modal", err)
		return lambda.SlackEphemeralResponse("Failed to open standup form. Please try again."), nil
	}

	// Return empty response (modal will handle interaction)
	return lambda.OK(""), nil
}

func handleConfigCommand(ctx context.Context, cmd slack.SlashCommand) (events.APIGatewayProxyResponse, error) {
	// TODO: Implement configuration interface
	return lambda.SlackEphemeralResponse("Configuration interface coming soon!"), nil
}

func handleReportCommand(ctx context.Context, cmd slack.SlashCommand) (events.APIGatewayProxyResponse, error) {
	// TODO: Implement reporting interface
	return lambda.SlackEphemeralResponse("Reporting interface coming soon!"), nil
}

func handleInteraction(ctx context.Context, payloadStr string) (events.APIGatewayProxyResponse, error) {
	var payload slack.InteractionCallback
	if err := json.Unmarshal([]byte(payloadStr), &payload); err != nil {
		return lambda.BadRequest("Invalid interaction payload"), nil
	}

	// Add user context
	ctx = botCtx.WithUserID(ctx, payload.User.ID)
	if payload.Channel.ID != "" {
		ctx = botCtx.WithChannelID(ctx, payload.Channel.ID)
	}

	logger := botCtx.Logger()
	logger.Info(ctx, "Interaction received",
		botcontext.Field{Key: "type", Value: payload.Type},
		botcontext.Field{Key: "callback_id", Value: payload.CallbackID},
	)

	switch payload.Type {
	case "view_submission":
		return handleViewSubmission(ctx, &payload)
	case "block_actions":
		return handleBlockActions(ctx, &payload)
	case "view_closed":
		// Nothing to do
		return lambda.OK(""), nil
	default:
		return lambda.BadRequest("Unknown interaction type"), nil
	}
}

func handleViewSubmission(ctx context.Context, payload *slack.InteractionCallback) (events.APIGatewayProxyResponse, error) {
	if payload.View == nil {
		return lambda.BadRequest("Missing view data"), nil
	}

	switch payload.View.CallbackID {
	case "standup_submission":
		return handleStandupSubmission(ctx, payload)
	default:
		return lambda.BadRequest("Unknown view callback"), nil
	}
}

func handleStandupSubmission(ctx context.Context, payload *slack.InteractionCallback) (events.APIGatewayProxyResponse, error) {
	// Parse modal metadata
	metadata, err := slack.ParseModalMetadata(payload.View.PrivateMetadata)
	if err != nil {
		return lambda.BadRequest("Invalid modal metadata"), nil
	}

	// Parse responses
	responses, err := slack.ParseModalSubmission(payload.View)
	if err != nil {
		return lambda.BadRequest("Failed to parse submission"), nil
	}

	// Create submission
	submission := &standup.StandupSubmission{
		SessionID: metadata.SessionID,
		ChannelID: metadata.ChannelID,
		Date:      metadata.Date,
		UserID:    payload.User.ID,
		UserName:  payload.User.Name,
		Responses: responses,
	}

	// Submit response
	if err := service.SubmitStandupResponse(ctx, submission); err != nil {
		botCtx.Logger().Error(ctx, "Failed to submit standup", err)
		return lambda.InternalServerError("Failed to save your standup. Please try again."), nil
	}

	// Return success (closes modal)
	return lambda.OK(""), nil
}

func handleBlockActions(ctx context.Context, payload *slack.InteractionCallback) (events.APIGatewayProxyResponse, error) {
	// TODO: Handle block actions if needed
	return lambda.OK(""), nil
}

func handleEvent(ctx context.Context, body string) (events.APIGatewayProxyResponse, error) {
	var wrapper slack.EventWrapper
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return lambda.BadRequest("Invalid event payload"), nil
	}

	// Handle different event types
	switch wrapper.Type {
	case "event_callback":
		return handleEventCallback(ctx, &wrapper)
	case "app_rate_limited":
		botCtx.Logger().Warn(ctx, "Rate limited by Slack")
		return lambda.OK(""), nil
	default:
		return lambda.BadRequest(fmt.Sprintf("Unknown event type: %s", wrapper.Type)), nil
	}
}

func handleEventCallback(ctx context.Context, wrapper *slack.EventWrapper) (events.APIGatewayProxyResponse, error) {
	// Add context
	if wrapper.Event.User != "" {
		ctx = botCtx.WithUserID(ctx, wrapper.Event.User)
	}
	if wrapper.Event.Channel != "" {
		ctx = botCtx.WithChannelID(ctx, wrapper.Event.Channel)
	}

	logger := botCtx.Logger()
	logger.Info(ctx, "Event received",
		botcontext.Field{Key: "event_type", Value: wrapper.Event.Type},
	)

	// Handle specific events
	switch wrapper.Event.Type {
	case "app_mention":
		// TODO: Handle mentions
	case "message":
		// TODO: Handle DM responses
	}

	// Always return 200 OK for events
	return lambda.OK(""), nil
}
