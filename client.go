package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"time"
)

// Client is a client for the OpenAI API.
//
// https://platform.openai.com/docs/api-reference
type Client struct {
	// APIKey is the API key to use for requests.
	APIKey string

	// HTTPClient is the HTTP client to use for requests.
	HTTPClient *http.Client

	// Organization is the organization to use for requests.
	Organization string
}

// ClientOption is a function that configures a Client.
type ClientOption func(*Client)

// WithHTTPClient is a ClientOption that sets the HTTP client to use for requests.
//
// If the client is nil, then http.DefaultClient is used
func WithHTTPClient(c *http.Client) ClientOption {
	return func(client *Client) {
		if c == nil {
			c = http.DefaultClient
		}
		client.HTTPClient = c
	}
}

// WithOrganization is a ClientOption that sets the organization to use for requests.
//
// https://platform.openai.com/docs/api-reference/authentication
func WithOrganization(org string) ClientOption {
	return func(client *Client) {
		client.Organization = org
	}
}

// NewClient returns a new Client with the given API key.
//
// # Example
//
//	c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		APIKey:     apiKey,
		HTTPClient: http.DefaultClient,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Role is the role of the user for a chat message.
type Role = string

const (
	// RoleSystem is a special used to ground the model within the context of the conversation.
	//
	// For example, it may be used to provide a name for the assistant, or to provide other global information
	// or instructions that the model should know about.
	RoleSystem Role = "system"

	// RoleUser is the role of the user for a chat message.
	RoleUser Role = "user"

	// RoleAssistant is the role of the assistant for a chat message.
	RoleAssistant Role = "assistant"

	// RoleFunction is a special role used to represent a function call.
	RoleFunction Role = "function"
)

// CreateCompletionRequest contains information for a "completion" request
// to the OpenAI API. This is the fundamental request type for the API.
//
// https://platform.openai.com/docs/api-reference/completions/create
type CreateCompletionRequest struct {
	// ID of the model to use. You can use the List models API to see all of your available models, or see our Model overview for descriptions of them.
	//
	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-model
	Model string `json:"model"`

	// The prompt(s) to generate completions for, encoded as a string, array of strings, array of tokens, or array of token arrays.
	//
	// Note that <|endoftext|> is the document separator that the model sees during training, so if a prompt is not specified the model
	// will generate as if from the beginning of a new document.
	//
	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-prompt
	Prompt []string `json:"prompt"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-suffix
	Suffix string `json:"suffix,omitempty"`

	// The maximum number of tokens to generate in the completion.
	//
	// The token count of your prompt plus max_tokens cannot exceed the model's context length. Most models have a context
	// length of 2048 tokens (except for the newest models, which support 4096).
	//
	// Defaults to 16 if not specified.
	//
	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-max_tokens
	MaxTokens int `json:"max_tokens,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-temperature
	//
	// Defaults to 1 if not specified.
	Temperature float64 `json:"temperature,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-top_p
	//
	// Defaults to 1 if not specified.
	TopP float64 `json:"top_p,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-n
	//
	// Defaults to 1 if not specified.
	N int `json:"n,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-stream
	//
	// Defaults to false if not specified.
	Stream bool `json:"stream,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-logprobs
	//
	// Defaults to nil.
	LogProbs *int `json:"logprobs,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-echo
	//
	// Defaults to false if not specified.
	Echo bool `json:"echo,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-stop
	Stop []string `json:"stop,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-presence_penalty
	//
	// Defaults to 0 if not specified.
	PresencePenalty int `json:"presence_penalty,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-frequency_penalty
	//
	// Defaults to 0 if not specified.
	FrequencyPenalty int `json:"frequency_penalty,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-best_of
	//
	// Defaults to 1 if not specified.
	//
	// WARNING: Because this parameter generates many completions, it can quickly consume your token quota.
	//          Use carefully and ensure that you have reasonable settings for max_tokens and stop.
	BestOf int `json:"best_of,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-logit_bias
	//
	// Defaults to nil.
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-user
	//
	// Defaults to nil.
	User string `json:"user,omitempty"`
}

// CreateCompletionResponse is the response from a "completion" request to the OpenAI API.
//
// https://platform.openai.com/docs/api-reference/completions/create
type CreateCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Text         string      `json:"text"`
		Index        int         `json:"index"`
		Logprobs     interface{} `json:"logprobs"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateCompletion performs a "completion" request using the OpenAI API.
//
// # Warning
//
// The completions API endpoint received its final update in July 2023 and
// has a different interface than the new [chat completions] endpoint. Instead
// of the input being a list of messages, the input is a freeform text string
// called a prompt.
//
// # Example
//
//	 resp, _ := client.CreateCompletion(ctx, &openai.CreateCompletionRequest{
//		Model: openai.ModelDavinci,
//		Prompt: []string{"Once upon a time"},
//		MaxTokens: 16,
//	 })
//
// Deprecated:  [github.com/picatz/openai.Client.CreateCompletion] is [deprecated] (legacy). Use [github.com/picatz/openai.Client.CreateChat] instead.
//
// https://platform.openai.com/docs/api-reference/completions/create
//
// [deprecated]: https://platform.openai.com/docs/guides/gpt/completions-api
// [chat completions]: https://platform.openai.com/docs/api-reference/chat/create
func (c *Client) CreateCompletion(ctx context.Context, req *CreateCompletionRequest) (*CreateCompletionResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	cResp := &CreateCompletionResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/models/list
type Models struct {
	Object string `json:"object"`
	Data   []struct {
		ID         string `json:"id"`
		Object     string `json:"object"`
		Created    int    `json:"created"`
		OwnedBy    string `json:"owned_by"`
		Permission []struct {
			ID                 string      `json:"id"`
			Object             string      `json:"object"`
			Created            int         `json:"created"`
			AllowCreateEngine  bool        `json:"allow_create_engine"`
			AllowSampling      bool        `json:"allow_sampling"`
			AllowLogprobs      bool        `json:"allow_logprobs"`
			AllowSearchIndices bool        `json:"allow_search_indices"`
			AllowView          bool        `json:"allow_view"`
			AllowFineTuning    bool        `json:"allow_fine_tuning"`
			Organization       string      `json:"organization"`
			Group              interface{} `json:"group"`
			IsBlocking         bool        `json:"is_blocking"`
		} `json:"permission"`
		Root   string      `json:"root"`
		Parent interface{} `json:"parent"`
	} `json:"data"`
}

// ListModels list model identifiers that can be used with the OpenAI API.
//
// # Example
//
//	resp, _ := client.ListModels(ctx)
//
//	for _, model := range resp.Data {
//	   fmt.Println(model.ID)
//	}
//
// https://platform.openai.com/docs/api-reference/models/list
func (c *Client) ListModels(ctx context.Context) (*Models, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/models", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	cResp := &Models{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// CreateEditRequest is the request for a "edit" request to the OpenAI API.
//
// https://platform.openai.com/docs/api-reference/edits/create
type CreateEditRequest struct {
	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-model
	//
	// Required.
	Model string `json:"model"`

	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-instruction
	//
	// Required.
	Instruction string `json:"instruction"`

	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-input
	Input string `json:"input"`

	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-n
	N int `json:"n,omitempty"`

	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-temperature
	Temperature float64 `json:"temperature,omitempty"`

	// https://platform.openai.com/docs/api-reference/edits/create#edits/create-top-p
	TopP float64 `json:"top_p,omitempty"`
}

// https://platform.openai.com/docs/api-reference/edits/create
type CreateEditResponse struct {
	Object  string `json:"object"`
	Created int    `json:"created"`
	Choices []struct {
		Text  string `json:"text"`
		Index int    `json:"index"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateEdit performs a "edit" request using the OpenAI API.
//
// # Warning
//
// Users of the Edits API and its associated models (e.g., text-davinci-edit-001 or code-davinci-edit-001)
// will need to migrate to GPT-3.5 Turbo by January 4, 2024.
//
// # Example
//
//	resp, _ := client.CreateEdit(ctx, &CreateEditRequest{
//		Model:       openai.ModelTextDavinciEdit001,
//		Instruction: "Change the word 'test' to 'example'",
//		Input:       "This is a test",
//	})
//
// Deprecated: [github.com/picatz/openai.Client.CreateEdit] is [deprecated] (legacy). Use [github.com/picatz/openai.Client.CreateChat] instead.
//
// https://platform.openai.com/docs/api-reference/edits/create
//
// [deprecated]: https://openai.com/blog/gpt-4-api-general-availability
func (c *Client) CreateEdit(ctx context.Context, req *CreateEditRequest) (*CreateEditResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/edits", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &CreateEditResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/images/create
type CreateImageRequest struct {
	// https://platform.openai.com/docs/api-reference/images/create#images/create-prompt
	//
	// Required. Max of 1,000 characters.
	Prompt string `json:"prompt"`

	// https://platform.openai.com/docs/api-reference/images/create#images-create-model
	//
	// Optional. Defaults to "dall-e-2".
	Model string `json:"model,omitempty"`

	// https://platform.openai.com/docs/api-reference/completions/create#completions/create-n
	//
	// Number of images to generate. Defaults to 1 if not specified. Most be between 1 and 10.
	N int `json:"n,omitempty"`

	// https://platform.openai.com/docs/api-reference/images/create#images/create-size
	//
	// Size of the image to generate. Must be one of 256x256, 512x512, or 1024x1024.
	Size string `json:"size,omitempty"`

	// https://platform.openai.com/docs/api-reference/images/create#images/create-response_format
	//
	// Defaults to "url". The format in which the generated images are returned. Must be one of "url" or "b64_json".
	ResponseFormat string `json:"response_format,omitempty"`

	// https://platform.openai.com/docs/api-reference/images/create#images/create-user
	User string `json:"user,omitempty"`

	// https://platform.openai.com/docs/api-reference/images/create#images-create-quality
	//
	// Optional. Either "standard" or "hd", defaults to "standard".
	Quality string `json:"quality,omitempty"`

	// https://platform.openai.com/docs/api-reference/images/create#images-create-style
	//
	// Optional. Either "vivid" or "natural", defaults to "vivid". Only valid for "dall-e-3" model.
	Style string `json:"style,omitempty"`
}

// CreateImageResponse ...
type CreateImageResponse struct {
	Created int `json:"created"`
	Data    []struct {
		// One of the following: "url" or "b64_json"
		URL     *string `json:"url"`
		B64JSON *string `json:"b64_json"`

		// If there were any prompt revisions made by the API.
		// Use this to refine further.
		RevisedPrompt *string `json:"revised_prompt"`
	} `json:"data"`
}

// CreateImage performs a "image" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.CreateImage(ctx, &openai.CreateImageRequest{
//		Prompt:         "Golang-style gopher mascot wearing an OpenAI t-shirt",
//		N:              1,
//		Size:           "256x256",
//		ResponseFormat: "url",
//	})
//
// https://platform.openai.com/docs/api-reference/images/create
func (c *Client) CreateImage(ctx context.Context, req *CreateImageRequest) (*CreateImageResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/images/generations", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &CreateImageResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil

}

// https://platform.openai.com/docs/api-reference/embeddings
type CreateEmbeddingRequest struct {
	// https://platform.openai.com/docs/api-reference/embeddings/create#embeddings/create-model
	//
	// Required. The text to embed.
	Model string `json:"model"`

	// https://platform.openai.com/docs/api-reference/embeddings/create#embeddings/create-input
	//
	// Required. The text to embed.
	Input string `json:"input"`

	// https://platform.openai.com/docs/api-reference/embeddings/create#embeddings/create-user
	User string `json:"user,omitempty"`
}

// CreateEmbeddingResponse ...
//
// https://platform.openai.com/docs/guides/embeddings/what-are-embeddings
type CreateEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// CreateEmbedding performs a "embedding" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.CreateEmbedding(ctx, &openai.CreateEmbeddingRequest{
//		Model: openai.ModelTextEmbeddingAda002,
//		Input: "The food was delicious and the waiter...",
//	})
//
// https://platform.openai.com/docs/api-reference/embeddings
func (c *Client) CreateEmbedding(ctx context.Context, req *CreateEmbeddingRequest) (*CreateEmbeddingResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/embeddings", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &CreateEmbeddingResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/moderations/create
type CreateModerationRequest struct {
	// https://platform.openai.com/docs/api-reference/moderations/create#moderations/create-model
	//
	// Optional. The model to use for moderation. Defaults to "text-moderation-latest".
	Model string `json:"model"`

	// https://platform.openai.com/docs/api-reference/moderations/create#moderations/create-input
	//
	// Required. The text to moderate.
	Input string `json:"input"`
}

// CreateModerationResponse ...
//
// https://platform.openai.com/docs/guides/moderations/what-are-moderations
type CreateModerationResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Results []struct {
		Categories struct {
			Hate            bool `json:"hate"`
			HateThreatening bool `json:"hate/threatening"`
			SelfHarm        bool `json:"self-harm"`
			Sexual          bool `json:"sexual"`
			SexualMinors    bool `json:"sexual/minors"`
			Violence        bool `json:"violence"`
			ViolenceGraphic bool `json:"violence/graphic"`
		} `json:"categories"`
		CategoryScores struct {
			Hate            float64 `json:"hate"`
			HateThreatening float64 `json:"hate/threatening"`
			SelfHarm        float64 `json:"self-harm"`
			Sexual          float64 `json:"sexual"`
			SexualMinors    float64 `json:"sexual/minors"`
			Violence        float64 `json:"violence"`
			ViolenceGraphic float64 `json:"violence/graphic"`
		} `json:"category_scores"`
		Flagged bool `json:"flagged"`
	} `json:"results"`
}

// CreateModeration performs a "moderation" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.CreateModeration(ctx, &openai.CreateModerationRequest{
//		Input: "I want to kill them.",
//	})
//
// https://platform.openai.com/docs/api-reference/moderations
func (c *Client) CreateModeration(ctx context.Context, req *CreateModerationRequest) (*CreateModerationResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/moderations", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &CreateModerationResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/files/list
type ListFilesRequest struct {
	// https://platform.openai.com/docs/api-reference/files/list#files-list-purpose
	//
	// Optional. Filter to only list files with the specified purpose (assistants, fine-tune, etc).
	Purpose string `json:"purpose,omitempty"`
}

// https://platform.openai.com/docs/api-reference/files/list
type ListFilesResponse struct {
	Data []struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Bytes     int    `json:"bytes"`
		CreatedAt int    `json:"created_at"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
	} `json:"data"`
	Object string `json:"object"`
}

// ListFiles performs a "list files" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.ListFiles(ctx, &openai.ListFilesRequest{})
//
// https://platform.openai.com/docs/api-reference/files
func (c *Client) ListFiles(ctx context.Context, req *ListFilesRequest) (*ListFilesResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/files", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &ListFilesResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/files/upload
type UploadFileRequest struct {
	// Name of the JSON Lines file to be uploaded.
	//
	// If the purpose is set to "fine-tune", each line is a JSON
	// record with "prompt" and "completion" fields representing
	// your training examples.
	//
	// Required.
	Name string `json:"name"`

	// Purpose of the uploaded documents.
	//
	// Use "fine-tune" for Fine-tuning. This allows us to validate t
	// the format of the uploaded file.
	//
	// Required.
	Purpose string `json:"purpose"`

	// Body of the file to upload.
	//
	// Required.
	Body io.Reader `json:"file"` // TODO: how to handle this?
}

// UploadFileResponse ...
//
// https://platform.openai.com/docs/api-reference/files/upload
type UploadFileResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int    `json:"bytes"`
	CreatedAt int    `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}

// UploadFile performs a "upload file" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.UploadFile(ctx, &openai.UploadFileRequest{
//		Name:    "fine-tune.jsonl",
//		Purpose: "fine-tune",
//	})
//
// https://platform.openai.com/docs/api-reference/files
func (c *Client) UploadFile(ctx context.Context, req *UploadFileRequest) (*UploadFileResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/files", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	r.Header.Set("Content-Type", "multipart/form-data")

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", req.Name)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(fw, req.Body)
	if err != nil {
		return nil, err
	}

	err = w.WriteField("purpose", req.Purpose)
	if err != nil {
		return nil, err
	}

	err = w.Close()
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(&b)
	r.ContentLength = int64(b.Len())
	r.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &UploadFileResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/files/delete
type DeleteFileRequest struct {
	// ID of the file to delete.
	//
	// Required.
	ID string `json:"id"`
}

// DeleteFileResponse ...
//
// https://platform.openai.com/docs/api-reference/files/delete
type DeleteFileResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// DeleteFile performs a "delete file" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.DeleteFile(ctx, &openai.DeleteFileRequest{
//		ID: "file-123",
//	})
//
// https://platform.openai.com/docs/api-reference/files/delete
func (c *Client) DeleteFile(ctx context.Context, req *DeleteFileRequest) (*DeleteFileResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/files/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &DeleteFileResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/files/retrieve
type GetFileInfoRequest struct {
	// ID of the file to retrieve.
	//
	// Required.
	ID string `json:"id"`
}

// GetFileInfoResponse ...
//
// https://platform.openai.com/docs/api-reference/files/retrieve
type GetFileInfoResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Bytes     int    `json:"bytes"`
	CreatedAt int    `json:"created_at"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
}

// GetFileInfo performs a "get file info (retrieve)" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.GetFileInfo(ctx, &openai.GetFileRequest{
//		ID: "file-123",
//	})
//
// https://platform.openai.com/docs/api-reference/files/retrieve
func (c *Client) GetFileInfo(ctx context.Context, req *GetFileInfoRequest) (*GetFileInfoResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/files/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	cResp := &GetFileInfoResponse{}
	err = json.NewDecoder(resp.Body).Decode(cResp)
	if err != nil {
		return nil, err
	}

	return cResp, nil
}

// https://platform.openai.com/docs/api-reference/files/retrieve-content
type GetFileContentRequest struct {
	// ID of the file to retrieve.
	//
	// Required.
	ID string `json:"id"`
}

// GetFileContentResponse ...
//
// https://platform.openai.com/docs/api-reference/files/retrieve-content
type GetFileContentResponse struct {
	// Body is the file content returned by the OpenAI API.
	//
	// The caller is responsible for closing the body, and should do so as soon as possible.
	Body io.ReadCloser
}

// GetFileContent performs a "get file content (retrieve content)" request using the OpenAI API.
//
// # Example
//
//	resp, _ := c.GetFileContent(ctx, &openai.GetFileContentRequest{
//		ID: "file-123",
//	})
//
// https://platform.openai.com/docs/api-reference/files/retrieve-content
func (c *Client) GetFileContent(ctx context.Context, req *GetFileContentRequest) (*GetFileContentResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/files/"+req.ID+"/contents", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return &GetFileContentResponse{
		Body: resp.Body,
	}, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/create
type CreateFineTuneRequest struct {
	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-training_file
	//
	// Required.
	TrainingFile string `json:"training_file"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-validation_file
	//
	// Optional.
	ValidationFile string `json:"validation_file,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-model
	//
	// Optional. Defaults to "curie".
	Model string `json:"model,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-epochs
	//
	// Optional. Defaults to 4.
	Epochs int `json:"n_epochs,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-batch_size
	//
	// Optional. Defaults to 32.
	BatchSize int `json:"batch_size,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-learning_rate_multiplier
	//
	// Optional. Default depends on the batch size.
	LearningRateMultiplier float64 `json:"learning_rate_multiplier,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-prompt_loss_weight
	//
	// Optional. Defaults to 0.01
	PromptLossWeight float64 `json:"prompt_loss_weight,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-compute_classification_metrics
	//
	// Optional. Defaults to false.
	ComputeClassificationMetrics bool `json:"compute_classification_metrics,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-classification_n_classes
	//
	// Optional, but required for multi-class classification.
	ClassificationNClasses int `json:"classification_n_classes,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-classification_positive_class
	//
	// Optional, but required for binary classification.
	ClassificationPositiveClass string `json:"classification_positive_class,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-classification_betas
	//
	// Optional, only used for binary classification.
	ClassificationBetas []float64 `json:"classification_betas,omitempty"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/create#fine-tunes/create-suffix
	//
	// A string of up to 40 characters that will be added to your fine-tuned model name.
	//
	// For example, a suffix of "custom-model-name" would produce a model name like
	// `ada:ft-your-org:custom-model-name-2022-02-15-04-21-04`.
	//
	// Optional.
	Suffix string `json:"suffix,omitempty"`
}

// CreateFineTuneResponse is the response from a "create fine-tune" request.
//
// https://platform.openai.com/docs/api-reference/fine-tunes/create
type CreateFineTuneResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Model     string `json:"model"`
	CreatedAt int    `json:"created_at"`
	Events    []struct {
		Object    string `json:"object"`
		CreatedAt int    `json:"created_at"`
		Level     string `json:"level"`
		Message   string `json:"message"`
	} `json:"events"`
	FineTunedModel interface{} `json:"fine_tuned_model"`
	Hyperparams    struct {
		BatchSize              int     `json:"batch_size"`
		LearningRateMultiplier float64 `json:"learning_rate_multiplier"`
		NEpochs                int     `json:"n_epochs"`
		PromptLossWeight       float64 `json:"prompt_loss_weight"`
	} `json:"hyperparams"`
	OrganizationID  string        `json:"organization_id"`
	ResultFiles     []interface{} `json:"result_files"`
	Status          string        `json:"status"`
	ValidationFiles []interface{} `json:"validation_files"`
	TrainingFiles   []struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Bytes     int    `json:"bytes"`
		CreatedAt int    `json:"created_at"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
	} `json:"training_files"`
	UpdatedAt int `json:"updated_at"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/create
func (c *Client) CreateFineTune(ctx context.Context, req *CreateFineTuneRequest) (*CreateFineTuneResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/fine-tunes", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateFineTuneResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/list
type ListFineTunesRequest struct {
	// No fields yet.
}

// https://platform.openai.com/docs/api-reference/fine-tunes/list
type ListFineTunesResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID              string         `json:"id"`
		Object          string         `json:"object"`
		Model           string         `json:"model"`
		CreatedAt       int            `json:"created_at"`
		FineTunedModel  any            `json:"fine_tuned_model"`
		Hyperparams     map[string]any `json:"hyperparams"`
		OrganizationID  string         `json:"organization_id"`
		ResultFiles     []any          `json:"result_files"`
		Status          string         `json:"status"`
		ValidationFiles []any          `json:"validation_files"`
		TrainingFiles   []any          `json:"training_files"`
		UpdatedAt       int            `json:"updated_at"`
	} `json:"data"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/list
func (c *Client) ListFineTunes(ctx context.Context, req *ListFineTunesRequest) (*ListFineTunesResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/fine-tunes", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res ListFineTunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/retrieve
type GetFineTuneRequest struct {
	ID string `json:"id"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/retrieve
type GetFineTuneResponse struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Model     string `json:"model"`
	CreatedAt int    `json:"created_at"`
	Events    []struct {
		Object    string `json:"object"`
		CreatedAt int    `json:"created_at"`
		Level     string `json:"level"`
		Message   string `json:"message"`
	} `json:"events"`
	FineTunedModel string `json:"fine_tuned_model"`
	Hyperparams    struct {
		BatchSize              int     `json:"batch_size"`
		LearningRateMultiplier float64 `json:"learning_rate_multiplier"`
		NEpochs                int     `json:"n_epochs"`
		PromptLossWeight       float64 `json:"prompt_loss_weight"`
	} `json:"hyperparams"`
	OrganizationID string `json:"organization_id"`
	ResultFiles    []struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Bytes     int    `json:"bytes"`
		CreatedAt int    `json:"created_at"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
	} `json:"result_files"`
	Status          string `json:"status"`
	ValidationFiles []any  `json:"validation_files"`
	TrainingFiles   []struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Bytes     int    `json:"bytes"`
		CreatedAt int    `json:"created_at"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
	} `json:"training_files"`
	UpdatedAt int `json:"updated_at"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/retrieve
func (c *Client) GetFineTune(ctx context.Context, req *GetFineTuneRequest) (*GetFineTuneResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/fine-tunes/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res GetFineTuneResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/cancel
type CancelFineTuneRequest struct {
	ID string `json:"id"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/cancel
type CancelFineTuneResponse struct {
	ID              string `json:"id"`
	Object          string `json:"object"`
	Model           string `json:"model"`
	CreatedAt       int    `json:"created_at"`
	Events          []any  `json:"events"`
	FineTunedModel  any    `json:"fine_tuned_model"`
	Hyperparams     any    `json:"hyperparams"`
	OrganizationID  string `json:"organization_id"`
	ResultFiles     []any  `json:"result_files"`
	Status          string `json:"status"`
	ValidationFiles []any  `json:"validation_files"`
	TrainingFiles   []struct {
		ID        string `json:"id"`
		Object    string `json:"object"`
		Bytes     int    `json:"bytes"`
		CreatedAt int    `json:"created_at"`
		Filename  string `json:"filename"`
		Purpose   string `json:"purpose"`
	} `json:"training_files"`
	UpdatedAt int `json:"updated_at"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/cancel
func (c *Client) CancelFineTune(ctx context.Context, req *CancelFineTuneRequest) (*CancelFineTuneResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/fine-tunes/"+req.ID+"/cancel", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CancelFineTuneResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/events
type ListFineTuneEventsRequest struct {
	// https://platform.openai.com/docs/api-reference/fine-tunes/events#fine-tunes/events-fine_tune_id
	//
	// Required.
	ID string `json:"id"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/events#fine-tunes/events-stream
	//
	// Optional.
	Stream bool `json:"stream"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/events
type ListFineTuneEventsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string `json:"object"`
		CreatedAt int    `json:"created_at"`
		Level     string `json:"level"`
		Message   string `json:"message"`
	} `json:"data"`

	// https://platform.openai.com/docs/api-reference/fine-tunes/events#fine-tunes/events-stream
	//
	// Only present if stream=true. Up to the caller to close the stream, e.g.: defer res.Stream.Close()
	Stream io.ReadCloser `json:"-"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/events
func (c *Client) ListFineTuneEvents(ctx context.Context, req *ListFineTuneEventsRequest) (*ListFineTuneEventsResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/fine-tunes/"+req.ID+"/events", nil)
	if err != nil {
		return nil, err
	}

	if req.Stream {
		q := r.URL.Query()
		q.Set("stream", "true")
		r.URL.RawQuery = q.Encode()
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res ListFineTuneEventsResponse
	if !req.Stream {
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
	} else {
		res.Stream = resp.Body
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/fine-tunes/delete-model
type DeleteFineTuneModelRequest struct {
	// https://platform.openai.com/docs/api-reference/fine-tunes/delete-model#fine-tunes/delete-model-model
	//
	// Required.
	ID string `json:"model"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/delete-model
type DeleteFineTuneModelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// https://platform.openai.com/docs/api-reference/fine-tunes/delete-model
func (c *Client) DeleteFineTuneModel(ctx context.Context, req *DeleteFineTuneModelRequest) (*DeleteFineTuneModelResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/fine-tunes/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res DeleteFineTuneModelResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// FunctionCallArguments is a map of argument name to value.
type FunctionCallArguments map[string]any

// FunctionCallArgumentValue returns the value of the argument with the given name.
func FunctionCallArgumentValue[T any](name string, args FunctionCallArguments) (T, error) {
	v, ok := args[name].(T)
	if !ok {
		return v, fmt.Errorf("argument %q is a %T not of type %T", name, args[name], v)
	}

	return v, nil
}

// FunctionCall describes a function call.
type FunctionCall struct {
	Name      string                `json:"name"`
	Arguments FunctionCallArguments `json:"arguments"`
}

// Implement custom JSON marhsalling and unmarhsalling to handle
// arguments, which come from a JSON string from the API directly.
//
// We turn this into a map[string]any that is a little easier to work with.
func (f *FunctionCall) UnmarshalJSON(b []byte) error {
	// First, unmarshal into a struct that has a map[string]json.RawMessage
	// for the arguments.
	var tmp struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}

	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}

	// Now, unmarshal the arguments into a map[string]any.
	var args map[string]any
	if err := json.Unmarshal([]byte(tmp.Arguments), &args); err != nil {
		return err
	}

	f.Name = tmp.Name
	f.Arguments = args

	return nil
}

// MarshalJSON marshals the function call into a JSON string.
func (f *FunctionCall) MarshalJSON() ([]byte, error) {
	// Marshal the arguments into a JSON string.
	args, err := json.Marshal(f.Arguments)
	if err != nil {
		return nil, err
	}

	// Marshal the struct with the arguments as a string.
	return json.Marshal(struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	}{
		Name:      f.Name,
		Arguments: string(args),
	})
}

// Function is a logical function that can be called by the model.
type Function struct {
	// Name is the name of the function.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-name
	//
	// Required.
	Name string `json:"name"`

	// Description is a description of the function.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-description
	//
	// Optional.
	Description string `json:"description,omitempty"`

	// Parameters are the arguments to the function.
	//
	// The parameters the functions accepts, described as a JSON Schema object.
	// See the guide for examples, and the JSON Schema reference for documentation
	// about the format.
	//
	// https://json-schema.org/understanding-json-schema/
	//
	// https://platform.openai.com/docs/guides/gpt/function-calling
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-parameters
	//
	// Required.
	Parameters *JSONSchema `json:"parameters,omitempty"`
}

// JSONSchema is a JSON Schema.
//
// https://json-schema.org/understanding-json-schema/reference/index.html
type JSONSchema struct {
	// Type is the type of the schema.
	Type string `json:"type,omitempty"`

	// Description is the description of the schema.
	Description string `json:"description,omitempty"`

	// Properties is the properties of the schema.
	Properties map[string]*JSONSchema `json:"properties,omitempty"`

	// Required is the required properties of the schema.
	Required []string `json:"required,omitempty"`

	// Enum is the enum of the schema.
	Enum []string `json:"enum,omitempty"`

	// Items is the items of the schema.
	Items *JSONSchema `json:"items,omitempty"`

	// AdditionalProperties is the additional properties of the schema.
	AdditionalProperties *JSONSchema `json:"additionalProperties,omitempty"`

	// Ref is the ref of the schema.
	Ref string `json:"$ref,omitempty"`

	// AnyOf is the anyOf of the schema.
	AnyOf []*JSONSchema `json:"anyOf,omitempty"`

	// AllOf is the allOf of the schema.
	AllOf []*JSONSchema `json:"allOf,omitempty"`

	// OneOf is the oneOf of the schema.
	OneOf []*JSONSchema `json:"oneOf,omitempty"`

	// Default is the default of the schema.
	Default any `json:"default,omitempty"`

	// Pattern is the pattern of the schema.
	Pattern string `json:"pattern,omitempty"`

	// MinItems is the minItems of the schema.
	MinItems int `json:"minItems,omitempty"`

	// MaxItems is the maxItems of the schema.
	MaxItems int `json:"maxItems,omitempty"`

	// UniqueItems is the uniqueItems of the schema.
	UniqueItems bool `json:"uniqueItems,omitempty"`

	// MultipleOf is the multipleOf of the schema.
	MultipleOf int `json:"multipleOf,omitempty"`

	// Min is the minimum of the schema.
	Min int `json:"min,omitempty"`

	// Max is the maximum of the schema.
	Max int `json:"max,omitempty"`

	// ExclusiveMin is the exclusiveMinimum of the schema.
	ExclusiveMin bool `json:"exclusiveMinimum,omitempty"`

	// ExclusiveMax is the exclusiveMaximum of the schema.
	ExclusiveMax bool `json:"exclusiveMaximum,omitempty"`
}

type ChatMessage struct {
	// Role is the role of the message, e.g. "user" or "bot".
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-role
	//
	// Required.
	Role string `json:"role"`

	// Content is the text of the message.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-content
	//
	// Optional.
	Content string `json:"content"`

	// Name is the author of this message. It is required if role is function,
	// and it should be the name of the function whose response is in the content.
	//
	// May contain a-z, A-Z, 0-9, and underscores, with a maximum length of 64 characters.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-name
	//
	// Optional.
	Name string `json:"name,omitempty"`

	// FunctionCall the name and arguments of a function that should be called,
	// as generated by the model.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-function_call
	//
	// Optional.
	FunctionCall *FunctionCall `json:"function_call,omitempty"`
}

// FunctionCallControl is an option used to control the behavior of a function call
// in a chat request. It can be used to specify the name of the function to call,
// "none", or "auto" (the default).
//
// https://platform.openai.com/docs/api-reference/chat/create#chat/create-function_call
type FunctionCallControl interface {
	isFunctionCallControl()
}

// FunctionCallControlNone is a function call option that indicates that no function
// should be called.
type FunctionCallControlNone struct{}

func (FunctionCallControlNone) isFunctionCallControl() {}

// MarhsalJSON marshals the function call option into a JSON string.
func (FunctionCallControlNone) MarshalJSON() ([]byte, error) {
	return json.Marshal("none")
}

// FunctionCallControlAuto is a function call option that indicates that the
// function to call should be determined automatically.
type FunctionCallControlAuto struct{}

func (FunctionCallControlAuto) isFunctionCallControl() {}

// MarhsalJSON marshals the function call option into a JSON string.
func (FunctionCallControlAuto) MarshalJSON() ([]byte, error) {
	return json.Marshal("auto")
}

// FunctionCallControlName is a function call option that indicates that the
// function to call should be determined by the given name.
type FunctionCallControlName string

func (FunctionCallControlName) isFunctionCallControl() {}

// MarhsalJSON marshals the function call option into a JSON string.
func (f FunctionCallControlName) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]string{
		"name": string(f),
	})
}

var (
	FunctionCallAuto = FunctionCallControlAuto{}
	FunctionCallNone = FunctionCallControlNone{}
)

func FunctionCallName(name string) FunctionCallControlName {
	return FunctionCallControlName(name)
}

// CreateChatRequest is sent to the API, which will return a chat response.
//
// This is the substrate for that OpenAI chat API, which can be used for
// enabling "chat sessions". The API is designed to be used in a loop,
// where the response from the previous request is typically used as the
// input for the next request, specifcally the `messages` field, which contains
// the current "context window" of the conversation that must be maintained
// by the caller.
//
// This is where the art of building a chat bot comes in, as the caller
// must decide how to manage the context window, e.g. how to maintain
// the long term memory of the conversation; what to include in the next request,
// and what to discard; how to handle the "end of conversation" signal, etc.
//
// To identify similar messages from past "memories", the caller can use the
// embedding API to obtain embeddings for the messages, and then use a similarity
// metric to identify similar messages; cosine similarity is often used, but it is
// not the only option.
//
// https://platform.openai.com/docs/api-reference/chat/create
type CreateChatRequest struct {
	// The model to use for the chat (e.g. "gpt3.5-turbo" or "gpt4").
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-model
	//
	// Required.
	Model string `json:"model,omitempty"`

	// The context window of the conversation, which is a list of messages.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-messages
	//
	// Required.
	Messages []ChatMessage `json:"messages,omitempty"`

	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-temperature
	//
	// Optional.
	Temperature float64 `json:"temperature,omitempty"`

	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-top_p
	//
	// Optional.
	TopP float64 `json:"top_p,omitempty"`

	// The number of responses to return, which is typically 1 (the default).
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-n
	//
	// Optional.
	N int `json:"n,omitempty"`

	// Enable streaming mode, which will return a stream instead of a list of
	// responses. This is useful for longer messages, where the caller can
	// process the response incrementally, instead of waiting for the entire
	// response to be returned.
	//
	// You can use this to enable a fun "typing" effect while the chat bot
	// is generating the response, or start transmitting the response as
	// soon as the first few tokens are available.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-stream
	//
	// Optional.
	Stream bool `json:"stream,omitempty"`

	// Up to 4 sequences where the API will stop generating further tokens.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-stop
	//
	// Optional.
	Stop []string `json:"stop,omitempty"`

	// The maximum number of tokens to generate in the chat completion.
	//
	// The total length of input tokens and generated tokens is limited
	// by the model's context length.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-max_tokens
	//
	// Optional.
	MaxTokens int `json:"max_tokens,omitempty"`

	// Number between -2.0 and 2.0. Positive values penalize new tokens based on whether
	// they appear in the text so far, increasing the model's likelihood to talk about new topics.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-presence_penalty
	//
	// Optional.
	PresencePenalty float64 `json:"presence_penalty,omitempty"`

	// Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing
	// frequency in the text so far, decreasing the model's likelihood to repeat the same line verbatim.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-frequency_penalty
	//
	// Optional.
	FrequencyPenalty float64 `json:"frequency_penalty,omitempty"`

	// Modify the likelihood of specified tokens appearing in the completion.
	//
	// This is a json object that maps tokens (specified by their token ID in the tokenizer)
	// to an associated bias value from -100 to 100. Mathematically, the bias is added to
	// the logits generated by the model prior to sampling. The exact effect will vary per
	// model, but values between -1 and 1 should decrease or increase likelihood of selection;
	// values like -100 or 100 should result in a ban or exclusive selection of the relevant token.
	//
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-logit_bias
	//
	// Optional.
	LogitBias map[string]float64 `json:"logit_bias,omitempty"`

	// A unique identifier representing your end-user, which can help OpenAI to monitor and detect abuse.
	//
	// https://platform.openai.com/docs/guides/safety-best-practices/end-user-ids
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-user
	//
	// Optional.
	User string `json:"user,omitempty"`

	// Functions are the functions that can be called by the model.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-functions
	//
	// Optional.
	Functions []*Function `json:"functions,omitempty"`

	// Controls how the model responds to function calls. "none" means the model does not
	// call a function, and responds to the end-user. "auto" means the model can pick
	// between an end-user or calling a function. Specifying a particular function
	// via {"name":\ "my_function"} forces the model to call that function. "none"
	// is the default when no functions are present. "auto" is the default if
	// functions are present.
	//
	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-function_call
	//
	// Optional.
	FunctionCall FunctionCallControl `json:"function_call,omitempty"`
}

// CreateChatResponse is recieved in response to a chat request.
//
// https://platform.openai.com/docs/api-reference/chat/create
type CreateChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Choices []struct {
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
		Index        int         `json:"index"`
	} `json:"choices"`

	// https://platform.openai.com/docs/api-reference/chat/create#chat/create-stream
	Stream io.ReadCloser `json:"-"`
}

// FirstChoice returns the first choice in the response, or an error if there are no choices.
func (r *CreateChatResponse) FirstChoice() (*ChatMessage, error) {
	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	return &r.Choices[0].Message, nil
}

// RandomChoice returns a random choice in the response, or an error if there are no choices.
func (r *CreateChatResponse) RandomChoice() (*ChatMessage, error) {
	if len(r.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned")
	}

	return &r.Choices[rand.Intn(len(r.Choices))].Message, nil
}

type ChatMessageStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int    `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		// Delta is either for role or content.
		Delta struct {
			Role    *string `json:"role"`
			Content *string `json:"content"`
		} `json:"delta"`
		Index        int `json:"index"`
		FinishReason any `json:"finish_reason"`
	} `json:"choices"`
}

// Content returns the content of the message, or an error if there are no choices.
func (c *ChatMessageStreamChunk) ContentDelta() bool {
	if c == nil {
		return false
	}

	if len(c.Choices) == 0 {
		return false
	}

	return c.Choices[0].Delta.Content != nil
}

// Content returns the content of the message, or an error if there are no choices.
func (c *ChatMessageStreamChunk) FirstChoice() (string, error) {
	if len(c.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	// Check if the delta is for the role.
	if c.Choices[0].Delta.Role != nil {
		return "", fmt.Errorf("delta is for role, not content")
	}

	return *c.Choices[0].Delta.Content, nil
}

// ReadStream reads the stream, applying the callback to each message.
//
// Messages are sent via sever-sent events (SSE).
func (r *CreateChatResponse) ReadStream(ctx context.Context, cb func(*ChatMessageStreamChunk) error) error {
	if r.Stream == nil {
		return fmt.Errorf("no stream")
	}

	// Close the stream when we're done.
	defer r.Stream.Close()

	s := bufio.NewScanner(r.Stream)

	for s.Scan() && ctx.Err() == nil {
		// Get the data from the line.
		data := s.Bytes()

		// Skip empty lines.
		if len(data) == 0 {
			continue
		}

		// Skip comments.
		if data[0] == ':' {
			continue
		}

		// Split the line into fields.
		fields := bytes.SplitN(data, []byte{':'}, 2)

		// Ensure there are two fields.
		if len(fields) != 2 {
			continue
		}

		// Ensure the first field is "data".
		if !bytes.Equal(fields[0], []byte("data")) {
			continue
		}

		// Check if data is [DONE].
		if bytes.Equal(fields[1], []byte("[DONE]")) {
			break
		}

		// Unmarshal the message.
		var chunk ChatMessageStreamChunk

		// Skip if we can't unmarshal.
		if err := json.Unmarshal(fields[1], &chunk); err != nil {
			continue
		}

		// Call the callback.
		if err := cb(&chunk); err != nil {
			return err
		}
	}

	// Check for scanner errors.
	if err := s.Err(); err != nil {
		return err
	}

	// Check for context errors.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}

// CreateChat sends a chat request to the API to obtain a chat response,
// creating a completion for the included chat messages (the conversation
// context and history).
//
// # Example
//
//	var history []openai.ChatMessage{
//	 	{
//	 		Role:    openai.ChatRoleSystem,
//	 		Content: "You are a helpful assistant for this example.",
//	 	},
//	 	{
//	 		Role:    openai.ChatRoleUser,
//	 		Content: "Hello!", // Get input from user.
//	  	},
//	 }
//
//	resp, _ := client.CreateChat(ctx, &openai.CreateChatRequest{
//		Model: openai.ModelGPT35Turbo,
//		Messages: history,
//	})
//
//	fmt.Println(resp.Choices[0].Message.Content)
//	// Hello how may I help you today?
//
//	// Update history, summarize, forget, etc. Then repeat.
//	history = appened(history, resp.Choices[0].Message)
//
// https://platform.openai.com/docs/api-reference/chat/create
func (c *Client) CreateChat(ctx context.Context, req *CreateChatRequest) (*CreateChatResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateChatResponse
	if !req.Stream {
		if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		defer resp.Body.Close()
	} else {
		res.Stream = resp.Body
	}

	return &res, nil
}

type AudioTranscriptableFile interface {
	io.ReadCloser
	Name() string
}

type AudioTranscriptionFileReadCloser struct {
	io.ReadCloser
	name string // Example: "audio.mp3"
}

func (a *AudioTranscriptionFileReadCloser) Name() string {
	return a.name
}

func NewAudioTranscriptableFileFromReadCloser(rc io.ReadCloser, name string) AudioTranscriptableFile {
	return &AudioTranscriptionFileReadCloser{
		ReadCloser: rc,
		name:       name,
	}
}

// AudioTranscriptionFile is a file to be used in a CreateAudioTranscriptionRequest,
// allowing a caller to provide various types of file types.
//
// Only provide one of the fields in this struct.
//
// https://platform.openai.com/docs/api-reference/audio/create#audio/create-file
type AudioTranscriptionFile struct {
	ReadCloser *AudioTranscriptionFileReadCloser

	File *os.File
}

// https://platform.openai.com/docs/api-reference/audio/create
type CreateAudioTranscriptionRequest struct {
	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-file
	//
	// Required.
	File AudioTranscriptableFile

	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-model
	//
	// Required.
	Model string

	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-prompt
	//
	// Optional.
	Prompt string

	// The format of the transcript output, in one of these options: json, text, srt, verbose_json, or vtt.
	//
	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-response_format
	//
	// Optional. Defaults to "json".
	ResponseFormat string

	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-temperature
	//
	// Optional.
	Temperature float64

	// https://platform.openai.com/docs/api-reference/audio/create#audio/create-language
	//
	// Optional.
	Language string
}

// responseFormat returns the intended response format of the transcription.
func (req *CreateAudioTranscriptionRequest) responseFormat() string {
	if req.ResponseFormat == "" {
		return "json"
	}
	return req.ResponseFormat
}

// https://platform.openai.com/docs/api-reference/audio/create
type CreateAudioTranscriptionResponse interface {
	Text() string
}

// https://platform.openai.com/docs/api-reference/audio/create
type CreateAudioTranscriptionResponseJSON struct {
	RawText string `json:"text"`
}

// https://platform.openai.com/docs/api-reference/audio/create
func (a *CreateAudioTranscriptionResponseJSON) Text() string {
	return a.RawText
}

// CreateAudioTranscription transcribes audio into the input language.
//
// https://platform.openai.com/docs/api-reference/audio/create
func (c *Client) CreateAudioTranscription(ctx context.Context, req *CreateAudioTranscriptionRequest) (CreateAudioTranscriptionResponse, error) {
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)

	// Write the file
	fw, err := w.CreateFormFile("file", req.File.Name())
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(fw, req.File); err != nil {
		return nil, err
	}

	// Write the model
	if err := w.WriteField("model", req.Model); err != nil {
		return nil, err
	}

	// Write the prompt
	if req.Prompt != "" {
		if err := w.WriteField("prompt", req.Prompt); err != nil {
			return nil, err
		}
	}

	// Write the response_format
	if req.ResponseFormat != "" {
		if err := w.WriteField("response_format", req.ResponseFormat); err != nil {
			return nil, err
		}
	}

	// Write the temperature
	if req.Temperature != 0 {
		if err := w.WriteField("temperature", strconv.FormatFloat(req.Temperature, 'f', -1, 64)); err != nil {
			return nil, err
		}
	}

	// Write the language
	if req.Language != "" {
		if err := w.WriteField("language", req.Language); err != nil {
			return nil, err
		}
	}

	// Close the writer
	if err := w.Close(); err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/transcriptions", b)
	if err != nil {
		return nil, err
	}

	r.Header.Set("Content-Type", w.FormDataContentType())

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateAudioTranscriptionResponse

	switch req.responseFormat() {
	case "json":
		res = &CreateAudioTranscriptionResponseJSON{}

		err := json.NewDecoder(resp.Body).Decode(res)
		if err != nil {
			return nil, err
		}
	// TODO: support other response formats
	// case "text":
	// 	res = &CreateAudioTranscriptionResponseText{}
	// case "srt":
	// 	res = &AudioTranscriptionResponseSRT{}
	// case "verbose_json":
	// 	res = &AudioTranscriptionResponseVerboseJSON{}
	// case "vtt":
	// 	res = &AudioTranscriptionResponseVTT{}
	default:
		return nil, fmt.Errorf("unknown response format: %s", req.ResponseFormat)
	}

	return res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/create
type CreateAssistantRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-model
	//
	// Required.
	Model string `json:"model"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-instructions
	//
	// Optional.
	Instructions string `json:"instructions,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-name
	//
	// Optional.
	Name string `json:"name,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-description
	//
	// Optional.
	Description string `json:"description,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-tools
	//
	// Optional.
	Tools []map[string]any `json:"tools,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-file_ids
	//
	// Optional.
	FileIDs []string `json:"file_ids,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistant#assistants-createassistant-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/assistants/object
type Assistant struct {
	ID           string           `json:"id"`
	Object       string           `json:"object"`
	Created      int              `json:"created"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Model        string           `json:"model"`
	Instructions string           `json:"instructions"`
	Tools        []map[string]any `json:"tools"`
	FileIDs      []string         `json:"file_ids"`
	Metadata     map[string]any   `json:"metadata"`
}

// https://platform.openai.com/docs/api-reference/assistants/create
type CreateAssistantResponse = Assistant

// https://platform.openai.com/docs/api-reference/assistants/create
func (c *Client) CreateAssistant(ctx context.Context, req *CreateAssistantRequest) (*CreateAssistantResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/assistants", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	r.Header.Set("OpenAI-Beta", "assistants=v1")

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateAssistantResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

type GetAssistantRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/get#assistants/get-id
	//
	// Required.
	ID string `json:"assistant_id"`
}

// https://platform.openai.com/docs/api-reference/assistants/get#assistants/get-id
type GetAssistantResponse = Assistant

// https://platform.openai.com/docs/api-reference/assistants/get#assistants/get-id
func (c *Client) GetAssistant(ctx context.Context, req *GetAssistantRequest) (*GetAssistantResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	r.Header.Set("OpenAI-Beta", "assistants=v1")

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res GetAssistantResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant
type UpdateAssistantRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/update#assistants/update-id
	//
	// Required.
	ID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-model
	//
	// Optional.
	Model string `json:"model,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-name
	//
	// Optional.
	Name string `json:"name,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-description
	//
	// Optional.
	Description string `json:"description,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-instructions
	//
	// Optional.
	Instructions string `json:"instructions,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-tools
	//
	// Optional.
	Tools []map[string]any `json:"tools,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-file_ids
	//
	// Optional.
	FileIDs []string `json:"file_ids,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/modifyAssistant#assistants-modifyassistant-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

func (c *Client) UpdateAssistant(ctx context.Context, req *UpdateAssistantRequest) (*Assistant, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/assistants/"+req.ID, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")

	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	r.Header.Set("OpenAI-Beta", "assistants=v1")

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res Assistant
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/deleteAssistant
type DeleteAssistantRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/delete#assistants/delete-id
	//
	// Required.
	ID string `json:"assistant_id"`
}

func (c *Client) DeleteAssistant(ctx context.Context, req *DeleteAssistantRequest) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/assistants/"+req.ID, nil)
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return nil
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-request
type ListAssistantsRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-limit
	//
	// Optional. Defaults to 20.
	Limit int `json:"limit,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-order
	//
	// Optional. Defaults to "desc".
	Order string `json:"order,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-after
	//
	// Optional.
	After string `json:"after,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-before
	//
	// Optional.
	Before string `json:"before,omitempty"`
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistants#assistants-listassistants-response
type ListAssistantsResponse struct {
	Data []Assistant `json:"data"`
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistants
func (c *Client) ListAssistants(ctx context.Context, req *ListAssistantsRequest) (*ListAssistantsResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	q := r.URL.Query()

	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}

	if req.Order != "" {
		q.Set("order", req.Order)
	}

	if req.After != "" {
		q.Set("after", req.After)
	}

	if req.Before != "" {
		q.Set("before", req.Before)
	}

	r.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res ListAssistantsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/file-object
type AssistantFile struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	Created     int    `json:"created"`
	AssistantID string `json:"assistant_id"`
}

// https://platform.openai.com/docs/api-reference/assistants/createAssistantFile
type CreateAssistantFileRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/createAssistantFile#assistants-createassistantfile-assistant_id
	//
	// Required.
	AssistantID string `json:"assistant_id"`

	// https://platform.openai.com/docs/api-reference/assistants/createAssistantFile#assistants-createassistantfile-file
	//
	// Required.
	FileID string `json:"file"`
}

// https://platform.openai.com/docs/api-reference/assistants/createAssistantFile#assistants-createassistantfile-response
type CreateAssistantFileResponse = AssistantFile

// https://platform.openai.com/docs/api-reference/assistants/createAssistantFile
func (c *Client) CreateAssistantFile(ctx context.Context, req *CreateAssistantFileRequest) (*CreateAssistantFileResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/assistants/"+req.AssistantID+"/files", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateAssistantFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/getAssistantFile
type GetAssistantFileRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/getAssistantFile#assistants-getassistantfile-assistant_id
	//
	// Required.
	AssistantID string `json:"assistant_id"`

	// https://platform.openai.com/docs/api-reference/assistants/getAssistantFile#assistants-getassistantfile-file_id
	//
	// Required.
	FileID string `json:"file_id"`
}

// https://platform.openai.com/docs/api-reference/assistants/getAssistantFile#assistants-getassistantfile-response
type GetAssistantFileResponse = AssistantFile

func (c *Client) GetAssistantFile(ctx context.Context, req *GetAssistantFileRequest) (*GetAssistantFileResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants/"+req.AssistantID+"/files/"+req.FileID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res GetAssistantFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/assistants/deleteAssistantFile
type DeleteAssistantFileRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/deleteAssistantFile#assistants-deleteassistantfile-assistant_id
	//
	// Required.
	AssistantID string `json:"assistant_id"`

	// https://platform.openai.com/docs/api-reference/assistants/deleteAssistantFile#assistants-deleteassistantfile-file_id
	//
	// Required.
	FileID string `json:"file_id"`
}

// https://platform.openai.com/docs/api-reference/assistants/deleteAssistantFile
func (c *Client) DeleteAssistantFile(ctx context.Context, req *DeleteAssistantFileRequest) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/assistants/"+req.AssistantID+"/files/"+req.FileID, nil)
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	r.Header.Set("OpenAI-Beta", "assistants=v1")

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return nil
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles
type ListAssistantFilesRequest struct {
	// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-assistant_id
	//
	// Required.
	AssistantID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-limit
	//
	// Optional. Defaults to 20.
	Limit int `json:"limit,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-order
	//
	// Optional. Defaults to "desc".
	Order string `json:"order,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-after
	//
	// Optional.
	After string `json:"after,omitempty"`

	// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-before
	//
	// Optional.
	Before string `json:"before,omitempty"`
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles#assistants-listassistantfiles-response
type ListAssistantFilesResponse struct {
	Data []AssistantFile `json:"data"`
}

// https://platform.openai.com/docs/api-reference/assistants/listAssistantFiles
func (c *Client) ListAssistantFiles(ctx context.Context, req *ListAssistantFilesRequest) (*ListAssistantFilesResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/assistants/"+req.AssistantID+"/files", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	q := r.URL.Query()

	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}

	if req.Order != "" {
		q.Set("order", req.Order)
	}

	if req.After != "" {
		q.Set("after", req.After)
	}

	if req.Before != "" {
		q.Set("before", req.Before)
	}

	r.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res ListAssistantFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/threads/object
type Thread struct {
	ID       string         `json:"id"`
	Object   string         `json:"object"`
	Created  int            `json:"created"`
	Metadata map[string]any `json:"metadata"`
}

// https://platform.openai.com/docs/api-reference/threads/createThread
type CreateThreadRequest struct {
	// https://platform.openai.com/docs/api-reference/threads/createThread#threads-createthread-messages
	//
	// Optional.
	Messages []*ChatMessage `json:"messages,omitempty"`

	// https://platform.openai.com/docs/api-reference/threads/createThread#threads-createthread-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/threads/createThread
type CreateThreadResponse = Thread

// https://platform.openai.com/docs/api-reference/threads/createThread
func (c *Client) CreateThread(ctx context.Context, req *CreateThreadRequest) (*CreateThreadResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res CreateThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/threads/getThread
type GetThreadRequest struct {
	// https://platform.openai.com/docs/api-reference/threads/getThread#threads-getthread-id
	//
	// Required.
	ID string `json:"thread_id"`
}

// https://platform.openai.com/docs/api-reference/threads/getThread#threads-getthread-response
type GetThreadResponse = Thread

func (c *Client) GetThread(ctx context.Context, req *GetThreadRequest) (*GetThreadResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+req.ID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res GetThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/threads/modifyThread
type UpdateThreadRequest struct {
	// https://platform.openai.com/docs/api-reference/threads/modifyThread#threads-modifythread-id
	//
	// Required.
	ID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/threads/modifyThread#threads-modifythread-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

type UpdateThreadResponse = Thread

func (c *Client) UpdateThread(ctx context.Context, req *UpdateThreadRequest) (*UpdateThreadResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPatch, "https://api.openai.com/v1/threads/"+req.ID, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res UpdateThreadResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/threads/deleteThread
type DeleteThreadRequest struct {
	// https://platform.openai.com/docs/api-reference/threads/deleteThread#threads-deletethread-id
	//
	// Required.
	ID string `json:"thread_id"`
}

// https://platform.openai.com/docs/api-reference/threads/deleteThread
func (c *Client) DeleteThread(ctx context.Context, req *DeleteThreadRequest) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodDelete, "https://api.openai.com/v1/threads/"+req.ID, nil)
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return nil
}

// https://platform.openai.com/docs/api-reference/messages/object
type ThreadMessageContent map[string]any

// Text returns the text value from the thread message content, or
// an empty string if the text value is not present.
func (t ThreadMessageContent) Text() string {
	textMap, ok := t["text"].(map[string]any)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%s", textMap["value"])
}

// https://platform.openai.com/docs/api-reference/messages/object
type ThreadMessage struct {
	ID          string                 `json:"id"`
	Object      string                 `json:"object"`
	CreatedAt   int                    `json:"created_at"`
	ThreadID    string                 `json:"thread_id"`
	Role        string                 `json:"role"`
	Content     []ThreadMessageContent `json:"content"`
	AssistantID string                 `json:"assistant_id,omitempty"`
	RunID       string                 `json:"run_id,omitempty"`
	FileIDs     []string               `json:"file_ids,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/messages/createMessage
type CreateMessageRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/createMessage#messages-createmessage-thread_id
	//
	// Required.
	ThreadID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/messages/createMessage#messages-createmessage-role
	//
	// Required.
	Role string `json:"role"`

	// https://platform.openai.com/docs/api-reference/messages/createMessage#messages-createmessage-content
	//
	// Required.
	Content string `json:"content"`

	// https://platform.openai.com/docs/api-reference/messages/createMessage#messages-createmessage-file_ids
	//
	// Optional.
	FileIDs []string `json:"file_ids,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/createMessage#messages-createmessage-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/messages/createMessage
type CreateMessageResponse = ThreadMessage

// https://platform.openai.com/docs/api-reference/messages/createMessage
func (c *Client) CreateMessage(ctx context.Context, req *CreateMessageRequest) (*CreateMessageResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+req.ThreadID+"/messages", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res CreateMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/messages/getMessage
type GetMessageRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/getMessage#messages-getmessage-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/messages/getMessage#messages-getmessage-message_id
	//
	// Required.
	MessageID string `json:"message_id"`
}

// https://platform.openai.com/docs/api-reference/messages/getMessage#messages-getmessage-response
type GetMessageResponse = ThreadMessage

func (c *Client) GetMessage(ctx context.Context, req *GetMessageRequest) (*GetMessageResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/messages/"+req.MessageID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res GetMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/messages/modifyMessage
type UpdateMessageRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/getMessage#messages-getmessage-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/messages/getMessage#messages-getmessage-message_id
	//
	// Required.
	MessageID string `json:"message_id"`

	// https://platform.openai.com/docs/api-reference/messages/modifyMessage#messages-modifymessage-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/messages/modifyMessage#messages-modifymessage-response
type UpdateMessageResponse = ThreadMessage

func (c *Client) UpdateMessage(ctx context.Context, req *UpdateMessageRequest) (*UpdateMessageResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPatch, "https://api.openai.com/v1/messages/"+req.MessageID, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res UpdateMessageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/messages/listMessages
type ListMessagesRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-limit
	//
	// Optional. Defaults to 20.
	Limit int `json:"limit,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-order
	//
	// Optional. Defaults to "desc".
	Order string `json:"order,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-after
	//
	// Optional.
	After string `json:"after,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-before
	//
	// Optional.
	Before string `json:"before,omitempty"`
}

// https://platform.openai.com/docs/api-reference/messages/listMessages#messages-listmessages-response
type ListMessagesResponse struct {
	Data []ThreadMessage `json:"data"`
}

func (c *Client) ListMessages(ctx context.Context, req *ListMessagesRequest) (*ListMessagesResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+req.ThreadID+"/messages", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	q := r.URL.Query()

	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}

	if req.Order != "" {
		q.Set("order", req.Order)
	}

	if req.After != "" {
		q.Set("after", req.After)
	}

	if req.Before != "" {
		q.Set("before", req.Before)
	}

	r.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res ListMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/messages/file-object
type MessageFile struct {
	ID        string `json:"id"`
	Object    string `json:"object"`
	Created   int    `json:"created"`
	MessageID string `json:"message_id"`
}

// https://platform.openai.com/docs/api-reference/messages/getMessageFile
type GetMessageFileRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/getMessageFile#messages-getmessagefile-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/messages/getMessageFile#messages-getmessagefile-message_id
	//
	// Required.
	MessageID string `json:"message_id"`

	// https://platform.openai.com/docs/api-reference/messages/getMessageFile#messages-getmessagefile-file_id
	//
	// Required.
	FileID string `json:"file_id"`
}

// https://platform.openai.com/docs/api-reference/messages/getMessageFile#messages-getmessagefile-response
type GetMessageFileResponse = MessageFile

func (c *Client) GetMessageFile(ctx context.Context, req *GetMessageFileRequest) (*GetMessageFileResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/messages/"+req.MessageID+"/files/"+req.FileID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res GetMessageFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/messages/listMessageFiles
type ListMessageFilesRequest struct {
	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-message_id
	//
	// Required.
	MessageID string `json:"message_id"`

	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-limit
	//
	// Optional. Defaults to 20.
	Limit int `json:"limit,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-order
	//
	// Optional. Defaults to "desc".
	Order string `json:"order,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-after
	//
	// Optional.
	After string `json:"after,omitempty"`

	// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-before
	//
	// Optional.
	Before string `json:"before,omitempty"`
}

// https://platform.openai.com/docs/api-reference/messages/listMessageFiles#messages-listmessagefiles-response
type ListMessageFilesResponse struct {
	Data []MessageFile `json:"data"`
}

func (c *Client) ListMessageFiles(ctx context.Context, req *ListMessageFilesRequest) (*ListMessageFilesResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/messages/"+req.MessageID+"/files", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	q := r.URL.Query()

	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}

	if req.Order != "" {
		q.Set("order", req.Order)
	}

	if req.After != "" {
		q.Set("after", req.After)
	}

	if req.Before != "" {
		q.Set("before", req.Before)
	}

	r.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res ListMessageFilesResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/object
type Run struct {
	ID             string           `json:"id"`
	Object         string           `json:"object"`
	CreatedAt      int              `json:"created_at"`
	ThreadID       string           `json:"thread_id"`
	AssistantID    string           `json:"assistant_id"`
	Status         string           `json:"status"`
	RequiredAction string           `json:"required_action,omitempty"`
	LastError      map[string]any   `json:"last_error,omitempty"`
	ExpiresAt      int              `json:"expires_at"`
	StartedAt      int              `json:"started_at,omitempty"`
	CancelledAt    int              `json:"cancelled_at,omitempty"`
	FailedAt       int              `json:"failed_at,omitempty"`
	CompletedAt    int              `json:"completed_at,omitempty"`
	Model          string           `json:"model"`
	Instructions   string           `json:"instructions"`
	Tools          []map[string]any `json:"tools"`
	FileIDs        []string         `json:"file_ids"`
	Metadata       map[string]any   `json:"metadata"`
}

// https://platform.openai.com/docs/api-reference/runs/createRun
type CreateRunRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-thread_id
	//
	// Required.
	ThreadID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-assistant_id
	//
	// Required.
	AssistantID string `json:"assistant_id"`

	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-model
	//
	// Optional. Defaults to the model associated with the assistant.
	Model string `json:"model,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-instructions
	//
	// Optional. Defaults to the instructions associated with the assistant.
	Instructions string `json:"instructions,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-tools
	//
	// Optional. Defaults to the tools associated with the assistant.
	Tools []map[string]any `json:"tools,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createRun#runs-createrun-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/createRun
type CreateRunResponse = Run

// https://platform.openai.com/docs/api-reference/runs/createRun
func (c *Client) CreateRun(ctx context.Context, req *CreateRunRequest) (*CreateRunResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res CreateRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/object#runs/object-status
type RunStatus = string

const (
	RunStatusQueued         RunStatus = "queued"
	RunStatusInProgress     RunStatus = "in_progress"
	RunStatusRequiresAction RunStatus = "requires_action"
	RunStatusCancelling     RunStatus = "cancelling"
	RunStatusCancelled      RunStatus = "cancelled"
	RunStatusFailed         RunStatus = "failed"
	RunStatusCompleted      RunStatus = "completed"
	RunStatusExpired        RunStatus = "expired"
)

// https://platform.openai.com/docs/api-reference/runs/getRun
type GetRunRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/getRun#runs-getrun-thread_id
	//
	// Required.
	ThreadID string

	// https://platform.openai.com/docs/api-reference/runs/getRun#runs-getrun-run_id
	//
	// Required.
	RunID string
}

// https://platform.openai.com/docs/api-reference/runs/getRun
type GetRunResponse = Run

// https://platform.openai.com/docs/api-reference/runs/getRun
func (c *Client) GetRun(ctx context.Context, req *GetRunRequest) (*GetRunResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res GetRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/modifyRun
type UpdateRunRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/modifyRun#runs-modifyrun-thread_id
	//
	// Required.
	ThreadID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/runs/modifyRun#runs-modifyrun-run_id
	//
	// Required.
	RunID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/runs/modifyRun#runs-modifyrun-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/modifyRun
type UpdateRunResponse = Run

// https://platform.openai.com/docs/api-reference/runs/modifyRun
func (c *Client) UpdateRun(ctx context.Context, req *UpdateRunRequest) (*UpdateRunResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res UpdateRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/listRuns
type ListRunsRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-thread_id
	//
	// Required.
	ThreadID string `json:"thread_id"`

	// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-limit
	//
	// Optional. Defaults to 20.
	Limit int `json:"limit,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-order
	//
	// Optional. Defaults to "desc".
	Order string `json:"order,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-after
	//
	// Optional.
	After string `json:"after,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-before
	//
	// Optional.
	Before string `json:"before,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/listRuns#runs-listruns-response
type ListRunsResponse struct {
	Data []Run `json:"data"`
}

type AssistantToolOutput struct {
	CallID string `json:"tool_call_id,omitempty"`
	Output string `json:"output,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs
type SubmitToolOutputsRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs#runs-submittooloutputs-thread_id
	//
	// Required.
	ThreadID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs#runs-submittooloutputs-run_id
	//
	// Required.
	RunID string `json:"-"`

	// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs#runs-submittooloutputs-tool_id
	//
	// Required.
	ToolOuputs []*AssistantToolOutput `json:"tool_outputs"`
}

// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs
type SubmitToolOutputsResponse = Run

// https://platform.openai.com/docs/api-reference/runs/submitToolOutputs
func (c *Client) SubmitToolOutputs(ctx context.Context, req *SubmitToolOutputsRequest) (*SubmitToolOutputsResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID+"/submit_tool_outputs", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res SubmitToolOutputsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/cancelRun
type CancelRunRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/cancelRun#runs-cancelrun-thread_id
	//
	// Required.
	ThreadID string

	// https://platform.openai.com/docs/api-reference/runs/cancelRun#runs-cancelrun-run_id
	//
	// Required.
	RunID string
}

// https://platform.openai.com/docs/api-reference/runs/cancelRun
func (c *Client) CancelRun(ctx context.Context, req *CancelRunRequest) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID+"/cancel", nil)
	if err != nil {
		return err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return nil
}

// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-thread
type CreateThreadAndRunRequestInitialThreadMessage struct {
	Role     string         `json:"role"`
	Content  string         `json:"content"`
	FilesIDs []string       `json:"file_ids,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type CreateThreadAndRunRequestInitialThread struct {
	Messages []*CreateThreadAndRunRequestInitialThreadMessage `json:"messages,omitempty"`
	Metadata map[string]any                                   `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun
type CreateThreadAndRunRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-assistant_id
	//
	// Required.
	AssistantID string `json:"assistant_id"`

	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-thread
	//
	// Optional.
	Thread *CreateThreadAndRunRequestInitialThread `json:"thread,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-model
	//
	// Optional. Defaults to the model associated with the assistant.
	Model string `json:"model,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-instructions
	//
	// Optional. Defaults to the instructions associated with the assistant.
	Instructions string `json:"instructions,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-tools
	//
	// Optional. Defaults to the tools associated with the assistant.
	Tools []map[string]any `json:"tools,omitempty"`

	// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun#runs-createthreadandrun-metadata
	//
	// Optional.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun
type CreateThreadAndRunResponse = Run

// https://platform.openai.com/docs/api-reference/runs/createThreadAndRun
func (c *Client) CreateThreadAndRun(ctx context.Context, req *CreateThreadAndRunRequest) (*CreateThreadAndRunResponse, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/threads/runs", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	var res CreateThreadAndRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/step-object
type RunStep struct {
	ID          string         `json:"id"`
	Object      string         `json:"object"`
	Created     int            `json:"created"`
	AssistantID string         `json:"assistant_id"`
	ThreadID    string         `json:"thread_id"`
	RunID       string         `json:"run_id"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	StepDetails map[string]any `json:"step_details"`
	LastError   map[string]any `json:"last_error,omitempty"`
	ExpiredAt   int            `json:"expired_at,omitempty"`
	CanceledAt  int            `json:"canceled_at,omitempty"`
	FailedAt    int            `json:"failed_at,omitempty"`
	CompletedAt int            `json:"completed_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// https://platform.openai.com/docs/api-reference/runs/getRunStep
type GetRunStepRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/getRunStep#runs-getrunstep-thread_id
	//
	// Required.
	ThreadID string

	// https://platform.openai.com/docs/api-reference/runs/getRunStep#runs-getrunstep-run_id
	//
	// Required.
	RunID string

	// https://platform.openai.com/docs/api-reference/runs/getRunStep#runs-getrunstep-step_id
	//
	// Required.
	StepID string
}

// https://platform.openai.com/docs/api-reference/runs/getRunStep
type GetRunStepResponse = RunStep

// https://platform.openai.com/docs/api-reference/runs/getRunStep
func (c *Client) GetRunStep(ctx context.Context, req *GetRunStepRequest) (*GetRunStepResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID+"/steps/"+req.StepID, nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res GetRunStepResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/runs/listRunSteps
type ListRunStepsRequest struct {
	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-thread_id
	//
	// Required.
	ThreadID string

	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-run_id
	//
	// Required.
	RunID string

	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-limit
	//
	// Optional. Defaults to 20.
	Limit int

	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-order
	//
	// Optional. Defaults to "desc".
	Order string

	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-after
	//
	// Optional.
	After string

	// https://platform.openai.com/docs/api-reference/runs/listRunSteps#runs-listrunsteps-before
	//
	// Optional.
	Before string
}

// https://platform.openai.com/docs/api-reference/runs/listRunSteps
type ListRunStepsResponse struct {
	Data []RunStep `json:"data"`
}

// https://platform.openai.com/docs/api-reference/runs/listRunSteps
func (c *Client) ListRunSteps(ctx context.Context, req *ListRunStepsRequest) (*ListRunStepsResponse, error) {
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.openai.com/v1/threads/"+req.ThreadID+"/runs/"+req.RunID+"/steps", nil)
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)
	r.Header.Set("OpenAI-Beta", "assistants=v1")

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	q := r.URL.Query()

	if req.Limit != 0 {
		q.Set("limit", strconv.Itoa(req.Limit))
	}

	if req.Order != "" {
		q.Set("order", req.Order)
	}

	if req.After != "" {
		q.Set("after", req.After)
	}

	if req.Before != "" {
		q.Set("before", req.Before)
	}

	r.URL.RawQuery = q.Encode()

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}
	defer resp.Body.Close()

	var res ListRunStepsResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &res, nil
}

// https://platform.openai.com/docs/api-reference/audio/createSpeech
type CreateSpeechRequest struct {
	// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-model
	//
	// Required.
	Model string `json:"model"`

	// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-input
	//
	// Required.
	Input string `json:"input"`

	// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-voice
	//
	// Required.
	Voice string `json:"voice,omitempty"`

	// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-response_format
	//
	// Optional. Defaults to "mp3".
	ResponseFormat string `json:"response_format,omitempty"`

	// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-speed
	//
	// Optional. Defaults to 1.
	Speed float64 `json:"speed,omitempty"`
}

// https://platform.openai.com/docs/api-reference/audio/createSpeech#audio-createspeech-response
func (c *Client) CreateSpeech(ctx context.Context, req *CreateSpeechRequest) (io.ReadCloser, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.openai.com/v1/audio/speech", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("Authorization", "Bearer "+c.APIKey)

	if c.Organization != "" {
		r.Header.Set("OpenAI-Organization", c.Organization)
	}

	resp, err := c.HTTPClient.Do(r)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d: %s: %s", resp.StatusCode, http.StatusText(resp.StatusCode), body)
	}

	return resp.Body, nil
}

// WaitForRun polls the API at the given inter until the run is completed, failed, cancelled, or expired.
//
// It returns nil if the run completed successfully, or an error if the run failed, was cancelled, or expired.
func WaitForRun(ctx context.Context, client *Client, threadID, runID string, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			run, err := client.GetRun(ctx, &GetRunRequest{
				ThreadID: threadID,
				RunID:    runID,
			})
			if err != nil {
				return err
			}

			switch run.Status {
			case RunStatusCompleted:
				return nil
			case RunStatusFailed:
				return fmt.Errorf("run %q failed: %v", runID, run.LastError)
			case RunStatusCancelled:
				return fmt.Errorf("run %q cancelled", runID)
			case RunStatusExpired:
				return fmt.Errorf("run %q expired", runID)
			}
		}
	}
}
