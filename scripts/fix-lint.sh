#!/usr/bin/env bash
# Quick fixes for common lint issues

set -euo pipefail

echo "Fixing common lint issues..."

# Fix error check issues in builders.go
echo "Fixing error checks..."

# Fix json.Marshal error check in builders.go
sed -i '' 's/data, _ := json.Marshal(metadata)/data, err := json.Marshal(metadata); if err != nil { return b }/' internal/slack/builders.go

# Fix type assertion in builders.go
cat > /tmp/builders_fix.txt << 'EOF'
	if placeholder != "" {
		if element, ok := input.Element.(PlainTextInputElement); ok {
			element.Placeholder = &TextBlock{
				Type: "plain_text",
				Text: placeholder,
			}
			input.Element = element
		}
	}
EOF

# Apply the fix
awk '
/if placeholder != ""/ { 
    print; 
    getline; # element := line
    getline; # element.Placeholder line
    getline; getline; getline; getline; # skip the rest
    system("cat /tmp/builders_fix.txt")
    next
}
{ print }
' internal/slack/builders.go > internal/slack/builders.go.tmp && mv internal/slack/builders.go.tmp internal/slack/builders.go

# Fix json.Marshal in response.go
sed -i '' 's/b, _ := json.Marshal(body)/b, err := json.Marshal(body); if err != nil { b = []byte("{}") }/' internal/lambda/response.go

# Fix error check in processor/main.go
sed -i '' 's/startDate, _ := task.Payload\["start_date"\].(string)/startDate, ok := task.Payload["start_date"].(string); if !ok { startDate = "" }/' cmd/processor/main.go

# Run gofumpt again
echo "Running gofumpt..."
gofumpt -w .

# Run goimports again
echo "Running goimports..."
goimports -w -local github.com/synaptiq/standup-bot .

echo "Lint fixes applied!"