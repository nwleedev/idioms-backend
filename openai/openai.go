package openai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nw.lee/idioms-backend/logger"
)

type OpenAi struct {
	apiKey string
	orgId  string
	logger logger.LoggerService
}

type OpenAiInterface interface {
	TextCompletion(args *TextCompletionArgs) (*string, error)
	Image(prompt string) (*string, error)
}

type TextCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type TextCompletionArgs struct {
	Model       string                  `json:"model"`
	Temperature float64                 `json:"temperature"`
	MaxTokens   int                     `json:"max_tokens"`
	Messages    []TextCompletionMessage `json:"messages"`
}

type TextCompletionResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		}
	} `json:"choices"`
}

type ImageBody struct {
	Prompt         string `json:"prompt"`
	Model          string `json:"model"`
	Quality        string `json:"quality"`
	ResponseFormat string `json:"response_format"`
	Style          string `json:"style"`
}

type ImageResponse struct {
	CreatedAt int64 `json:"created"`
	Data      []struct {
		URL string `json:"url"`
	} `json:"data"`
}

func (args *TextCompletionArgs) AddMessage(role string, content string) *TextCompletionArgs {
	if args.Messages == nil {
		args.Messages = []TextCompletionMessage{}
	}
	args.Messages = append(args.Messages, TextCompletionMessage{Role: role, Content: content})
	args.MaxTokens += len(content)
	return args
}

func NewOpenAi(apiKey string, orgId string, logger logger.LoggerService) *OpenAi {
	openAi := new(OpenAi)
	openAi.apiKey = apiKey
	openAi.orgId = orgId
	openAi.logger = logger

	return openAi
}

func (openAi *OpenAi) TextCompletion(args *TextCompletionArgs) (*string, error) {
	url := "https://api.openai.com/v1/chat/completions"

	buf, err := json.Marshal(args)
	if err != nil {
		openAi.logger.Println("Invalid arguments")
		openAi.logger.PrintError("", err)
		return nil, err
	}
	body := bytes.NewBuffer(buf)
	token := fmt.Sprintf("Bearer %s", openAi.apiKey)

	req, _ := http.NewRequest(http.MethodPost, url, body)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("authorization", token)
	req.Header.Add("OpenAI-Organization", openAi.orgId)
	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		openAi.logger.Println("Failed to execute text completion")
		openAi.logger.PrintError("", err)
		return nil, err
	}
	defer resp.Body.Close()

	response := new(TextCompletionResponse)
	err = json.NewDecoder(resp.Body).Decode(response)
	if err != nil || len(response.Choices) == 0 {
		openAi.logger.Println("Failed to decode response")
		openAi.logger.PrintError("", err)
		return nil, err
	}
	content := response.Choices[0].Message.Content
	return &content, nil
}

func (openAi *OpenAi) Image(prompt string) (*string, error) {
	url := "https://api.openai.com/v1/images/generations"
	message := fmt.Sprintf("Here are the instructions you must follow. \n%s", prompt)

	data := &ImageBody{
		Prompt:         message,
		Model:          "dall-e-3",
		Quality:        "hd",
		ResponseFormat: "url",
		Style:          "vivid",
	}

	buf, err := json.Marshal(data)
	if err != nil {
		openAi.logger.Println("Failed to create a new request")
		openAi.logger.PrintError("", err)
		return nil, err
	}
	body := bytes.NewBuffer(buf)
	token := fmt.Sprintf("Bearer %s", openAi.apiKey)
	req, _ := http.NewRequest(http.MethodPost, url, body)

	req.Header.Add("content-type", "application/json")
	req.Header.Add("authorization", token)
	req.Header.Add("OpenAI-Organization", openAi.orgId)
	client := new(http.Client)
	resp, err := client.Do(req)

	if err != nil {
		openAi.logger.Println("Failed to create a new image from prompt %s.", prompt)
		openAi.logger.PrintError("", err)
		return nil, err
	}

	defer resp.Body.Close()

	response := new(ImageResponse)
	err = json.NewDecoder(resp.Body).Decode(response)

	if err != nil || len(response.Data) == 0 {
		openAi.logger.Println("Failed to decode response")
		openAi.logger.PrintError("", err)
		return nil, err
	}

	image := response.Data[0].URL
	return &image, nil
}
