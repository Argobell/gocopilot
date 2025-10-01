package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
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

	// load .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseUrl := os.Getenv("OPENAI_API_BASE_URL")

	// Initialize OpenAI client
	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseUrl),
	)
	if *verbose {
		log.Println("OpenAI client initialized")
	}

	scanner := bufio.NewScanner(os.Stdin)
	getUserMessage := func() (string, bool) {
		if !scanner.Scan() {
			return "", false
		}
		return scanner.Text(), true
	}

	agent := NewAgent(&client, getUserMessage, *verbose)
	err = agent.Run(context.TODO())
	if err != nil {
		fmt.Printf("Error: %s\n", err.Error())
	}

}

func NewAgent(client *openai.Client, getUserMessage func() (string, bool), verbose bool) *Agent {
	return &Agent{
		client:         client,
		getUserMessage: getUserMessage,
		verbose:        verbose,
	}
}

type Agent struct {
	client         *openai.Client
	getUserMessage func() (string, bool)
	verbose        bool
}

func (a *Agent) Run(ctx context.Context) error {
	conversation := []openai.ChatCompletionMessageParamUnion{}

	if a.verbose {
		log.Println("Starting chat session")
	}
	fmt.Println("Chat with gopilot (use 'ctrl-c' to quit)")

	for {
		fmt.Print("\u001b[94mYou\u001b[0m: ")
		userInput, ok := a.getUserMessage()
		if !ok {
			if a.verbose {
				log.Println("User input ended, breaking from chat loop")
			}
			break
		}

		// Skip empty messages
		if userInput == "" {
			if a.verbose {
				log.Println("Skipping empty message")
			}
			continue
		}

		if a.verbose {
			log.Printf("User input received: %q", userInput)
		}

		userMessage := openai.UserMessage(userInput)
		conversation = append(conversation, userMessage)

		if a.verbose {
			log.Printf("Sending message to GPT, conversation length: %d", len(conversation))
		}

		response, err := a.runInference(ctx, conversation)
		if err != nil {
			if a.verbose {
				log.Printf("Error during inference: %v", err)
			}
			return err
		}

		if a.verbose {
			log.Printf("Received response from GPT")
		}

		// Extract assistant's response
		if len(response.Choices) > 0 {
			assistantMessage := response.Choices[0].Message
			fmt.Printf("\u001b[93mGPT\u001b[0m: %s\n", assistantMessage.Content)

			// Add assistant's message to conversation
			conversation = append(conversation, openai.AssistantMessage(assistantMessage.Content))
		}
	}

	if a.verbose {
		log.Println("Chat session ended")
	}
	return nil
}

func (a *Agent) runInference(ctx context.Context, conversation []openai.ChatCompletionMessageParamUnion) (*openai.ChatCompletion, error) {

	model := os.Getenv("MODEL")
	if model == "" {
		model = "gemini-2.5-flash" // default model
	}

	if a.verbose {
		log.Printf("Making API call to OpenAI with model: %s", model)
	}

	response, err := a.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:     model,
		Messages:  conversation,
		MaxTokens: openai.Int(1024),
	})

	if a.verbose {
		if err != nil {
			log.Printf("API call failed: %v", err)
		} else {
			log.Printf("API call successful, response received")
		}
	}

	return response, err
}
