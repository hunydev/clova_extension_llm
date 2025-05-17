package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	openai "github.com/sashabaranov/go-openai"
)

type ClovaRequest struct {
	Version string `json:"version"`
	Session struct {
		New       bool                   `json:"new"`
		SessionID string                 `json:"sessionId"`
		User      map[string]interface{} `json:"user"`
	} `json:"session"`
	Context struct {
		System struct {
			Application struct {
				ApplicationID string `json:"applicationId"`
			} `json:"application"`
			User struct {
				UserID      string `json:"userId"`
				AccessToken string `json:"accessToken"`
			} `json:"user"`
			Device struct {
				DeviceID string `json:"deviceId"`
			} `json:"device"`
		} `json:"System"`
	} `json:"context"`
	Request struct {
		Type   string `json:"type"`
		Intent struct {
			Name  string `json:"name"`
			Slots struct {
				Question struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				} `json:"question"`
			} `json:"slots"`
		} `json:"intent"`
	} `json:"request"`
}

type OutputSpeech struct {
	Type   string `json:"type"`
	Values struct {
		Type  string `json:	ype"`
		Lang  string `json:"lang"`
		Value string `json:"value"`
	} `json:"values"`
}

type ClovaResponse struct {
	Version           string                 `json:"version"`
	SessionAttributes map[string]interface{} `json:"sessionAttributes"`
	Response          struct {
		OutputSpeech     OutputSpeech  `json:"outputSpeech"`
		Card             interface{}   `json:"card"`
		Directives       []interface{} `json:"directives"`
		ShouldEndSession bool          `json:"shouldEndSession"`
	} `json:"response"`
}

func handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Parse the incoming request
	var clovaReq ClovaRequest
	if err := json.Unmarshal([]byte(request.Body), &clovaReq); err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("Error parsing request: %v", err),
		}, nil
	}

	// Check if it's an IntentRequest
	if clovaReq.Request.Type != "IntentRequest" || clovaReq.Request.Intent.Name != "AskLLMIntent" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "Invalid request type or intent",
		}, nil
	}

	// Get the question from the intent
	question := clovaReq.Request.Intent.Slots.Question.Value
	if question == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Body:       "No question provided",
		}, nil
	}

	// Call OpenAI API
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       "OpenAI API key not configured",
		}, nil
	}

	client := openai.NewClient(openaiKey)

	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: "gpt-4.1-nano",
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant that provides clear and concise answers in Korean.",
				},
				{
					Role:    openai.ChatMessageRoleFunction,
					Content: "TTS가 답변을 잘 할 수 있도록 이모티콘은 쓰지 않고, 영어나 숫자는 한글로 노말라이즈 하여 적절하게 발화할 수 있도록 해줘.",
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: question,
				},
			},
		},
	)

	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error calling OpenAI: %v", err),
		}, nil
	}

	// Prepare the response
	var clovaResp ClovaResponse
	clovaResp.Version = "0.1.0"
	clovaResp.SessionAttributes = map[string]interface{}{}
	clovaResp.Response.OutputSpeech.Type = "SimpleSpeech"
	clovaResp.Response.OutputSpeech.Values.Type = "PlainText"
	clovaResp.Response.OutputSpeech.Values.Lang = "ko"
	clovaResp.Response.OutputSpeech.Values.Value = resp.Choices[0].Message.Content
	clovaResp.Response.Card = struct{}{}
	clovaResp.Response.Directives = []interface{}{}
	clovaResp.Response.ShouldEndSession = false

	// Convert response to JSON
	responseJSON, err := json.Marshal(clovaResp)
	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Body:       fmt.Sprintf("Error creating response: %v", err),
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: string(responseJSON),
	}, nil
}

func main() {
	lambda.Start(handler)
}
