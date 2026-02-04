package ai

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
	"github.com/yourusername/open-z3950-gateway/pkg/z3950"
)

type Librarian struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewLibrarian(ctx context.Context) (*Librarian, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY not set")
	}

	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-3-pro") // Use the latest model
	return &Librarian{client: client, model: model}, nil
}

func (l *Librarian) GetInsight(ctx context.Context, record *z3950.MARCRecord) (string, error) {
	prompt := fmt.Sprintf(`You are a professional librarian. Analyze the following book metadata and provide a brief insight.
Title: %s
Author: %s
Subjects: %s
Summary: %s

Please provide:
1. A catchy 1-sentence recommendation.
2. A list of 3 reasons why someone should read this book.
3. Suggested target audience.

Keep it concise and professional. Output in Chinese (Simplified).`, 
		record.Title, record.Author, record.Subject, record.Summary)

	resp, err := l.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "AI was unable to generate an insight for this book.", nil
	}

	// Extract text from parts
	var result string
	for _, part := range resp.Candidates[0].Content.Parts {
		result += fmt.Sprintf("%v", part)
	}

	return result, nil
}
