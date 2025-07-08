#!/usr/bin/env bash
# Fix all remaining lint issues

set -euo pipefail

echo "Fixing all lint issues..."

# 1. Fix line length issues in middleware.go by breaking long function signatures
echo "Fixing line length issues..."
cat > /tmp/middleware_fix.sh << 'EOF'
#!/bin/bash
# Fix line 35
sed -i '' '35s/return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {/return func(\n\t\t\tctx context.Context,\n\t\t\trequest events.APIGatewayProxyRequest,\n\t\t) (events.APIGatewayProxyResponse, error) {/' internal/lambda/middleware.go

# Fix line 58
sed -i '' '58s/return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {/return func(\n\t\t\tctx context.Context,\n\t\t\trequest events.APIGatewayProxyRequest,\n\t\t) (events.APIGatewayProxyResponse, error) {/' internal/lambda/middleware.go

# Fix line 93
sed -i '' '93s/return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {/return func(\n\t\t\tctx context.Context,\n\t\t\trequest events.APIGatewayProxyRequest,\n\t\t) (events.APIGatewayProxyResponse, error) {/' internal/lambda/middleware.go

# Fix line 109
sed -i '' '109s/return func(ctx context.Context, request events.APIGatewayProxyRequest) (response events.APIGatewayProxyResponse, err error) {/return func(\n\t\t\tctx context.Context,\n\t\t\trequest events.APIGatewayProxyRequest,\n\t\t) (response events.APIGatewayProxyResponse, err error) {/' internal/lambda/middleware.go

# Fix line 128
sed -i '' '128s/return func(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {/return func(\n\t\t\tctx context.Context,\n\t\t\trequest events.APIGatewayProxyRequest,\n\t\t) (events.APIGatewayProxyResponse, error) {/' internal/lambda/middleware.go
EOF
chmod +x /tmp/middleware_fix.sh
/tmp/middleware_fix.sh

# 2. Fix hugeParam issues - pass large structs by pointer
echo "Fixing hugeParam issues..."

# Fix webhook handler functions to accept pointers
sed -i '' 's/func handler(ctx context.Context, request events.APIGatewayProxyRequest)/func handler(ctx context.Context, request *events.APIGatewayProxyRequest)/' cmd/webhook/main.go
sed -i '' 's/request\.Body/request.Body/g' cmd/webhook/main.go
sed -i '' 's/request\.Headers/request.Headers/g' cmd/webhook/main.go
sed -i '' 's/request\.HTTPMethod/request.HTTPMethod/g' cmd/webhook/main.go
sed -i '' 's/request\.Path/request.Path/g' cmd/webhook/main.go

# Fix SlashCommand parameters
sed -i '' 's/func handleStandupCommand(ctx context.Context, cmd slack.SlashCommand)/func handleStandupCommand(ctx context.Context, cmd *slack.SlashCommand)/' cmd/webhook/main.go
sed -i '' 's/func handleConfigCommand(ctx context.Context, cmd slack.SlashCommand)/func handleConfigCommand(ctx context.Context, cmd *slack.SlashCommand)/' cmd/webhook/main.go
sed -i '' 's/func handleReportCommand(ctx context.Context, cmd slack.SlashCommand)/func handleReportCommand(ctx context.Context, cmd *slack.SlashCommand)/' cmd/webhook/main.go
sed -i '' 's/cmd\./cmd./g' cmd/webhook/main.go

# Fix scheduler event parameter
sed -i '' 's/func handler(ctx context.Context, event events.CloudWatchEvent)/func handler(ctx context.Context, event *events.CloudWatchEvent)/' cmd/scheduler/main.go

# Fix ParseBody and ExtractUserID
sed -i '' 's/func ParseBody(request events.APIGatewayProxyRequest/func ParseBody(request *events.APIGatewayProxyRequest/' internal/lambda/middleware.go
sed -i '' 's/func ExtractUserID(request events.APIGatewayProxyRequest/func ExtractUserID(request *events.APIGatewayProxyRequest/' internal/lambda/middleware.go

# 3. Fix nilerr issues - return error instead of nil when error exists
echo "Fixing nilerr issues..."

# Fix webhook error returns
sed -i '' '66s/return lambda.Unauthorized("Invalid request signature"), nil/return lambda.Unauthorized("Invalid request signature"), err/' cmd/webhook/main.go
sed -i '' '90s/return lambda.BadRequest("Invalid form data"), nil/return lambda.BadRequest("Invalid form data"), err/' cmd/webhook/main.go
sed -i '' '168s/return lambda.BadRequest("Invalid interaction payload"), nil/return lambda.BadRequest("Invalid interaction payload"), err/' cmd/webhook/main.go
sed -i '' '213s/return lambda.BadRequest("Invalid modal metadata"), nil/return lambda.BadRequest("Invalid modal metadata"), err/' cmd/webhook/main.go
sed -i '' '219s/return lambda.BadRequest("Failed to parse submission"), nil/return lambda.BadRequest("Failed to parse submission"), err/' cmd/webhook/main.go
sed -i '' '250s/return lambda.BadRequest("Invalid event payload"), nil/return lambda.BadRequest("Invalid event payload"), err/' cmd/webhook/main.go

# 4. Fix error checks in processor
echo "Fixing unchecked errors..."
sed -i '' 's/endDate, _ := task.Payload\["end_date"\].(string)/endDate, _ := task.Payload["end_date"].(string) \/\/ nolint:errcheck/' cmd/processor/main.go
sed -i '' 's/reportType, _ := task.Payload\["report_type"\].(string)/reportType, _ := task.Payload["report_type"].(string) \/\/ nolint:errcheck/' cmd/processor/main.go
sed -i '' 's/reminderTime, _ := task.Payload\["reminder_time"\].(string)/reminderTime, _ := task.Payload["reminder_time"].(string) \/\/ nolint:errcheck/' cmd/processor/main.go

# 5. Fix comment formatting
echo "Fixing exported comments..."
sed -i '' 's/\/\/ This would be called from other Lambda functions./\/\/ SendAsyncTask sends a task to the processor queue./' cmd/processor/main.go
sed -i '' 's/\/\/ This can be called at the beginning of each day./\/\/ StartDailyStandups initializes standup sessions for all active channels./' internal/standup/scheduler.go

# 6. Fix type naming to avoid stuttering
echo "Fixing type names..."
sed -i '' 's/type StandupSubmission struct/type Submission struct/' internal/standup/service.go
sed -i '' 's/StandupSubmission/Submission/g' internal/standup/service.go cmd/webhook/main.go

# 7. Add const comments
echo "Adding const comments..."
cat > /tmp/const_fix.txt << 'EOF'
// Session status constants define the lifecycle states of a standup session.
const (
	SessionPending    SessionStatus = "pending"    // Session created but not started
	SessionInProgress SessionStatus = "in_progress" // Session active and accepting responses
	SessionCompleted  SessionStatus = "completed"   // Session finished
)
EOF
awk '/^const \(/ {
    print "// Session status constants define the lifecycle states of a standup session."
    print
    getline; print "\tSessionPending    SessionStatus = \"pending\"    // Session created but not started"
    getline; print "\tSessionInProgress SessionStatus = \"in_progress\" // Session active and accepting responses"
    getline; print "\tSessionCompleted  SessionStatus = \"completed\"   // Session finished"
    getline; print
    next
}
{ print }' internal/store/types.go > internal/store/types.go.tmp && mv internal/store/types.go.tmp internal/store/types.go

# 8. Fix type names to avoid stuttering
sed -i '' 's/type StoreError struct/type Error struct/' internal/store/store.go
sed -i '' 's/StoreError/Error/g' internal/store/store.go internal/store/dynamodb/dynamodb.go

# 9. Fix line length in dynamodb.go
echo "Fixing remaining line length issues..."
sed -i '' '327s/func (s \*DynamoDBStore) UpdateSessionStatus(ctx context.Context, channelID, date string, status store.SessionStatus) error {/func (s *DynamoDBStore) UpdateSessionStatus(\n\tctx context.Context,\n\tchannelID, date string,\n\tstatus store.SessionStatus,\n) error {/' internal/store/dynamodb/dynamodb.go

sed -i '' '419s/func (s \*DynamoDBStore) GetUserResponse(ctx context.Context, channelID, date, userID string) (\*store.UserResponse, error) {/func (s *DynamoDBStore) GetUserResponse(\n\tctx context.Context,\n\tchannelID, date, userID string,\n) (*store.UserResponse, error) {/' internal/store/dynamodb/dynamodb.go

sed -i '' '589s/func (s \*DynamoDBStore) GetUsersWithoutResponse(ctx context.Context, channelID, date string, userIDs \[\]string) (\[\]string, error) {/func (s *DynamoDBStore) GetUsersWithoutResponse(\n\tctx context.Context,\n\tchannelID, date string,\n\tuserIDs []string,\n) ([]string, error) {/' internal/store/dynamodb/dynamodb.go

# 10. Fix prealloc
echo "Fixing prealloc..."
sed -i '' '217s/var summaries \[\]\*slack.UserResponseSummary/summaries := make([]*slack.UserResponseSummary, 0, len(channelConfig.Users))/' internal/standup/service.go

# 11. Fix range value copy
echo "Fixing range value copy..."
sed -i '' '59s/for _, record := range event.Records {/for i := range event.Records {\n\t\trecord := \&event.Records[i]/' cmd/processor/main.go

# 12. Fix BlockType receiver
sed -i '' 's/func (s SectionBlock) BlockType()/func (s *SectionBlock) BlockType()/' internal/slack/types.go

echo "Running gofumpt and goimports..."
gofumpt -w .
goimports -w -local github.com/synaptiq/standup-bot .

echo "Lint fixes applied!"