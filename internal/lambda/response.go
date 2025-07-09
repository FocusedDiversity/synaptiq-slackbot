package lambda

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
)

// Response creates an API Gateway response.
func Response(statusCode int, body interface{}) events.APIGatewayProxyResponse {
	var bodyStr string

	switch v := body.(type) {
	case string:
		bodyStr = v
	case []byte:
		bodyStr = string(v)
	default:
		b, err := json.Marshal(body)
		if err != nil {
			b = []byte("{}")
		}
		bodyStr = string(b)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: bodyStr,
	}
}

// OK returns a 200 OK response.
func OK(body interface{}) events.APIGatewayProxyResponse {
	return Response(http.StatusOK, body)
}

// Created returns a 201 Created response.
func Created(body interface{}) events.APIGatewayProxyResponse {
	return Response(http.StatusCreated, body)
}

// NoContent returns a 204 No Content response.
func NoContent() events.APIGatewayProxyResponse {
	return Response(http.StatusNoContent, "")
}

// BadRequest returns a 400 Bad Request response.
func BadRequest(message string) events.APIGatewayProxyResponse {
	return Response(http.StatusBadRequest, map[string]string{
		"error": message,
	})
}

// Unauthorized returns a 401 Unauthorized response.
func Unauthorized(message string) events.APIGatewayProxyResponse {
	return Response(http.StatusUnauthorized, map[string]string{
		"error": message,
	})
}

// Forbidden returns a 403 Forbidden response.
func Forbidden(message string) events.APIGatewayProxyResponse {
	return Response(http.StatusForbidden, map[string]string{
		"error": message,
	})
}

// NotFound returns a 404 Not Found response.
func NotFound(message string) events.APIGatewayProxyResponse {
	return Response(http.StatusNotFound, map[string]string{
		"error": message,
	})
}

// InternalServerError returns a 500 Internal Server Error response.
func InternalServerError(message string) events.APIGatewayProxyResponse {
	return Response(http.StatusInternalServerError, map[string]string{
		"error": message,
	})
}

// SlackResponse returns a response formatted for Slack.
func SlackResponse(text string) events.APIGatewayProxyResponse {
	return OK(map[string]interface{}{
		"response_type": "in_channel",
		"text":          text,
	})
}

// SlackEphemeralResponse returns an ephemeral response for Slack.
func SlackEphemeralResponse(text string) events.APIGatewayProxyResponse {
	return OK(map[string]interface{}{
		"response_type": "ephemeral",
		"text":          text,
	})
}

// SlackBlockResponse returns a block-formatted response for Slack.
func SlackBlockResponse(blocks interface{}) events.APIGatewayProxyResponse {
	return OK(map[string]interface{}{
		"response_type": "in_channel",
		"blocks":        blocks,
	})
}
