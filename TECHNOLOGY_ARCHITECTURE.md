# Technology Architecture - Slack Stand-up Bot

## Executive Summary

This document outlines a serverless, scale-to-zero architecture for a Slack stand-up bot using Go, AWS SAM, and Lambda, ensuring zero costs during idle periods while maintaining instant scalability.

## Recommended Stack: AWS SAM + Lambda

### Core Architecture
**Go + AWS Lambda + API Gateway + DynamoDB**

> **Decision**: We will use DynamoDB as our primary database to achieve true scale-to-zero costs. If complex reporting requirements emerge that cannot be efficiently handled with DynamoDB, we will evaluate migrating to Aurora Serverless v2 at that time.

- **Rationale**: True scale-to-zero with pay-per-use pricing
- **Benefits**: 
  - Zero cost when not in use (with DynamoDB)
  - Automatic scaling to handle any load
  - No infrastructure management
  - Built-in high availability
  - AWS Free Tier generous limits

### Database Options Comparison

| Feature | DynamoDB | Aurora Serverless v2 PostgreSQL |
|---------|----------|--------------------------------|
| **Scale to Zero** | ✅ True $0 when idle | ❌ Min ~$43/month (0.5 ACU) |
| **Cold Start** | ✅ None | ⚠️ 5-30 seconds from paused |
| **Query Flexibility** | ❌ Limited (NoSQL) | ✅ Full SQL with joins |
| **Learning Curve** | ⚠️ NoSQL patterns | ✅ Familiar SQL |
| **Lambda Integration** | ✅ Direct SDK calls | ⚠️ Requires VPC + RDS Proxy |
| **Reporting/Analytics** | ❌ Complex aggregations | ✅ Native SQL aggregations |
| **Go Libraries** | ✅ AWS SDK | ✅ pgx, sqlx, GORM |
| **Best For** | Real-time operations | Complex queries & reporting |

### Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Slack API     │────▶│  API Gateway    │────▶│ Lambda Functions│
│   (Webhooks)    │     │   (REST API)    │     │   (Go Runtime)  │
└─────────────────┘     └─────────────────┘     └────────┬────────┘
                                                          │
                              ┌───────────────────────────┴────────────────────┐
                              │                                                │
                              ▼                                                ▼
                    ┌─────────────────┐                              ┌─────────────────┐
                    │   DynamoDB      │                              │  EventBridge    │
                    │ (NoSQL Store)   │                              │  (Scheduling)   │
                    └─────────────────┘                              └─────────────────┘
```

## Project Structure

```
standup-bot/
├── cmd/
│   ├── webhook/
│   │   └── main.go             # Slack webhook handler
│   ├── scheduler/
│   │   └── main.go             # Scheduled standup trigger
│   └── processor/
│       └── main.go             # Async processing handler
├── internal/
│   ├── slack/
│   │   ├── client.go           # Slack API client
│   │   ├── blocks.go           # Block Kit builders
│   │   ├── verify.go           # Request verification
│   │   └── handlers.go         # Event/command handlers
│   ├── standup/
│   │   ├── service.go          # Core business logic
│   │   ├── questions.go        # Question management
│   │   └── reporter.go         # Summary generation
│   ├── store/
│   │   ├── dynamodb.go         # DynamoDB client
│   │   └── models.go           # Data models
│   └── lambda/
│       └── middleware.go       # Lambda middleware
├── template.yaml               # SAM template
├── samconfig.toml             # SAM configuration
├── Makefile                   # Build automation
└── go.mod
```

## SAM Template

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
Description: Slack Stand-up Bot

Globals:
  Function:
    Runtime: provided.al2
    Architectures:
      - x86_64
    Environment:
      Variables:
        SLACK_BOT_TOKEN: !Ref SlackBotToken
        SLACK_SIGNING_SECRET: !Ref SlackSigningSecret
        TABLE_NAME: !Ref StandupTable
    Timeout: 30
    MemorySize: 256

Parameters:
  SlackBotToken:
    Type: String
    NoEcho: true
  SlackSigningSecret:
    Type: String
    NoEcho: true

Resources:
  # API Gateway
  SlackApi:
    Type: AWS::Serverless::Api
    Properties:
      StageName: prod
      Cors:
        AllowMethods: "'POST'"
        AllowHeaders: "'Content-Type,X-Slack-Signature,X-Slack-Request-Timestamp'"
        AllowOrigin: "'https://slack.com'"

  # Webhook Handler Lambda
  WebhookFunction:
    Type: AWS::Serverless::Function
    Metadata:
      BuildMethod: makefile
    Properties:
      Handler: bootstrap
      CodeUri: cmd/webhook/
      Events:
        SlackWebhook:
          Type: Api
          Properties:
            RestApiId: !Ref SlackApi
            Path: /slack/events
            Method: POST
      Policies:
        - DynamoDBCrudPolicy:
            TableName: !Ref StandupTable
        - LambdaInvokeFunction:
            FunctionName: !Ref ProcessorFunction

  # Async Processor Lambda
  ProcessorFunction:
    Type: AWS::Serverless::Function
    Metadata:
      BuildMethod: makefile
    Properties:
      Handler: bootstrap
      CodeUri: cmd/processor/
      ReservedConcurrentExecutions: 10
      Policies:
        - DynamoDBCrudPolicy:
            TableName: !Ref StandupTable

  # Scheduled Trigger Lambda
  SchedulerFunction:
    Type: AWS::Serverless::Function
    Metadata:
      BuildMethod: makefile
    Properties:
      Handler: bootstrap
      CodeUri: cmd/scheduler/
      Events:
        EveryMinute:
          Type: Schedule
          Properties:
            Schedule: rate(1 minute)
            Enabled: true
      Policies:
        - DynamoDBCrudPolicy:
            TableName: !Ref StandupTable
        - LambdaInvokeFunction:
            FunctionName: !Ref ProcessorFunction

  # DynamoDB Tables
  StandupTable:
    Type: AWS::DynamoDB::Table
    Properties:
      BillingMode: PAY_PER_REQUEST
      AttributeDefinitions:
        - AttributeName: PK
          AttributeType: S
        - AttributeName: SK
          AttributeType: S
        - AttributeName: GSI1PK
          AttributeType: S
        - AttributeName: GSI1SK
          AttributeType: S
      KeySchema:
        - AttributeName: PK
          KeyType: HASH
        - AttributeName: SK
          KeyType: RANGE
      GlobalSecondaryIndexes:
        - IndexName: GSI1
          KeySchema:
            - AttributeName: GSI1PK
              KeyType: HASH
            - AttributeName: GSI1SK
              KeyType: RANGE
          Projection:
            ProjectionType: ALL
      TimeToLiveSpecification:
        AttributeName: TTL
        Enabled: true

Outputs:
  WebhookUrl:
    Description: Slack Webhook URL
    Value: !Sub "${SlackApi}.execute-api.${AWS::Region}.amazonaws.com/prod/slack/events"
```

### Alternative SAM Template with Aurora Serverless v2

```yaml
# Additional resources for Aurora Serverless v2 option
Resources:
  # VPC for Aurora (required)
  VPC:
    Type: AWS::EC2::VPC
    Properties:
      CidrBlock: 10.0.0.0/16
      EnableDnsHostnames: true
      EnableDnsSupport: true

  # Private subnets for Lambda and Aurora
  PrivateSubnetA:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      AvailabilityZone: !Select [0, !GetAZs '']
      CidrBlock: 10.0.1.0/24

  PrivateSubnetB:
    Type: AWS::EC2::Subnet
    Properties:
      VpcId: !Ref VPC
      AvailabilityZone: !Select [1, !GetAZs '']
      CidrBlock: 10.0.2.0/24

  # Aurora Serverless v2 Cluster
  AuroraCluster:
    Type: AWS::RDS::DBCluster
    Properties:
      Engine: aurora-postgresql
      EngineMode: provisioned
      EngineVersion: '15.4'
      DatabaseName: standup
      MasterUsername: postgres
      MasterUserPassword: !Ref DBPassword
      ServerlessV2ScalingConfiguration:
        MinCapacity: 0.5
        MaxCapacity: 1
      DBSubnetGroupName: !Ref DBSubnetGroup
      VpcSecurityGroupIds:
        - !Ref DatabaseSecurityGroup

  AuroraInstance:
    Type: AWS::RDS::DBInstance
    Properties:
      DBClusterIdentifier: !Ref AuroraCluster
      DBInstanceClass: db.serverless
      Engine: aurora-postgresql

  # RDS Proxy for connection pooling
  DBProxy:
    Type: AWS::RDS::DBProxy
    Properties:
      DBProxyName: standup-proxy
      EngineFamily: POSTGRESQL
      Auth:
        - SecretArn: !Ref DBSecret
      RoleArn: !GetAtt DBProxyRole.Arn
      VpcSubnetIds:
        - !Ref PrivateSubnetA
        - !Ref PrivateSubnetB
      RequireTLS: true
      MaxConnectionsPercent: 100
      MaxIdleConnectionsPercent: 50

  # Update Lambda functions for VPC
  WebhookFunction:
    Properties:
      VpcConfig:
        SecurityGroupIds:
          - !Ref LambdaSecurityGroup
        SubnetIds:
          - !Ref PrivateSubnetA
          - !Ref PrivateSubnetB
      Environment:
        Variables:
          DATABASE_URL: !GetAtt DBProxy.Endpoint
```

## Lambda Functions

### Webhook Handler
```go
// cmd/webhook/main.go
package main

import (
    "context"
    "encoding/json"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/aws/aws-sdk-go-v2/aws"
    "github.com/aws/aws-sdk-go-v2/service/lambda"
)

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Verify Slack signature
    if !verifySlackSignature(request) {
        return events.APIGatewayProxyResponse{StatusCode: 401}, nil
    }
    
    // Handle URL verification challenge
    var payload map[string]interface{}
    json.Unmarshal([]byte(request.Body), &payload)
    
    if payload["type"] == "url_verification" {
        return events.APIGatewayProxyResponse{
            StatusCode: 200,
            Body: payload["challenge"].(string),
        }, nil
    }
    
    // Invoke processor Lambda asynchronously
    lambdaClient := lambda.NewFromConfig(cfg)
    lambdaClient.Invoke(ctx, &lambda.InvokeInput{
        FunctionName:   aws.String(os.Getenv("PROCESSOR_FUNCTION")),
        InvocationType: aws.String("Event"), // Async
        Payload:        []byte(request.Body),
    })
    
    // Return immediately
    return events.APIGatewayProxyResponse{StatusCode: 200}, nil
}

func main() {
    lambda.Start(handler)
}
```

### Scheduler Function
```go
// cmd/scheduler/main.go
package main

import (
    "context"
    "time"
    "github.com/aws/aws-lambda-go/lambda"
)

func handler(ctx context.Context) error {
    // Query active standup configurations
    configs, err := store.GetActiveStandupConfigs(ctx)
    if err != nil {
        return err
    }
    
    now := time.Now()
    for _, config := range configs {
        if shouldTriggerStandup(config, now) {
            // Invoke processor to handle standup
            invokeProcessor(ctx, StandupTriggerEvent{
                ConfigID: config.ID,
                ChannelID: config.ChannelID,
            })
        }
    }
    
    return nil
}
```

## Database Schemas

### Option 1: DynamoDB Schema (Single Table Design)
```
# Workspace
PK: WORKSPACE#<team_id>
SK: WORKSPACE#<team_id>

# Standup Configuration
PK: WORKSPACE#<team_id>
SK: CONFIG#<channel_id>
GSI1PK: ACTIVE#true
GSI1SK: SCHEDULE#<time>

# Standup Session
PK: SESSION#<session_id>
SK: SESSION#<session_id>
GSI1PK: CHANNEL#<channel_id>
GSI1SK: TIMESTAMP#<scheduled_time>

# User Response
PK: SESSION#<session_id>
SK: USER#<user_id>

# Scheduled Messages (for retries)
PK: SCHEDULED
SK: TIMESTAMP#<send_time>#<message_id>
TTL: <unix_timestamp>
```

### Option 2: Aurora Serverless v2 PostgreSQL Schema
```sql
-- Workspaces table
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slack_team_id VARCHAR(255) UNIQUE NOT NULL,
    slack_team_name VARCHAR(255) NOT NULL,
    bot_token TEXT NOT NULL, -- Encrypted
    app_token TEXT NOT NULL, -- Encrypted
    installed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Standup configurations
CREATE TABLE standup_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id VARCHAR(255) NOT NULL,
    channel_name VARCHAR(255) NOT NULL,
    schedule_cron VARCHAR(100) NOT NULL, -- Cron expression
    timezone VARCHAR(100) NOT NULL DEFAULT 'UTC',
    questions JSONB NOT NULL DEFAULT '[]'::jsonb,
    active_days TEXT[] DEFAULT ARRAY['mon','tue','wed','thu','fri'],
    is_active BOOLEAN DEFAULT true,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(workspace_id, channel_id)
);

-- Standup sessions
CREATE TABLE standup_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES standup_configs(id) ON DELETE CASCADE,
    scheduled_for TIMESTAMP WITH TIME ZONE NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    thread_ts VARCHAR(255), -- Slack thread timestamp
    participant_count INTEGER DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'cancelled'))
);

-- User responses
CREATE TABLE standup_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES standup_sessions(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    responses JSONB NOT NULL,
    response_time_seconds INTEGER, -- Time taken to respond
    submitted_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(session_id, user_id)
);

-- Indexes for performance
CREATE INDEX idx_configs_active ON standup_configs(is_active, workspace_id);
CREATE INDEX idx_sessions_scheduled ON standup_sessions(scheduled_for, status);
CREATE INDEX idx_responses_session ON standup_responses(session_id, submitted_at);

-- Example queries for reporting
-- Get participation rate for a channel
SELECT 
    date_trunc('week', s.scheduled_for) as week,
    COUNT(DISTINCT r.user_id) as participants,
    AVG(r.response_time_seconds) as avg_response_time
FROM standup_sessions s
JOIN standup_responses r ON s.id = r.session_id
WHERE s.config_id = $1
GROUP BY week
ORDER BY week DESC;
```

## Database Access Patterns

### DynamoDB Go Implementation
```go
// internal/store/dynamodb.go
type DynamoStore struct {
    client *dynamodb.Client
    table  string
}

// Get active configurations
func (s *DynamoStore) GetActiveConfigs(ctx context.Context) ([]StandupConfig, error) {
    resp, err := s.client.Query(ctx, &dynamodb.QueryInput{
        TableName: &s.table,
        IndexName: aws.String("GSI1"),
        KeyConditionExpression: aws.String("GSI1PK = :pk"),
        ExpressionAttributeValues: map[string]types.AttributeValue{
            ":pk": &types.AttributeValueMemberS{Value: "ACTIVE#true"},
        },
    })
    // Parse and return configs
}

// Store response
func (s *DynamoStore) StoreResponse(ctx context.Context, resp StandupResponse) error {
    item := map[string]types.AttributeValue{
        "PK": &types.AttributeValueMemberS{Value: fmt.Sprintf("SESSION#%s", resp.SessionID)},
        "SK": &types.AttributeValueMemberS{Value: fmt.Sprintf("USER#%s", resp.UserID)},
        "responses": &types.AttributeValueMemberM{Value: marshalResponses(resp.Responses)},
        "submittedAt": &types.AttributeValueMemberS{Value: resp.SubmittedAt.Format(time.RFC3339)},
    }
    _, err := s.client.PutItem(ctx, &dynamodb.PutItemInput{
        TableName: &s.table,
        Item:      item,
    })
    return err
}
```

### Aurora PostgreSQL Go Implementation
```go
// internal/store/postgres.go
type PostgresStore struct {
    db *pgxpool.Pool
}

// Get active configurations with SQL
func (s *PostgresStore) GetActiveConfigs(ctx context.Context) ([]StandupConfig, error) {
    query := `
        SELECT id, workspace_id, channel_id, schedule_cron, timezone, questions
        FROM standup_configs
        WHERE is_active = true
    `
    rows, err := s.db.Query(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var configs []StandupConfig
    for rows.Next() {
        var c StandupConfig
        err := rows.Scan(&c.ID, &c.WorkspaceID, &c.ChannelID, 
                        &c.Schedule, &c.Timezone, &c.Questions)
        if err != nil {
            return nil, err
        }
        configs = append(configs, c)
    }
    return configs, nil
}

// Store response with transaction
func (s *PostgresStore) StoreResponse(ctx context.Context, resp StandupResponse) error {
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)
    
    // Insert response
    _, err = tx.Exec(ctx, `
        INSERT INTO standup_responses (session_id, user_id, user_name, responses, submitted_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (session_id, user_id) DO UPDATE
        SET responses = $4, submitted_at = $5
    `, resp.SessionID, resp.UserID, resp.UserName, resp.Responses, resp.SubmittedAt)
    
    if err != nil {
        return err
    }
    
    // Update participant count
    _, err = tx.Exec(ctx, `
        UPDATE standup_sessions 
        SET participant_count = (
            SELECT COUNT(DISTINCT user_id) 
            FROM standup_responses 
            WHERE session_id = $1
        )
        WHERE id = $1
    `, resp.SessionID)
    
    return tx.Commit(ctx)
}
```

## Makefile for Building

```makefile
.PHONY: build deploy test

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o bootstrap main.go

deploy:
	sam build
	sam deploy --guided

test:
	go test ./...

local:
	sam local start-api --env-vars local-env.json

clean:
	rm -rf .aws-sam/
	find . -name bootstrap -delete
```

## Development Workflow

### Local Development
```bash
# Install SAM CLI
brew install aws-sam-cli

# Start local API
sam local start-api

# Test with ngrok for Slack webhooks
ngrok http 3000
```

### Deployment
```bash
# First time deployment
sam deploy --guided

# Subsequent deployments
sam deploy

# View logs
sam logs -f WebhookFunction --tail
```

## Cost Analysis

### AWS Lambda Pricing (Scale-to-Zero)
- **Free Tier**: 1M requests/month, 400,000 GB-seconds
- **Beyond Free Tier**: 
  - $0.20 per 1M requests
  - $0.0000166667 per GB-second

### Database Cost Comparison

| Users | DynamoDB | Aurora Serverless v2 |
|-------|----------|---------------------|
| 0-100 | $0/month | $43/month (0.5 ACU min) |
| 100-1000 | $0-2/month | $43-86/month |
| 1000-5000 | $5-10/month | $86-172/month |
| 5000+ | $20-50/month | $172+/month |

### Example Cost Breakdown (1000 users)
**DynamoDB Option:**
- Lambda: $0 (free tier)
- API Gateway: $0 (free tier) 
- DynamoDB: ~$2/month
- **Total: ~$2/month**

**Aurora Serverless v2 Option:**
- Lambda: $0 (free tier)
- API Gateway: $0 (free tier)
- Aurora: $43-86/month (0.5-1 ACU)
- RDS Proxy: $15/month
- VPC NAT Gateway: $45/month
- **Total: ~$103-146/month**

## Performance Optimizations

### 1. Lambda Cold Starts
```go
// Minimize cold starts
func init() {
    // Initialize SDK clients
    cfg, _ = config.LoadDefaultConfig(context.Background())
    dynamoClient = dynamodb.NewFromConfig(cfg)
    slackClient = slack.New(os.Getenv("SLACK_BOT_TOKEN"))
}
```

### 2. Connection Reuse
```go
// Reuse HTTP client across invocations
var httpClient = &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        10,
        MaxIdleConnsPerHost: 2,
    },
}
```

### 3. Provisioned Concurrency (if needed)
```yaml
ProvisionedConcurrencyConfig:
  ProvisionedConcurrentExecutions: 2
```

## Security Best Practices

### 1. Secrets Management
```yaml
SlackBotToken:
  Type: AWS::SecretsManager::Secret
  Properties:
    Description: Slack Bot Token
    SecretString: !Sub |
      {
        "token": "${SlackBotTokenValue}"
      }
```

### 2. IAM Roles (Least Privilege)
```yaml
Policies:
  - Version: '2012-10-17'
    Statement:
      - Effect: Allow
        Action:
          - dynamodb:GetItem
          - dynamodb:PutItem
          - dynamodb:Query
        Resource: !GetAtt StandupTable.Arn
```

### 3. API Gateway Throttling
```yaml
MethodSettings:
  - ResourcePath: "/*"
    HttpMethod: "*"
    ThrottlingBurstLimit: 100
    ThrottlingRateLimit: 50
```

## Monitoring and Observability

### CloudWatch Metrics
- Lambda invocations and errors
- API Gateway 4xx/5xx errors
- DynamoDB consumed capacity
- Cold start frequency

### X-Ray Tracing
```go
import "github.com/aws/aws-xray-sdk-go/xray"

func handler(ctx context.Context) error {
    ctx, seg := xray.BeginSegment(ctx, "standup-handler")
    defer seg.Close(nil)
    
    // Your code here
}
```

### Alarms
```yaml
HighErrorRate:
  Type: AWS::CloudWatch::Alarm
  Properties:
    MetricName: Errors
    Namespace: AWS/Lambda
    Statistic: Sum
    Period: 300
    EvaluationPeriods: 1
    Threshold: 10
```

## Migration from Development to Production

### 1. Development (Socket Mode)
- Use Socket Mode for local development
- No public URL required
- Easier debugging

### 2. Staging (Lambda + ngrok)
- Deploy to Lambda
- Use ngrok for public URL
- Test with real Slack events

### 3. Production (Full Lambda)
- Custom domain with Route53
- CloudFront distribution
- Multi-region failover (optional)

## Advantages of SAM + Lambda

1. **True Scale-to-Zero**: Zero cost when idle
2. **Automatic Scaling**: Handle 1 to 1M users
3. **No Maintenance**: AWS manages all infrastructure
4. **Built-in HA**: Multi-AZ by default
5. **Pay-per-Use**: Perfect for sporadic usage patterns

## Decision Matrix

### Choose DynamoDB When:
- Cost is the primary concern (true $0 idle)
- You have predictable access patterns
- Real-time performance is critical
- You want simpler infrastructure (no VPC)
- Team is comfortable with NoSQL patterns

### Choose Aurora Serverless v2 When:
- You need complex reporting and analytics
- Team has strong SQL expertise
- You can accept $43/month minimum cost
- You need ACID transactions
- Future features may require complex queries

## Hybrid Approach (Best of Both Worlds)
Consider using both databases:
1. **DynamoDB** for real-time operations (webhooks, storing responses)
2. **Aurora** for analytics and reporting (ETL from DynamoDB via Lambda)
3. Use DynamoDB Streams to trigger Lambda for data sync

## Conclusion

The AWS SAM + Lambda architecture provides:
- **Flexible database options** to match your needs
- **True scale-to-zero** with DynamoDB ($0 idle cost)
- **Rich querying** with Aurora Serverless v2
- **Serverless simplicity** with no infrastructure management
- **Enterprise scalability** from 10 to 100,000+ users

For most stand-up bot use cases, **DynamoDB is recommended** due to its zero idle cost and perfect fit for the predictable access patterns of a Slack bot. Consider Aurora Serverless v2 only if complex reporting is a core requirement from day one.