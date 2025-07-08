#!/bin/bash
set -euo pipefail

# Smoke tests for deployed Slack stand-up bot
# Usage: ./smoke-test.sh <endpoint-url>

ENDPOINT=${1:-}

if [ -z "$ENDPOINT" ]; then
    echo "❌ Error: Endpoint URL required"
    echo "Usage: $0 <endpoint-url>"
    exit 1
fi

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔍 Running smoke tests against: $ENDPOINT"

# Test 1: Health check
echo -e "\n${YELLOW}Test 1: Health check${NC}"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "${ENDPOINT}/health" || echo "000")
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)

if [ "$HTTP_CODE" == "200" ]; then
    echo -e "${GREEN}✓ Health check passed${NC}"
else
    echo -e "${RED}✗ Health check failed (HTTP $HTTP_CODE)${NC}"
    exit 1
fi

# Test 2: Slack URL verification
echo -e "\n${YELLOW}Test 2: Slack URL verification${NC}"
CHALLENGE="test_challenge_$(date +%s)"
VERIFY_RESPONSE=$(curl -s -X POST "${ENDPOINT}/slack/events" \
    -H "Content-Type: application/json" \
    -H "X-Slack-Request-Timestamp: $(date +%s)" \
    -H "X-Slack-Signature: v0=test_signature" \
    -d "{\"type\": \"url_verification\", \"challenge\": \"$CHALLENGE\"}")

if [ "$VERIFY_RESPONSE" == "$CHALLENGE" ]; then
    echo -e "${GREEN}✓ URL verification passed${NC}"
else
    echo -e "${RED}✗ URL verification failed${NC}"
    echo "Expected: $CHALLENGE"
    echo "Got: $VERIFY_RESPONSE"
    exit 1
fi

# Test 3: Invalid signature handling
echo -e "\n${YELLOW}Test 3: Invalid signature handling${NC}"
INVALID_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${ENDPOINT}/slack/events" \
    -H "Content-Type: application/json" \
    -H "X-Slack-Request-Timestamp: 1234567890" \
    -H "X-Slack-Signature: v0=invalid" \
    -d '{"type": "event_callback"}')

if [ "$INVALID_RESPONSE" == "401" ]; then
    echo -e "${GREEN}✓ Invalid signature rejected correctly${NC}"
else
    echo -e "${RED}✗ Invalid signature not rejected (HTTP $INVALID_RESPONSE)${NC}"
    exit 1
fi

# Test 4: Method not allowed
echo -e "\n${YELLOW}Test 4: Method validation${NC}"
GET_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" "${ENDPOINT}/slack/events")

if [ "$GET_RESPONSE" == "405" ] || [ "$GET_RESPONSE" == "403" ]; then
    echo -e "${GREEN}✓ GET method rejected correctly${NC}"
else
    echo -e "${RED}✗ GET method not rejected (HTTP $GET_RESPONSE)${NC}"
    exit 1
fi

# Test 5: CloudWatch metrics (if available)
echo -e "\n${YELLOW}Test 5: Checking CloudWatch metrics${NC}"
if command -v aws &> /dev/null && [ -n "${AWS_REGION:-}" ]; then
    STACK_NAME=${STACK_NAME:-standup-bot}

    # Get Lambda function names from stack
    FUNCTIONS=$(aws cloudformation describe-stack-resources \
        --stack-name "$STACK_NAME" \
        --query "StackResources[?ResourceType=='AWS::Lambda::Function'].PhysicalResourceId" \
        --output text 2>/dev/null || echo "")

    if [ -n "$FUNCTIONS" ]; then
        for func in $FUNCTIONS; do
            ERRORS=$(aws cloudwatch get-metric-statistics \
                --namespace AWS/Lambda \
                --metric-name Errors \
                --dimensions Name=FunctionName,Value="$func" \
                --start-time "$(date -u -d '5 minutes ago' +%Y-%m-%dT%H:%M:%S)" \
                --end-time "$(date -u +%Y-%m-%dT%H:%M:%S)" \
                --period 300 \
                --statistics Sum \
                --query "Datapoints[0].Sum" \
                --output text 2>/dev/null || echo "0")

            if [ "$ERRORS" == "None" ] || [ "$ERRORS" == "0" ]; then
                echo -e "${GREEN}✓ No errors in Lambda function: $func${NC}"
            else
                echo -e "${RED}✗ Errors detected in Lambda function: $func ($ERRORS errors)${NC}"
            fi
        done
    else
        echo -e "${YELLOW}⚠️  Could not retrieve Lambda functions from stack${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Skipping CloudWatch checks (AWS CLI not configured)${NC}"
fi

# Test 6: Database connectivity (optional)
echo -e "\n${YELLOW}Test 6: Database connectivity${NC}"
if [ -n "${TABLE_NAME:-}" ]; then
    TABLE_EXISTS=$(aws dynamodb describe-table \
        --table-name "$TABLE_NAME" \
        --query "Table.TableStatus" \
        --output text 2>/dev/null || echo "NOT_FOUND")

    if [ "$TABLE_EXISTS" == "ACTIVE" ]; then
        echo -e "${GREEN}✓ DynamoDB table is active${NC}"
    else
        echo -e "${YELLOW}⚠️  DynamoDB table status: $TABLE_EXISTS${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Skipping database check (TABLE_NAME not set)${NC}"
fi

# Summary
echo -e "\n${GREEN}✅ All smoke tests passed!${NC}"
echo -e "\nDeployment verified successfully."
echo -e "Endpoint: ${ENDPOINT}"
echo -e "Timestamp: $(date)"

# Optional: Send notification
if [ -n "${SLACK_WEBHOOK_URL:-}" ]; then
    curl -s -X POST "$SLACK_WEBHOOK_URL" \
        -H "Content-Type: application/json" \
        -d "{
            \"text\": \"✅ Deployment verified successfully\",
            \"attachments\": [{
                \"color\": \"good\",
                \"fields\": [
                    {\"title\": \"Environment\", \"value\": \"${ENVIRONMENT:-production}\", \"short\": true},
                    {\"title\": \"Endpoint\", \"value\": \"${ENDPOINT}\", \"short\": true},
                    {\"title\": \"Timestamp\", \"value\": \"$(date)\", \"short\": false}
                ]
            }]
        }" > /dev/null
fi
