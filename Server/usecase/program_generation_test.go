package usecase_test

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dvpico/Programming-Learning-AR/Server/usecase"
)

const (
	badDraft  = `{"bad":true}`
	goodDraft = `{"ok":true}`
)

// generationFixture implements every dependency of ProgramGeneration so that a
// case can be described by the dependency behavior it needs. The validator
// rejects badDraft with ErrInvalidGeneratedProgram and accepts everything else.
type generationFixture struct {
	// outputs are returned by successive GenerateProgram calls.
	outputs         []jsontext.Value
	validatedOutput jsontext.Value

	quotaErr, generatorErr, validatorErr, fieldErr error

	// Optional hooks run before the corresponding default behavior, letting a
	// case cancel the context or block at a precise point.
	onGenerate func(ctx context.Context)
	onValidate func(ctx context.Context)

	inputs    []usecase.ProgramGenerationInput
	subjects  []string
	deadlines []time.Time
}

func draftOutputs(drafts ...string) []jsontext.Value {
	outputs := make([]jsontext.Value, len(drafts))
	for i, d := range drafts {
		outputs[i] = jsontext.Value(d)
	}
	return outputs
}

func (f *generationFixture) GetGenerationField(_ context.Context, _, _ string) (jsontext.Value, error) {
	return jsontext.Value(`{"stage":1}`), f.fieldErr
}

func (f *generationFixture) Reserve(_ context.Context, subject string, _ int) error {
	f.subjects = append(f.subjects, subject)
	return f.quotaErr
}

func (f *generationFixture) GenerateProgram(ctx context.Context, in usecase.ProgramGenerationInput) (jsontext.Value, error) {
	f.inputs = append(f.inputs, in)
	if deadline, ok := ctx.Deadline(); ok {
		f.deadlines = append(f.deadlines, deadline)
	}
	if f.onGenerate != nil {
		f.onGenerate(ctx)
	}
	if f.generatorErr != nil {
		return nil, f.generatorErr
	}
	if len(f.inputs) > len(f.outputs) {
		return nil, errors.New("fixture has no output for this attempt")
	}
	return f.outputs[len(f.inputs)-1], nil
}

func (f *generationFixture) ValidateGeneratedProgram(ctx context.Context, p jsontext.Value) (jsontext.Value, error) {
	if f.onValidate != nil {
		f.onValidate(ctx)
	}
	if f.validatorErr != nil {
		return nil, f.validatorErr
	}
	if string(p) == badDraft {
		return nil, fmt.Errorf("%w: missing main", usecase.ErrInvalidGeneratedProgram)
	}
	if f.validatedOutput != nil {
		return f.validatedOutput, nil
	}
	return p, nil
}

func (f *generationFixture) deps() usecase.ProgramGenerationDeps {
	return usecase.ProgramGenerationDeps{Fields: f, Quota: f, Generator: f, Validator: f, MaxTokens: 100, Timeout: time.Second}
}

func (f *generationFixture) service() *usecase.ProgramGeneration {
	return usecase.NewProgramGeneration(f.deps())
}

func TestProgramGenerationRetriesInvalidDraft(t *testing.T) {
	t.Parallel()
	f := &generationFixture{outputs: draftOutputs(badDraft, goodDraft)}
	got, err := f.service().Generate(context.Background(), "google-sub", "field-1", "go forward")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if string(got) != goodDraft {
		t.Errorf("output: %s", got)
	}
	if len(f.inputs) != 2 || len(f.subjects) != 2 {
		t.Fatalf("calls: %d, reservations: %d", len(f.inputs), len(f.subjects))
	}
	if f.subjects[0] != "google-sub" || f.subjects[1] != "google-sub" || f.inputs[1].MaxTokens != 100 {
		t.Errorf("quota inputs: %+v %+v", f.subjects, f.inputs[1])
	}
	if string(f.inputs[1].PreviousOutput) != badDraft || f.inputs[1].ValidationError == "" || string(f.inputs[1].Field) != `{"stage":1}` {
		t.Errorf("retry input: %+v", f.inputs[1])
	}
}

func TestProgramGenerationWithoutFieldPreservesBattleMode(t *testing.T) {
	t.Parallel()
	f := &generationFixture{outputs: draftOutputs(goodDraft)}
	got, err := f.service().GenerateWithOptions(context.Background(), "google-sub", usecase.ProgramGenerationRequest{
		Prompt: "collect luggage", IsBattle: true,
	})
	if err != nil || string(got) != goodDraft {
		t.Errorf("GenerateWithOptions = %s, %v", got, err)
	}
	if len(f.inputs) != 1 || !f.inputs[0].IsBattle || string(f.inputs[0].Field) != `{}` {
		t.Errorf("generation input = %+v", f.inputs)
	}
}

func TestProgramGenerationReturnsPreparedProgram(t *testing.T) {
	t.Parallel()
	f := &generationFixture{outputs: draftOutputs(goodDraft), validatedOutput: jsontext.Value(`{"serverAssignedHintIds":true}`)}
	got, err := f.service().Generate(context.Background(), "sub", "field", "go")
	if err != nil || string(got) != string(f.validatedOutput) {
		t.Errorf("Generate = %s, %v; want validator output %s", got, err, f.validatedOutput)
	}
}

func TestProgramGenerationFailures(t *testing.T) {
	t.Parallel()
	serviceErr := errors.New("service failure")
	tests := []struct {
		name                   string
		fixture                func() *generationFixture
		subject, field, prompt string
		want                   error
		calls                  int
	}{
		{"unauthenticated", func() *generationFixture { return &generationFixture{} }, "", "field", "prompt", usecase.ErrUnauthenticated, 0},
		{"empty prompt", func() *generationFixture { return &generationFixture{} }, "sub", "field", " ", usecase.ErrInvalidGenerationRequest, 0},
		{"empty field", func() *generationFixture { return &generationFixture{} }, "sub", " ", "prompt", usecase.ErrInvalidGenerationRequest, 0},
		{"field missing", func() *generationFixture { return &generationFixture{fieldErr: usecase.ErrNotFound} }, "sub", "field", "prompt", usecase.ErrNotFound, 0},
		{"quota", func() *generationFixture {
			return &generationFixture{quotaErr: usecase.ErrGenerationQuotaExceeded}
		}, "sub", "field", "prompt", usecase.ErrGenerationQuotaExceeded, 0},
		{"provider", func() *generationFixture { return &generationFixture{generatorErr: serviceErr} }, "sub", "field", "prompt", serviceErr, 1},
		{"validator failure", func() *generationFixture {
			return &generationFixture{outputs: draftOutputs(`{}`), validatorErr: serviceErr}
		}, "sub", "field", "prompt", serviceErr, 1},
		{"invalid output exhausts attempts", func() *generationFixture {
			return &generationFixture{outputs: draftOutputs(`not json`, badDraft)}
		}, "sub", "field", "prompt", usecase.ErrInvalidGeneratedProgram, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := tt.fixture()
			got, err := f.service().Generate(context.Background(), tt.subject, tt.field, tt.prompt)
			if !errors.Is(err, tt.want) {
				t.Errorf("error = %v, want %v", err, tt.want)
			}
			if got != nil {
				t.Errorf("unexpected output: %s", got)
			}
			if len(f.inputs) != tt.calls {
				t.Errorf("calls = %d, want %d", len(f.inputs), tt.calls)
			}
		})
	}
}

func TestProgramGenerationRequiresConfiguration(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*usecase.ProgramGenerationDeps)
	}{
		{"no fields", func(d *usecase.ProgramGenerationDeps) { d.Fields = nil }},
		{"no quota", func(d *usecase.ProgramGenerationDeps) { d.Quota = nil }},
		{"no generator", func(d *usecase.ProgramGenerationDeps) { d.Generator = nil }},
		{"no validator", func(d *usecase.ProgramGenerationDeps) { d.Validator = nil }},
		{"zero max tokens", func(d *usecase.ProgramGenerationDeps) { d.MaxTokens = 0 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &generationFixture{outputs: draftOutputs(goodDraft)}
			deps := f.deps()
			tt.mutate(&deps)
			got, err := usecase.NewProgramGeneration(deps).Generate(context.Background(), "sub", "field", "prompt")
			if err == nil || got != nil {
				t.Errorf("Generate = %s, %v; want configuration error", got, err)
			}
			if len(f.subjects) != 0 || len(f.inputs) != 0 {
				t.Error("misconfigured service consumed quota or generated")
			}
		})
	}
}

func TestProgramGenerationAttemptLimit(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		maxAttempts int
		wantCalls   int
	}{
		{"default", 0, usecase.DefaultGenerationAttempts},
		{"custom", 3, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := &generationFixture{outputs: draftOutputs(badDraft, badDraft, badDraft, badDraft)}
			deps := f.deps()
			deps.MaxAttempts = tt.maxAttempts
			_, err := usecase.NewProgramGeneration(deps).Generate(context.Background(), "sub", "field", "prompt")
			if !errors.Is(err, usecase.ErrInvalidGeneratedProgram) {
				t.Errorf("error = %v", err)
			}
			if len(f.inputs) != tt.wantCalls || len(f.subjects) != tt.wantCalls {
				t.Errorf("generator calls = %d, reservations = %d, want %d", len(f.inputs), len(f.subjects), tt.wantCalls)
			}
		})
	}
}

func TestProgramGenerationDefaultTimeout(t *testing.T) {
	t.Parallel()
	f := &generationFixture{outputs: draftOutputs(goodDraft)}
	deps := f.deps()
	deps.Timeout = 0
	before := time.Now()
	if _, err := usecase.NewProgramGeneration(deps).Generate(context.Background(), "sub", "field", "prompt"); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(f.deadlines) != 1 {
		t.Fatalf("generator saw %d deadlines, want 1", len(f.deadlines))
	}
	const tolerance = 5 * time.Second
	if remaining := f.deadlines[0].Sub(before); remaining < usecase.DefaultGenerationTimeout-tolerance || remaining > usecase.DefaultGenerationTimeout+tolerance {
		t.Errorf("deadline in %s, want about %s", remaining, usecase.DefaultGenerationTimeout)
	}
}

func TestProgramGenerationCanceled(t *testing.T) {
	t.Parallel()
	f := &generationFixture{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := f.service().Generate(ctx, "sub", "field", "prompt")
	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v", err)
	}
	if len(f.inputs) != 0 || len(f.subjects) != 0 {
		t.Error("canceled request consumed quota or generated")
	}
}

func TestProgramGenerationDiscardsOutputWhenContextEndsMidway(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture func(cancel context.CancelFunc) *generationFixture
	}{
		{"during generation", func(cancel context.CancelFunc) *generationFixture {
			return &generationFixture{outputs: draftOutputs(goodDraft), onGenerate: func(context.Context) { cancel() }}
		}},
		{"after a valid draft is validated", func(cancel context.CancelFunc) *generationFixture {
			return &generationFixture{outputs: draftOutputs(goodDraft), onValidate: func(context.Context) { cancel() }}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			got, err := tt.fixture(cancel).service().Generate(ctx, "sub", "field", "prompt")
			if !errors.Is(err, context.Canceled) || got != nil {
				t.Errorf("Generate = %s, %v; want nil, context.Canceled", got, err)
			}
		})
	}
}

func TestProgramGenerationTimesOutWhileGenerating(t *testing.T) {
	t.Parallel()
	f := &generationFixture{onGenerate: func(ctx context.Context) { <-ctx.Done() }, outputs: draftOutputs(goodDraft)}
	deps := f.deps()
	deps.Timeout = 10 * time.Millisecond
	got, err := usecase.NewProgramGeneration(deps).Generate(context.Background(), "sub", "field", "prompt")
	if !errors.Is(err, context.DeadlineExceeded) || got != nil {
		t.Errorf("Generate = %s, %v; want nil, context.DeadlineExceeded", got, err)
	}
}

// A failed attempt still consumed its reservation, so the retry draws on the
// same budget and is refused once the budget cannot cover another attempt.
func TestProgramGenerationRetryIsRefusedWhenFirstAttemptUsedTheBudget(t *testing.T) {
	t.Parallel()
	f := &generationFixture{outputs: draftOutputs(badDraft)}
	repo := &quotaRepository{usage: make(map[quotaKey]int)}
	quota, err := usecase.NewGenerationTokenQuota(repo, 150, time.Hour, nil)
	if err != nil {
		t.Fatal(err)
	}
	deps := f.deps()
	deps.Quota = quota
	_, err = usecase.NewProgramGeneration(deps).Generate(context.Background(), "alice", "field", "prompt")
	if !errors.Is(err, usecase.ErrGenerationQuotaExceeded) {
		t.Errorf("error = %v", err)
	}
	if len(f.inputs) != 1 {
		t.Errorf("generator calls = %d, want 1", len(f.inputs))
	}
	if used := repo.totalUsage(); used != 100 {
		t.Errorf("tokens used = %d, want 100 for the failed first attempt", used)
	}
}
