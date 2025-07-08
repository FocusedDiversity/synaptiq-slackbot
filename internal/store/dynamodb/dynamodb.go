package dynamodb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/synaptiq/standup-bot/internal/store"
)

// Store implements the Store interface using DynamoDB.
type Store struct {
	client    *dynamodb.Client
	tableName string
	ttlDays   int // TTL for old records in days
}

// NewStore creates a new DynamoDB store.
func NewStore(client *dynamodb.Client, tableName string, ttlDays int) store.Store {
	return &Store{
		client:    client,
		tableName: tableName,
		ttlDays:   ttlDays,
	}
}

// Helper functions for key generation.
func workspaceKey(teamID string) (pk, sk string) {
	return fmt.Sprintf("WORKSPACE#%s", teamID), fmt.Sprintf("WORKSPACE#%s", teamID)
}

func channelConfigKey(teamID, channelID string) (pk, sk string) {
	return fmt.Sprintf("WORKSPACE#%s", teamID), fmt.Sprintf("CONFIG#%s", channelID)
}

func sessionKey(channelID, date string) (pk, sk string) {
	return fmt.Sprintf("SESSION#%s#%s", channelID, date), fmt.Sprintf("SESSION#%s#%s", channelID, date)
}

func userResponseKey(channelID, date, userID string) (pk, sk string) {
	return fmt.Sprintf("SESSION#%s#%s", channelID, date), fmt.Sprintf("USER#%s", userID)
}

func reminderKey(channelID, date, userID, time string) (pk, sk string) {
	return fmt.Sprintf("REMINDER#%s#%s", channelID, date), fmt.Sprintf("USER#%s#%s", userID, time)
}

// calculateTTL calculates TTL timestamp for records.
func (s *Store) calculateTTL(baseTime time.Time) *int64 {
	if s.ttlDays <= 0 {
		return nil
	}
	ttl := baseTime.AddDate(0, 0, s.ttlDays).Unix()
	return &ttl
}

// SaveWorkspaceConfig saves workspace configuration.
func (s *Store) SaveWorkspaceConfig(ctx context.Context, config *store.WorkspaceConfig) error {
	pk, sk := workspaceKey(config.TeamID)

	item := map[string]interface{}{
		"PK":           pk,
		"SK":           sk,
		"team_id":      config.TeamID,
		"team_name":    config.TeamName,
		"bot_token":    config.BotToken,
		"app_token":    config.AppToken,
		"installed_at": config.InstalledAt,
		"updated_at":   time.Now(),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return &store.Error{Code: "MARSHAL_ERROR", Message: "Failed to marshal item", Err: err}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return &store.Error{Code: "PUT_ERROR", Message: "Failed to save workspace config", Err: err}
	}

	return nil
}

// GetWorkspaceConfig retrieves workspace configuration.
func (s *Store) GetWorkspaceConfig(ctx context.Context, teamID string) (*store.WorkspaceConfig, error) {
	pk, sk := workspaceKey(teamID)

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, &store.Error{Code: "GET_ERROR", Message: "Failed to get workspace config", Err: err}
	}

	if result.Item == nil {
		return nil, store.ErrNotFound
	}

	var config store.WorkspaceConfig
	if err := attributevalue.UnmarshalMap(result.Item, &config); err != nil {
		return nil, &store.Error{Code: "UNMARSHAL_ERROR", Message: "Failed to unmarshal item", Err: err}
	}

	return &config, nil
}

// SaveChannelConfig saves channel configuration.
func (s *Store) SaveChannelConfig(ctx context.Context, config *store.ChannelConfig) error {
	pk, sk := channelConfigKey(config.TeamID, config.ChannelID)

	item := map[string]interface{}{
		"PK":           pk,
		"SK":           sk,
		"team_id":      config.TeamID,
		"channel_id":   config.ChannelID,
		"channel_name": config.ChannelName,
		"enabled":      config.Enabled,
		"schedule":     config.Schedule,
		"users":        config.Users,
		"templates":    config.Templates,
		"questions":    config.Questions,
		"updated_at":   time.Now(),
		// GSI1 for querying active channels
		"GSI1PK": fmt.Sprintf("ACTIVE#%t", config.Enabled),
		"GSI1SK": fmt.Sprintf("CHANNEL#%s#%s", config.TeamID, config.ChannelID),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return &store.Error{Code: "MARSHAL_ERROR", Message: "Failed to marshal item", Err: err}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return &store.Error{Code: "PUT_ERROR", Message: "Failed to save channel config", Err: err}
	}

	return nil
}

// GetChannelConfig retrieves channel configuration.
func (s *Store) GetChannelConfig(ctx context.Context, teamID, channelID string) (*store.ChannelConfig, error) {
	pk, sk := channelConfigKey(teamID, channelID)

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, &store.Error{Code: "GET_ERROR", Message: "Failed to get channel config", Err: err}
	}

	if result.Item == nil {
		return nil, store.ErrNotFound
	}

	var config store.ChannelConfig
	if err := attributevalue.UnmarshalMap(result.Item, &config); err != nil {
		return nil, &store.Error{Code: "UNMARSHAL_ERROR", Message: "Failed to unmarshal item", Err: err}
	}

	return &config, nil
}

// ListChannelConfigs lists all channel configurations for a workspace.
func (s *Store) ListChannelConfigs(ctx context.Context, teamID string) ([]*store.ChannelConfig, error) {
	pk := fmt.Sprintf("WORKSPACE#%s", teamID)

	keyCond := expression.Key("PK").Equal(expression.Value(pk)).And(
		expression.Key("SK").BeginsWith("CONFIG#"),
	)

	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	var configs []*store.ChannelConfig
	paginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &store.Error{Code: "QUERY_ERROR", Message: "Failed to query channel configs", Err: err}
		}

		for _, item := range page.Items {
			var config store.ChannelConfig
			if err := attributevalue.UnmarshalMap(item, &config); err != nil {
				continue // Skip invalid items
			}
			configs = append(configs, &config)
		}
	}

	return configs, nil
}

// ListActiveChannelConfigs lists all active channel configurations across all workspaces.
func (s *Store) ListActiveChannelConfigs(ctx context.Context) ([]*store.ChannelConfig, error) {
	keyCond := expression.Key("GSI1PK").Equal(expression.Value("ACTIVE#true"))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	var configs []*store.ChannelConfig
	paginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		IndexName:                 aws.String("GSI1"),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &store.Error{Code: "QUERY_ERROR", Message: "Failed to query active configs", Err: err}
		}

		for _, item := range page.Items {
			var config store.ChannelConfig
			if err := attributevalue.UnmarshalMap(item, &config); err != nil {
				continue // Skip invalid items
			}
			configs = append(configs, &config)
		}
	}

	return configs, nil
}

// CreateSession creates a new standup session.
func (s *Store) CreateSession(ctx context.Context, session *store.Session) error {
	pk, sk := sessionKey(session.ChannelID, session.Date)

	item := map[string]interface{}{
		"PK":             pk,
		"SK":             sk,
		"session_id":     session.SessionID,
		"channel_id":     session.ChannelID,
		"date":           session.Date,
		"status":         session.Status,
		"summary_posted": session.SummaryPosted,
		"created_at":     session.CreatedAt,
		"TTL":            s.calculateTTL(session.CreatedAt),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return &store.Error{Code: "MARSHAL_ERROR", Message: "Failed to marshal item", Err: err}
	}

	// Use conditional put to avoid overwriting existing sessions
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName:           aws.String(s.tableName),
		Item:                av,
		ConditionExpression: aws.String("attribute_not_exists(PK)"),
	})
	if err != nil {
		var cfe *types.ConditionalCheckFailedException
		if errors.As(err, &cfe) {
			return store.ErrAlreadyExists
		}
		return &store.Error{Code: "PUT_ERROR", Message: "Failed to create session", Err: err}
	}

	return nil
}

// GetSession retrieves a standup session.
func (s *Store) GetSession(ctx context.Context, channelID, date string) (*store.Session, error) {
	pk, sk := sessionKey(channelID, date)

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, &store.Error{Code: "GET_ERROR", Message: "Failed to get session", Err: err}
	}

	if result.Item == nil {
		return nil, store.ErrNotFound
	}

	var session store.Session
	if err := attributevalue.UnmarshalMap(result.Item, &session); err != nil {
		return nil, &store.Error{Code: "UNMARSHAL_ERROR", Message: "Failed to unmarshal item", Err: err}
	}

	return &session, nil
}

// UpdateSessionStatus updates the status of a session.
func (s *Store) UpdateSessionStatus(
	ctx context.Context,
	channelID, date string,
	status store.SessionStatus,
) error {
	pk, sk := sessionKey(channelID, date)

	update := expression.Set(expression.Name("status"), expression.Value(status))
	if status == store.SessionCompleted {
		update = update.Set(expression.Name("completed_at"), expression.Value(time.Now()))
	}

	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return &store.Error{Code: "UPDATE_ERROR", Message: "Failed to update session status", Err: err}
	}

	return nil
}

// MarkSummaryPosted marks a session summary as posted.
func (s *Store) MarkSummaryPosted(ctx context.Context, channelID, date string) error {
	pk, sk := sessionKey(channelID, date)

	update := expression.Set(expression.Name("summary_posted"), expression.Value(true))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return &store.Error{Code: "UPDATE_ERROR", Message: "Failed to mark summary posted", Err: err}
	}

	return nil
}

// SaveUserResponse saves a user's standup response.
func (s *Store) SaveUserResponse(ctx context.Context, response *store.UserResponse) error {
	pk, sk := userResponseKey(response.ChannelID, response.Date, response.UserID)

	item := map[string]interface{}{
		"PK":             pk,
		"SK":             sk,
		"session_id":     response.SessionID,
		"channel_id":     response.ChannelID,
		"date":           response.Date,
		"user_id":        response.UserID,
		"user_name":      response.UserName,
		"responses":      response.Responses,
		"submitted_at":   response.SubmittedAt,
		"reminder_count": response.ReminderCount,
		"TTL":            s.calculateTTL(response.SubmittedAt),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return &store.Error{Code: "MARSHAL_ERROR", Message: "Failed to marshal item", Err: err}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return &store.Error{Code: "PUT_ERROR", Message: "Failed to save user response", Err: err}
	}

	return nil
}

// GetUserResponse retrieves a user's standup response.
func (s *Store) GetUserResponse(
	ctx context.Context,
	channelID, date, userID string,
) (*store.UserResponse, error) {
	pk, sk := userResponseKey(channelID, date, userID)

	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, &store.Error{Code: "GET_ERROR", Message: "Failed to get user response", Err: err}
	}

	if result.Item == nil {
		return nil, store.ErrNotFound
	}

	var response store.UserResponse
	if err := attributevalue.UnmarshalMap(result.Item, &response); err != nil {
		return nil, &store.Error{Code: "UNMARSHAL_ERROR", Message: "Failed to unmarshal item", Err: err}
	}

	return &response, nil
}

// ListUserResponses lists all user responses for a session.
func (s *Store) ListUserResponses(ctx context.Context, channelID, date string) ([]*store.UserResponse, error) {
	pk := fmt.Sprintf("SESSION#%s#%s", channelID, date)

	keyCond := expression.Key("PK").Equal(expression.Value(pk)).And(
		expression.Key("SK").BeginsWith("USER#"),
	)

	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	var responses []*store.UserResponse
	paginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &store.Error{Code: "QUERY_ERROR", Message: "Failed to query user responses", Err: err}
		}

		for _, item := range page.Items {
			var response store.UserResponse
			if err := attributevalue.UnmarshalMap(item, &response); err != nil {
				continue // Skip invalid items
			}
			responses = append(responses, &response)
		}
	}

	return responses, nil
}

// IncrementReminderCount increments the reminder count for a user.
func (s *Store) IncrementReminderCount(ctx context.Context, channelID, date, userID string) error {
	pk, sk := userResponseKey(channelID, date, userID)

	update := expression.Add(expression.Name("reminder_count"), expression.Value(1))
	expr, err := expression.NewBuilder().WithUpdate(update).Build()
	if err != nil {
		return &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	_, err = s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return &store.Error{Code: "UPDATE_ERROR", Message: "Failed to increment reminder count", Err: err}
	}

	return nil
}

// SaveReminder saves a reminder record.
func (s *Store) SaveReminder(ctx context.Context, reminder *store.Reminder) error {
	pk, sk := reminderKey(reminder.ChannelID, reminder.Date, reminder.UserID, reminder.Time)

	item := map[string]interface{}{
		"PK":         pk,
		"SK":         sk,
		"channel_id": reminder.ChannelID,
		"date":       reminder.Date,
		"user_id":    reminder.UserID,
		"time":       reminder.Time,
		"sent_at":    reminder.SentAt,
		"message_ts": reminder.MessageTS,
		"TTL":        s.calculateTTL(reminder.SentAt),
	}

	av, err := attributevalue.MarshalMap(item)
	if err != nil {
		return &store.Error{Code: "MARSHAL_ERROR", Message: "Failed to marshal item", Err: err}
	}

	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      av,
	})
	if err != nil {
		return &store.Error{Code: "PUT_ERROR", Message: "Failed to save reminder", Err: err}
	}

	return nil
}

// ListReminders lists all reminders for a channel and date.
func (s *Store) ListReminders(ctx context.Context, channelID, date string) ([]*store.Reminder, error) {
	pk := fmt.Sprintf("REMINDER#%s#%s", channelID, date)

	keyCond := expression.Key("PK").Equal(expression.Value(pk))

	expr, err := expression.NewBuilder().WithKeyCondition(keyCond).Build()
	if err != nil {
		return nil, &store.Error{Code: "EXPRESSION_ERROR", Message: "Failed to build expression", Err: err}
	}

	var reminders []*store.Reminder
	paginator := dynamodb.NewQueryPaginator(s.client, &dynamodb.QueryInput{
		TableName:                 aws.String(s.tableName),
		KeyConditionExpression:    expr.KeyCondition(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, &store.Error{Code: "QUERY_ERROR", Message: "Failed to query reminders", Err: err}
		}

		for _, item := range page.Items {
			var reminder store.Reminder
			if err := attributevalue.UnmarshalMap(item, &reminder); err != nil {
				continue // Skip invalid items
			}
			reminders = append(reminders, &reminder)
		}
	}

	return reminders, nil
}

// GetPendingSessions gets all sessions that need processing.
func (s *Store) GetPendingSessions(ctx context.Context, currentTime time.Time) ([]*store.Session, error) {
	// This would need a GSI on status to be efficient
	// For now, we'll need to scan or implement differently
	// In production, we'd create GSI2 with status as partition key
	return nil, &store.Error{Code: "NOT_IMPLEMENTED", Message: "GetPendingSessions not implemented"}
}

// GetUsersWithoutResponse gets users who haven't submitted responses.
func (s *Store) GetUsersWithoutResponse(
	ctx context.Context,
	channelID, date string,
	userIDs []string,
) ([]string, error) {
	// Get all responses for the session
	responses, err := s.ListUserResponses(ctx, channelID, date)
	if err != nil {
		return nil, err
	}

	// Create a map of users who have responded
	respondedUsers := make(map[string]bool)
	for _, resp := range responses {
		respondedUsers[resp.UserID] = true
	}

	// Find users who haven't responded
	var missingUsers []string
	for _, userID := range userIDs {
		if !respondedUsers[userID] {
			missingUsers = append(missingUsers, userID)
		}
	}

	return missingUsers, nil
}
