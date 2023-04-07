package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"rpc_test/gpt_stream/openai"
)

var (
	token   = flag.String("token", "", "OPENAPI KEY")
	content = flag.String("content", "", "request content")
)

func main() {
	flag.Parse()

	cli := openai.NewClient(*token)
	background := context.Background()

	req := openai.ChatCompletionRequest{
		Model: openai.GPT3Dot5Turbo,
		Messages: []openai.ChatCompletionMessage{{
			Role:    "user",
			Content: *content,
		}},
		Temperature: 0.7,
		TopP:        1,
		MaxTokens:   2048,
		Stream:      true,
	}
	stream, err := cli.CreateChatCompletionStream(background, req)
	if err != nil {
		log.Printf("ChatCompletion failed, %v", err)
		panic(err)
	}
	defer stream.Close()
	var gptAnswers []string
	for {
		rsp, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Printf("stream finished")
			break
		}

		if err != nil {
			log.Printf("Stream error, %v", err)
			panic(err)
		}

		if len(rsp.Choices) == 0 || len(rsp.Choices[0].Delta.Content) == 0 {
			continue
		}
		log.Printf("stream data:%v", rsp.Choices[0].Delta.Content)
		gptAnswers = append(gptAnswers, rsp.Choices[0].Delta.Content)
	}
	log.Printf("complete data:\n%v", gptAnswers)
}
