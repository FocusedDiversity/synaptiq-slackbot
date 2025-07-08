# Deployment Guide

This guide walks through deploying the Synaptiq Standup Slackbot to AWS.

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **AWS CLI** configured with credentials
3. **AWS SAM CLI** installed
4. **Go 1.21+** installed
5. **Slack App** created with the following:
   - Bot User OAuth Token
   - Signing Secret
   - Event Subscriptions enabled
   - Slash Commands configured

## Slack App Setup

### 1. Create Slack App

1. Go to [api.slack.com/apps](https://api.slack.com/apps)
2. Click "Create New App" → "From scratch"
3. Name: "Standup Bot"
4. Choose your workspace

### 2. Configure OAuth & Permissions

1. Navigate to "OAuth & Permissions"
2. Add Bot Token Scopes:
   - `chat:write` - Send messages
   - `chat:write.public` - Post to public channels
   - `im:write` - Send DMs
   - `users:read` - Get user info
   - `users:read.email` - Get user emails
   - `channels:read` - List channels
   - `groups:read` - List private channels
3. Install to Workspace
4. Copy the "Bot User OAuth Token" (starts with `xoxb-`)

### 3. Configure Event Subscriptions

1. Navigate to "Event Subscriptions"
2. Enable Events
3. Request URL will be set after deployment
4. Subscribe to bot events:
   - `app_mention`
   - `message.im`
5. Save Changes

### 4. Configure Slash Commands

Create the following commands:

1. `/standup` - Submit daily standup
   - Command: `/standup`
   - Request URL: Will be set after deployment
   - Short Description: "Submit your daily standup"

2. `/standup-config` - Configure standup settings
   - Command: `/standup-config`
   - Request URL: Will be set after deployment
   - Short Description: "Configure standup settings"

3. `/standup-report` - View standup reports
   - Command: `/standup-report`
   - Request URL: Will be set after deployment
   - Short Description: "View standup reports"

### 5. Configure Interactivity

1. Navigate to "Interactivity & Shortcuts"
2. Turn on Interactivity
3. Request URL will be set after deployment

### 6. Get Signing Secret

1. Navigate to "Basic Information"
2. Copy the "Signing Secret"

## AWS Deployment

### 1. First-Time Setup

```bash
# Clone the repository
git clone https://github.com/synaptiq/standup-bot.git
cd standup-bot

# Install dependencies
make deps

# Copy and configure the bot
cp config.example.yaml config.yaml
# Edit config.yaml with your channel IDs and settings

# Set environment variables
export SLACK_BOT_TOKEN="xoxb-your-bot-token"
export SLACK_SIGNING_SECRET="your-signing-secret"
```

### 2. Deploy to AWS

#### Option A: Guided Deployment (Recommended for first time)

```bash
make deploy-guided
```

This will prompt you for:

- Stack name
- AWS Region
- Slack tokens
- Parameter values
- Confirmation to deploy

#### Option B: Direct Deployment

```bash
# Deploy to development
make deploy ENVIRONMENT=dev

# Deploy to staging
make deploy ENVIRONMENT=staging SAM_CONFIG_ENV=staging

# Deploy to production
make deploy ENVIRONMENT=prod SAM_CONFIG_ENV=prod
```

### 3. Update Slack App URLs

After deployment, get the API URLs:

```bash
make outputs
```

Update your Slack app with the URLs:

1. **Event Subscriptions**:
   - Request URL: `https://<api-id>.execute-api.<region>.amazonaws.com/<stage>/slack/events`

2. **Slash Commands**:
   - Request URL: `https://<api-id>.execute-api.<region>.amazonaws.com/<stage>/slack/commands`

3. **Interactivity**:
   - Request URL: `https://<api-id>.execute-api.<region>.amazonaws.com/<stage>/slack/interactive`

## Configuration

### Update Channel Configuration

1. Edit `config.yaml` with your channel settings:
   - Channel IDs
   - User IDs
   - Schedule times
   - Message templates

2. Redeploy the application:

   ```bash
   make deploy
   ```

### Environment-Specific Configuration

Create separate config files for each environment:

- `config.dev.yaml`
- `config.staging.yaml`
- `config.prod.yaml`

Update the Lambda environment variable:

```yaml
CONFIG_PATH: ./config.prod.yaml
```

## Monitoring

### View Logs

```bash
# Webhook function logs
make logs-webhook

# Scheduler function logs
make logs-scheduler

# Processor function logs
make logs-processor
```

### CloudWatch Alarms

The stack creates alarms for:

- High error rates
- Messages in DLQ

Configure SNS notifications for production alerts.

### X-Ray Tracing

View distributed traces in AWS X-Ray console to debug issues.

## Testing

### Local Testing

```bash
# Start local API
make local-api

# Test with ngrok
ngrok http 3000
```

### Integration Testing

```bash
# Generate test events
make generate-events

# Test individual functions
make test-webhook
make test-scheduler
make test-processor
```

## Maintenance

### Update Dependencies

```bash
make deps
```

### Update Lambda Functions

```bash
# Make code changes
# Then redeploy
make deploy
```

### Scale Configuration

Adjust in `template.yaml`:

- Memory allocation
- Timeout values
- Reserved concurrency
- SQS batch sizes

## Rollback

If issues occur:

```bash
# View stack history
aws cloudformation list-stack-resources --stack-name synaptiq-standup-bot-prod

# Rollback to previous version
aws cloudformation update-stack \
  --stack-name synaptiq-standup-bot-prod \
  --use-previous-template
```

## Troubleshooting

### Common Issues

1. **"Invalid signature" errors**
   - Verify SLACK_SIGNING_SECRET is correct
   - Check timestamp validation (5-minute window)

2. **"Channel not found" errors**
   - Ensure bot is invited to the channel
   - Verify channel ID in config.yaml

3. **DMs not sending**
   - Check bot has `im:write` permission
   - Verify user is not a bot

4. **High Lambda costs**
   - Review CloudWatch logs for errors
   - Check for infinite loops
   - Optimize DynamoDB queries

### Debug Mode

Enable debug logging:

```bash
export LOG_LEVEL=debug
make deploy
```

## Security Best Practices

1. **Rotate Tokens Regularly**
   - Update in AWS Secrets Manager
   - Redeploy functions

2. **Restrict IAM Permissions**
   - Review Lambda execution roles
   - Apply least privilege

3. **Monitor Access**
   - Enable CloudTrail
   - Review API Gateway logs

4. **Data Retention**
   - Configure DynamoDB TTL
   - Review compliance requirements

## Cost Optimization

1. **Lambda**
   - Optimize memory allocation
   - Use ARM architecture
   - Enable Lambda SnapStart

2. **DynamoDB**
   - Use on-demand billing
   - Configure appropriate TTL
   - Monitor consumed capacity

3. **CloudWatch**
   - Set appropriate log retention
   - Use log filters

## Support

For issues:

1. Check CloudWatch logs
2. Review X-Ray traces
3. Open GitHub issue
4. Contact maintainers
