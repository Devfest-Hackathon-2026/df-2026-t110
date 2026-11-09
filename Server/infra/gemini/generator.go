// Package gemini implements Picolang generation using the Vertex AI REST API.
package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

const outputSafetyTokens = 64

type Generator struct {
	client   *http.Client
	endpoint string
}

var _ usecase.ProgramGenerator = (*Generator)(nil)

func NewGenerator(client *http.Client, project, location, model string) (*Generator, error) {
	if client == nil || project == "" || location == "" || model == "" {
		return nil, errors.New("missing Vertex AI generator configuration")
	}
	host := location + "-aiplatform.googleapis.com"
	if location == "global" {
		host = "aiplatform.googleapis.com"
	}
	return &Generator{client: client, endpoint: fmt.Sprintf("https://%s/v1/projects/%s/locations/%s/publishers/google/models/%s", host, url.PathEscape(project), url.PathEscape(location), url.PathEscape(model))}, nil
}

func (g *Generator) GenerateProgram(ctx context.Context, input usecase.ProgramGenerationInput) (json.RawMessage, error) {
	contents := []content{{Role: "user", Parts: []part{{Text: generationPrompt(input)}}}}
	var count struct {
		TotalTokens int `json:"totalTokens"`
	}
	if err := g.post(ctx, "countTokens", struct {
		Contents []content `json:"contents"`
	}{contents}, &count); err != nil {
		return nil, fmt.Errorf("count Gemini tokens: %w", err)
	}
	outputTokens := input.MaxTokens - count.TotalTokens - outputSafetyTokens
	if count.TotalTokens <= 0 || outputTokens < 128 {
		return nil, usecase.ErrInvalidGenerationRequest
	}
	var response struct {
		Candidates []struct {
			Content struct {
				Parts []part `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	request := struct {
		Contents         []content `json:"contents"`
		GenerationConfig any       `json:"generationConfig"`
	}{
		Contents: contents,
		// Function names are dynamic map keys, so a narrow Vertex response schema
		// would discard valid Picolang functions. JSON mode is followed by full
		// server-side parse, name resolution, type checking, and a retry.
		GenerationConfig: struct {
			MaxOutputTokens  int    `json:"maxOutputTokens"`
			ResponseMimeType string `json:"responseMimeType"`
		}{outputTokens, "application/json"},
	}
	if err := g.post(ctx, "generateContent", request, &response); err != nil {
		return nil, fmt.Errorf("generate Gemini program: %w", err)
	}
	if len(response.Candidates) == 0 {
		return nil, errors.New("Gemini returned no program candidate")
	}
	var result strings.Builder
	for _, piece := range response.Candidates[0].Content.Parts {
		if !piece.Thought {
			result.WriteString(piece.Text)
		}
	}
	if result.Len() == 0 {
		return nil, errors.New("Gemini returned an empty program")
	}
	return json.RawMessage(result.String()), nil
}

type part struct {
	Text    string `json:"text"`
	Thought bool   `json:"thought,omitempty"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

func (g *Generator) post(ctx context.Context, method string, payload, output any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint+":"+method, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := g.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Vertex AI %s returned HTTP %d", method, response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(output); err != nil {
		return fmt.Errorf("decode Vertex AI response: %w", err)
	}
	return nil
}

func generationPrompt(input usecase.ProgramGenerationInput) string {
	mode := "教材のゴールへ進むロボット"
	if input.IsBattle {
		mode = "対戦で荷物を運び、地形に応じて行動するロボット"
	}
	var prompt strings.Builder
	fmt.Fprintf(&prompt, "Picolang v0 のプログラムをJSONだけで返してください。用途: %s。\n", mode)
	prompt.WriteString("picolangVersionは0。picolang.v0.programに関数を置く。mainは引数なし・戻り値unitで、bodyの最後にunitをreturnする。全ノードのhintIdは重複しないUUIDにする。\n")
	prompt.WriteString("組み込み関数: forward():bool、back():bool、rotate(rotation):unit、wait(integer):unit、pick_up():bool、release():bool、drill():bool、search_front():terrain、search_now():terrain、is_rock(terrain):bool、is_sea(terrain):bool、is_goal(terrain):bool、distance(distance_target):integer。rotationリテラルは{kind:literal,hintId:UUID,type:rotation,value:leftまたはright}。使う関数だけを呼び、存在しない関数は作らない。\n")
	prompt.WriteString("条件分岐は{kind:if,hintId:UUID,condition:bool式,then:block,else:block}。繰り返しは{kind:while,hintId:UUID,condition:bool式,body:block}。blockは{kind:block,hintId:UUID,stmts:[文...]}。\n")
	prompt.WriteString("例: {\"picolangVersion\":0,\"picolang\":{\"v0\":{\"program\":{\"main\":{\"kind\":\"function\",\"hintId\":\"00000000-0000-4000-8000-000000000001\",\"params\":[],\"returnType\":\"unit\",\"body\":{\"kind\":\"block\",\"hintId\":\"00000000-0000-4000-8000-000000000002\",\"stmts\":[{\"kind\":\"call\",\"hintId\":\"00000000-0000-4000-8000-000000000003\",\"func\":\"forward\",\"args\":[]},{\"kind\":\"return\",\"hintId\":\"00000000-0000-4000-8000-000000000004\",\"value\":{\"kind\":\"literal\",\"hintId\":\"00000000-0000-4000-8000-000000000005\",\"type\":\"unit\",\"value\":null}}]}}}}}}\n")
	fmt.Fprintf(&prompt, "盤面: %s\n依頼: %s\n", input.Field, input.Prompt)
	if len(input.PreviousOutput) > 0 {
		fmt.Fprintf(&prompt, "前回の不正な出力: %s\n検証エラー: %s\n修正して完全なJSONを返してください。", input.PreviousOutput, input.ValidationError)
	}
	return prompt.String()
}
