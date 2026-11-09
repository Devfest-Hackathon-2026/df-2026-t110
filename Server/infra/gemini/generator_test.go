package gemini

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func testResponse(body string) *http.Response {
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}

func TestGeneratorCountsTokensAndRequestsJSON(t *testing.T) {
	t.Parallel()
	var calls int
	client := &http.Client{Transport: transportFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.URL.Host != "aiplatform.googleapis.com" || request.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL)
		}
		var payload map[string]any
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Errorf("request JSON: %v", err)
		}
		contents, _ := payload["contents"].([]any)
		if len(contents) != 1 {
			t.Errorf("contents = %v", contents)
		} else {
			first, _ := contents[0].(map[string]any)
			parts, _ := first["parts"].([]any)
			if len(parts) == 1 {
				part, _ := parts[0].(map[string]any)
				text, _ := part["text"].(string)
				if !strings.Contains(text, "対戦") || !strings.Contains(text, "collect") || !strings.Contains(text, `{"terrain":[]}`) {
					t.Errorf("prompt omitted context: %s", text)
				}
			}
		}
		if strings.HasSuffix(request.URL.Path, ":countTokens") {
			return testResponse(`{"totalTokens":100}`), nil
		}
		if !strings.HasSuffix(request.URL.Path, ":generateContent") {
			t.Errorf("unexpected API method: %s", request.URL.Path)
		}
		config, _ := payload["generationConfig"].(map[string]any)
		if config["maxOutputTokens"] != float64(836) || config["responseMimeType"] != "application/json" {
			t.Errorf("generation config = %v", config)
		}
		return testResponse(`{"candidates":[{"content":{"parts":[{"text":"ignored","thought":true},{"text":"{\"picolangVersion\":0}"}]}}]}`), nil
	})}
	g, err := NewGenerator(client, "project", "global", "gemini-3.5-flash")
	if err != nil {
		t.Fatal(err)
	}
	got, err := g.GenerateProgram(context.Background(), usecase.ProgramGenerationInput{
		Prompt: "collect", Field: jsontext.Value(`{"terrain":[]}`), IsBattle: true, MaxTokens: 1000,
	})
	if err != nil || string(got) != `{"picolangVersion":0}` || calls != 2 {
		t.Errorf("GenerateProgram = %s, %v; calls = %d", got, err, calls)
	}
}

func TestGeneratorRejectsInputThatExhaustsBudget(t *testing.T) {
	t.Parallel()
	var calls int
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return testResponse(`{"totalTokens":900}`), nil
	})}
	g, err := NewGenerator(client, "project", "asia-northeast1", "gemini-3.5-flash")
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.GenerateProgram(context.Background(), usecase.ProgramGenerationInput{Prompt: "go", MaxTokens: 1000})
	if !errors.Is(err, usecase.ErrInvalidGenerationRequest) || calls != 1 {
		t.Errorf("error = %v; calls = %d", err, calls)
	}
}
