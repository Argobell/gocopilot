package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"gocopilot/internal/agent"
	"gocopilot/internal/config"
	"gocopilot/internal/logger"
	"gocopilot/internal/tools"
)

func main() {
    verbose := flag.Bool("verbose", false, "enable verbose logging")
    reasoning := flag.Bool("reasoning", false, "enable multi-step reasoning chain")
    flag.Parse()

	// Load configuration
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
		os.Exit(1)
	}

    cfg := config.Load()
    cfg.Verbose = *verbose
    cfg.ReasoningEnabled = *reasoning

	// Setup logger
	var logLevel logger.Level
	if cfg.Verbose {
		logLevel = logger.LevelDebug
	} else {
		logLevel = logger.LevelInfo
	}
	log := logger.New(logLevel)

	// Initialize OpenAI client
	client := openai.NewClient(
		option.WithAPIKey(cfg.OpenAIAPIKey),
		option.WithBaseURL(cfg.OpenAIBaseURL),
	)

	log.Info("OpenAI client initialized")

	// Initialize tool registry
	toolRegistry := tools.NewRegistry()
	if err := tools.RegisterBuiltinTools(toolRegistry, log); err != nil {
		log.Error("Failed to register built-in tools: %v", err)
		os.Exit(1)
	}

	// Setup user input
	scanner := bufio.NewScanner(os.Stdin)
	inputProvider := &ConsoleInputProvider{scanner: scanner}

	// Setup output handler
	outputHandler := &agent.DefaultOutputHandler{}

	gocopilot := agent.NewAgent(
		&OpenAIClientWrapper{client: &client},
		inputProvider,
		outputHandler,
		toolRegistry,
		cfg,
		log,
	)

	fmt.Println("🤖 [1;36mGocopilot[0m - AI-powered coding assistant")
	fmt.Println("Type your questions or commands below (use 'ctrl-c' to quit)")
	fmt.Println()

	if *reasoning {
		log.Info("Multi-step reasoning mode enabled")
		fmt.Println("[33m🔍 Multi-step reasoning mode enabled[0m")
		fmt.Println("Agent will reason through complex problems step by step")
		fmt.Println()
	}


	if err := gocopilot.Run(context.TODO()); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
		os.Exit(1)
	}
}

// ConsoleInputProvider implements UserInputProvider for console input
type ConsoleInputProvider struct {
	scanner *bufio.Scanner
}

func (c *ConsoleInputProvider) GetUserMessage() (string, bool) {
	fmt.Print("\u001b[1;34m💬 You\u001b[0m: ")
	if !c.scanner.Scan() {
		return "", false
	}
	return c.scanner.Text(), true
}

// OpenAIClientWrapper wraps the OpenAI client to implement InferenceClient
type OpenAIClientWrapper struct {
	client *openai.Client
}

func (w *OpenAIClientWrapper) ChatCompletion(
	ctx context.Context,
	params openai.ChatCompletionNewParams,
) (*openai.ChatCompletion, error) {
	return w.client.Chat.Completions.New(ctx, params)
}
