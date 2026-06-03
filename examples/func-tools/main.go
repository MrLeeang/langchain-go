package main

import (
	"context"
	"fmt"
	"os"

	"github.com/MrLeeang/langchain-go/v2/agents"
	"github.com/MrLeeang/langchain-go/v2/llms"
	"github.com/MrLeeang/langchain-go/v2/tools"
)

type weatherArgs struct {
	City string `json:"city" description:"City name" required:"true"`
}

func getWeather(_ context.Context, args weatherArgs) (string, error) {
	return fmt.Sprintf("Weather in %s: sunny, 25°C", args.City), nil
}

func main() {
	ctx := context.Background()

	llm := llms.NewOpenAIModel(llms.Config{
		BaseURL: "https://api.openai.com/v1",
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4o-mini",
	})

	reg := tools.NewRegistry()
	_ = reg.RegisterFunc(getWeather,
		tools.WithName("get_weather"),
		tools.WithDescription("Get current weather for a city"),
	)

	agent := agents.CreateReactAgent(ctx, llm,
		agents.WithRegistry(reg),
		agents.WithMaxIterations(5),
	).WithPrompt("You are a helpful assistant. Use tools when needed.")

	resp, err := agent.Run("What is the weather in Paris?")
	if err != nil {
		panic(err)
	}
	fmt.Println(resp)
}
