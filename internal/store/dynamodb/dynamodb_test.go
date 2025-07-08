package dynamodb

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDynamoDBClient is a mock implementation of the DynamoDB client
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) UpdateItem(ctx context.Context, params *dynamodb.UpdateItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.UpdateItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dynamodb.QueryOutput), args.Error(1)
}

func TestSaveWorkspaceConfig(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := NewDynamoDBStore(mockClient, "test-table", 30)

	config := &store.WorkspaceConfig{
		TeamID:      "T123456",
		TeamName:    "Test Team",
		BotToken:    "xoxb-test-token",
		AppToken:    "xapp-test-token",
		InstalledAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		return *input.TableName == "test-table" &&
			input.Item["PK"].(*types.AttributeValueMemberS).Value == "WORKSPACE#T123456" &&
			input.Item["SK"].(*types.AttributeValueMemberS).Value == "WORKSPACE#T123456"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := store.SaveWorkspaceConfig(context.Background(), config)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestGetWorkspaceConfig(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := NewDynamoDBStore(mockClient, "test-table", 30)

	t.Run("found", func(t *testing.T) {
		mockClient.On("GetItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.GetItemInput) bool {
			return *input.TableName == "test-table" &&
				input.Key["PK"].(*types.AttributeValueMemberS).Value == "WORKSPACE#T123456"
		})).Return(&dynamodb.GetItemOutput{
			Item: map[string]types.AttributeValue{
				"team_id":   &types.AttributeValueMemberS{Value: "T123456"},
				"team_name": &types.AttributeValueMemberS{Value: "Test Team"},
				"bot_token": &types.AttributeValueMemberS{Value: "xoxb-test-token"},
			},
		}, nil).Once()

		config, err := store.GetWorkspaceConfig(context.Background(), "T123456")
		assert.NoError(t, err)
		assert.Equal(t, "T123456", config.TeamID)
		assert.Equal(t, "Test Team", config.TeamName)
	})

	t.Run("not found", func(t *testing.T) {
		mockClient.On("GetItem", mock.Anything, mock.Anything).Return(&dynamodb.GetItemOutput{}, nil).Once()

		_, err := store.GetWorkspaceConfig(context.Background(), "T999999")
		assert.Equal(t, store.ErrNotFound, err)
	})
}

func TestSaveChannelConfig(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := NewDynamoDBStore(mockClient, "test-table", 30)

	config := &store.ChannelConfig{
		TeamID:      "T123456",
		ChannelID:   "C123456",
		ChannelName: "engineering-standup",
		Enabled:     true,
		Schedule: store.ScheduleConfig{
			Timezone:      "America/New_York",
			SummaryTime:   "09:00",
			ReminderTimes: []string{"08:30", "08:50"},
			ActiveDays:    []string{"Mon", "Tue", "Wed", "Thu", "Fri"},
		},
		Users:     []string{"U123", "U456"},
		Questions: []string{"What did you do?", "What will you do?"},
		UpdatedAt: time.Now(),
	}

	mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		return *input.TableName == "test-table" &&
			input.Item["PK"].(*types.AttributeValueMemberS).Value == "WORKSPACE#T123456" &&
			input.Item["SK"].(*types.AttributeValueMemberS).Value == "CONFIG#C123456" &&
			input.Item["GSI1PK"].(*types.AttributeValueMemberS).Value == "ACTIVE#true"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := store.SaveChannelConfig(context.Background(), config)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestCreateSession(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := NewDynamoDBStore(mockClient, "test-table", 30)

	session := &store.Session{
		SessionID:     "sess-123",
		ChannelID:     "C123456",
		Date:          "2024-01-15",
		Status:        store.SessionPending,
		SummaryPosted: false,
		CreatedAt:     time.Now(),
	}

	t.Run("success", func(t *testing.T) {
		mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
			return *input.TableName == "test-table" &&
				input.Item["PK"].(*types.AttributeValueMemberS).Value == "SESSION#C123456#2024-01-15" &&
				input.Item["SK"].(*types.AttributeValueMemberS).Value == "SESSION#C123456#2024-01-15" &&
				*input.ConditionExpression == "attribute_not_exists(PK)"
		})).Return(&dynamodb.PutItemOutput{}, nil).Once()

		err := store.CreateSession(context.Background(), session)
		assert.NoError(t, err)
	})

	t.Run("already exists", func(t *testing.T) {
		mockClient.On("PutItem", mock.Anything, mock.Anything).Return(nil, &types.ConditionalCheckFailedException{
			Message: aws.String("The conditional request failed"),
		}).Once()

		err := store.CreateSession(context.Background(), session)
		assert.Equal(t, store.ErrAlreadyExists, err)
	})
}

func TestSaveUserResponse(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := NewDynamoDBStore(mockClient, "test-table", 30)

	response := &store.UserResponse{
		SessionID: "sess-123",
		ChannelID: "C123456",
		Date:      "2024-01-15",
		UserID:    "U123456",
		UserName:  "alice",
		Responses: map[string]string{
			"q1": "Worked on feature X",
			"q2": "Will work on feature Y",
		},
		SubmittedAt:   time.Now(),
		ReminderCount: 0,
	}

	mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
		return *input.TableName == "test-table" &&
			input.Item["PK"].(*types.AttributeValueMemberS).Value == "SESSION#C123456#2024-01-15" &&
			input.Item["SK"].(*types.AttributeValueMemberS).Value == "USER#U123456"
	})).Return(&dynamodb.PutItemOutput{}, nil)

	err := store.SaveUserResponse(context.Background(), response)
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestGetUsersWithoutResponse(t *testing.T) {
	mockClient := new(MockDynamoDBClient)
	store := &DynamoDBStore{
		client:    mockClient,
		tableName: "test-table",
		ttlDays:   30,
	}

	// Mock ListUserResponses (Query)
	mockClient.On("Query", mock.Anything, mock.Anything).Return(&dynamodb.QueryOutput{
		Items: []map[string]types.AttributeValue{
			{
				"user_id": &types.AttributeValueMemberS{Value: "U123"},
			},
			{
				"user_id": &types.AttributeValueMemberS{Value: "U456"},
			},
		},
	}, nil)

	userIDs := []string{"U123", "U456", "U789", "U000"}
	missingUsers, err := store.GetUsersWithoutResponse(context.Background(), "C123456", "2024-01-15", userIDs)

	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"U789", "U000"}, missingUsers)
	mockClient.AssertExpectations(t)
}

func TestKeyGeneration(t *testing.T) {
	tests := []struct {
		name   string
		fn     func() (string, string)
		wantPK string
		wantSK string
	}{
		{
			name: "workspace key",
			fn: func() (string, string) {
				return workspaceKey("T123456")
			},
			wantPK: "WORKSPACE#T123456",
			wantSK: "WORKSPACE#T123456",
		},
		{
			name: "channel config key",
			fn: func() (string, string) {
				return channelConfigKey("T123456", "C789012")
			},
			wantPK: "WORKSPACE#T123456",
			wantSK: "CONFIG#C789012",
		},
		{
			name: "session key",
			fn: func() (string, string) {
				return sessionKey("C123456", "2024-01-15")
			},
			wantPK: "SESSION#C123456#2024-01-15",
			wantSK: "SESSION#C123456#2024-01-15",
		},
		{
			name: "user response key",
			fn: func() (string, string) {
				return userResponseKey("C123456", "2024-01-15", "U789012")
			},
			wantPK: "SESSION#C123456#2024-01-15",
			wantSK: "USER#U789012",
		},
		{
			name: "reminder key",
			fn: func() (string, string) {
				return reminderKey("C123456", "2024-01-15", "U789012", "08:30")
			},
			wantPK: "REMINDER#C123456#2024-01-15",
			wantSK: "USER#U789012#08:30",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pk, sk := tt.fn()
			assert.Equal(t, tt.wantPK, pk)
			assert.Equal(t, tt.wantSK, sk)
		})
	}
}

func TestCalculateTTL(t *testing.T) {
	store := &DynamoDBStore{ttlDays: 30}
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	ttl := store.calculateTTL(baseTime)
	assert.NotNil(t, ttl)

	expectedTTL := baseTime.AddDate(0, 0, 30).Unix()
	assert.Equal(t, expectedTTL, *ttl)

	// Test with zero TTL days
	store.ttlDays = 0
	ttl = store.calculateTTL(baseTime)
	assert.Nil(t, ttl)
}
