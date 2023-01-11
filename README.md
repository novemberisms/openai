# openai
 
> **Warning**: this is a work in progress Go client implementation for OpenAI's API.

```go
c := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
```

```go
resp, _ := c.CreateCompletion(ctx, &openai.CreateComletionRequest{
	Model:     openai.ModelDavinci,
	Prompt:    []string{"This is a test"},
	MaxTokens: 5,
})

for _, choice := range resp.Choices {
    fmt.Println(choice.Text)
}
```

```go
resp, _ := c.CreateEdit(ctx, &openai.CreateEditRequest{
	Model:       openai.ModelTextDavinciEdit001,
	Instruction: "Change the word 'test' to 'example'",
	Input:       "This is a test",
})

for _, choice := range resp.Choices {
    fmt.Println(choice.Index, choice.Text)
}
```

```go
resp, _ := c.CreateImage(ctx, &openai.CreateImageRequest{
	Prompt:         "Golang-style gopher mascot wearing an OpenAI t-shirt",
	N:              1,
	Size:           "256x256",
	ResponseFormat: "url",
})

fmt.Println(*resp.Data[0].URL)
```