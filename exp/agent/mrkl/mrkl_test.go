package mrkl

import (
	"os"
	"strings"
	"testing"

	"github.com/tmc/langchaingo/exp/tools"
	"github.com/tmc/langchaingo/llms/openai"
)

func TestOneShotZeroAgent(t *testing.T) {
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	llm, err := openai.New()
	if err != nil {
		t.Fatal(err)
	}

	testTool := tools.Tool{
		Name:        "Geo",
		Description: "A tool that answers questions about geography",
		Run: func(input string) (string, error) {
			return "Paris", nil
		},
	}

	// You need to provide a list of tools.Tool to initialize the OneShotZeroAgent
	toolsList := []tools.Tool{testTool} // Provide your list of tools here

	opts := map[string]any{
		"verbose":    false,
		"maxRetries": 3,
	}

	agent, err := NewOneShotAgent(llm, toolsList, opts)
	if err != nil {
		t.Fatal(err)
	}

	query := "What is the capital of France?" // Provide your query here
	finish, err := agent.Run(query)
	if err != nil {
		t.Fatal(err)
	}

	answer, ok := finish.ReturnValues["answer"]
	if !ok {
		t.Error("No value in return values answer field")
		return
	}

	result, ok := answer.(string)
	result = strings.TrimSpace(result)
	if !strings.Contains(result, "Paris") {
		t.Errorf("Expected to get Paris. Got %s", result)
	}

	if !ok {
		t.Error("No value in return values answer field")
		return
	}

}
