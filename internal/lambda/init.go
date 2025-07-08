package lambda

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	botconfig "github.com/synaptiq/standup-bot/config"
	botcontext "github.com/synaptiq/standup-bot/context"
	"github.com/synaptiq/standup-bot/internal/slack"
	"github.com/synaptiq/standup-bot/internal/store"
	dynamodbstore "github.com/synaptiq/standup-bot/internal/store/dynamodb"
)

// InitConfig contains initialization configuration
type InitConfig struct {
	ConfigPath    string
	TableName     string
	TTLDays       int
	SlackTokenEnv string
}

// DefaultInitConfig returns default initialization config
func DefaultInitConfig() InitConfig {
	return InitConfig{
		ConfigPath:    os.Getenv("CONFIG_PATH"),
		TableName:     os.Getenv("DYNAMODB_TABLE"),
		TTLDays:       30,
		SlackTokenEnv: "SLACK_BOT_TOKEN",
	}
}

// Initialize initializes all components for Lambda
func Initialize(ctx context.Context, initCfg InitConfig) (botcontext.BotContext, store.Store, slack.Client, error) {
	// Load configuration
	if initCfg.ConfigPath == "" {
		initCfg.ConfigPath = "config.yaml"
	}

	provider := botconfig.NewYAMLProvider(initCfg.ConfigPath)
	cfg, err := provider.Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	validator := botconfig.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		return nil, nil, nil, fmt.Errorf("invalid config: %w", err)
	}

	// Load AWS configuration
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(awsCfg)

	// Create store
	if initCfg.TableName == "" {
		initCfg.TableName = cfg.DatabaseTable()
	}
	dataStore := dynamodbstore.NewDynamoDBStore(dynamoClient, initCfg.TableName, initCfg.TTLDays)

	// Create Slack client
	slackToken := os.Getenv(initCfg.SlackTokenEnv)
	if slackToken == "" {
		slackToken = cfg.BotToken()
	}
	slackClient := slack.NewClient(slackToken)

	// Create secrets client
	secretsClient := &awsSecretsClient{
		client: secretsmanager.NewFromConfig(awsCfg),
	}

	// Create bot context
	botCtx, err := botcontext.New(botcontext.Options{
		Config:         cfg,
		ConfigProvider: provider,
		DynamoDB:       &dynamoDBClient{store: dataStore},
		SecretsManager: secretsClient,
		SlackClient:    &slackClientWrapper{client: slackClient},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create bot context: %w", err)
	}

	return botCtx, dataStore, slackClient, nil
}

// dynamoDBClient wraps the store to implement botcontext.DynamoDBClient
type dynamoDBClient struct {
	store store.Store
}

func (c *dynamoDBClient) PutItem(ctx context.Context, item interface{}) error {
	// This is a simplified implementation
	// In practice, you'd need to handle different item types
	return fmt.Errorf("not implemented")
}

func (c *dynamoDBClient) GetItem(ctx context.Context, key interface{}, item interface{}) error {
	return fmt.Errorf("not implemented")
}

func (c *dynamoDBClient) Query(ctx context.Context, params interface{}) ([]interface{}, error) {
	return nil, fmt.Errorf("not implemented")
}

// slackClientWrapper wraps the slack client to implement botcontext.SlackClient
type slackClientWrapper struct {
	client slack.Client
}

func (w *slackClientWrapper) PostMessage(ctx context.Context, channelID string, message string) error {
	_, err := w.client.PostMessage(ctx, channelID, slack.WithText(message))
	return err
}

func (w *slackClientWrapper) SendDM(ctx context.Context, userID string, message string) error {
	dmChannel, err := w.client.OpenDM(ctx, userID)
	if err != nil {
		return err
	}

	_, err = w.client.PostMessage(ctx, dmChannel, slack.WithText(message))
	return err
}

func (w *slackClientWrapper) OpenModal(ctx context.Context, triggerID string, view interface{}) error {
	modal, ok := view.(*slack.Modal)
	if !ok {
		return fmt.Errorf("invalid modal type")
	}

	return w.client.OpenModal(ctx, triggerID, modal)
}

// awsSecretsClient implements botcontext.SecretsClient
type awsSecretsClient struct {
	client *secretsmanager.Client
}

func (c *awsSecretsClient) GetSecret(ctx context.Context, secretID string) (string, error) {
	result, err := c.client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: &secretID,
	})
	if err != nil {
		return "", err
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	return "", fmt.Errorf("secret value is not a string")
}
