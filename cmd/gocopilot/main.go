package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"gocopilot/internal/agent"
	"gocopilot/internal/tools"
)

func main() {
	verbose := flag.Bool("verbose", false, "enable verbose logging")
	flag.Parse()

	if *verbose {
		log.SetOutput(os.Stderr)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println("Verbose logging enabled")
	} else {
		log.SetOutput(os.Stdout)
		log.SetFlags(0)
		log.SetPrefix("")
	}

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_API_BASE_URL")

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	)
	if *verbose {
		log.Println("OpenAI client initialized")
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	toolDefs := []tools.ToolDefinition{
		tools.ReadFileDefinition,
		tools.ListFilesDefinition,
		tools.BashDefinition,
		tools.EditFileDefinition,
		tools.CodeSearchDefinition,
	}
	if *verbose {
		log.Printf("Initialized %d tools", len(toolDefs))
	}

	fmt.Println("Chat with Gocopilot (use 'ctrl-c' to quit)")

	gocopilot := agent.NewAgent(&client, getUserMessage, toolDefs, *verbose)
	if err := gocopilot.Run(context.TODO()); err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}
}
