package picolangbot

import (
	"fmt"
	"slices"
	"strings"
)

type AnalysisStage string

const (
	IndexStage     AnalysisStage = "index"
	TypeCheckStage AnalysisStage = "typecheck"
)

type DiagnosticCode string

const (
	CodeInvalidAST           DiagnosticCode = "invalid_ast"
	CodeInvalidName          DiagnosticCode = "invalid_name"
	CodeFunctionConflict     DiagnosticCode = "function_conflict"
	CodeDuplicateVariable    DiagnosticCode = "duplicate_variable"
	CodeFunctionNameConflict DiagnosticCode = "function_name_conflict"
	CodeUndefinedVariable    DiagnosticCode = "undefined_variable"
	CodeUndefinedFunction    DiagnosticCode = "undefined_function"
	CodeInvalidMain          DiagnosticCode = "invalid_main"
	CodeInvalidType          DiagnosticCode = "invalid_type"
	CodeInvalidVariableIndex DiagnosticCode = "invalid_variable_index"
	CodeTypeMismatch         DiagnosticCode = "type_mismatch"
	CodeArgumentCount        DiagnosticCode = "argument_count"
	CodeMissingReturn        DiagnosticCode = "missing_return"
	CodeInvalidLoopControl   DiagnosticCode = "invalid_loop_control"
	CodeCannotInferType      DiagnosticCode = "cannot_infer_type"
)

// Diagnostic identifies a semantic error without requiring callers to parse
// its message. Expected/Actual describe mismatches; RelatedHintID points to
// an earlier declaration when one is involved in a conflict.
type Diagnostic struct {
	Stage         AnalysisStage
	Code          DiagnosticCode
	Function      Name
	HintID        HintID
	RelatedHintID HintID
	Expected      string
	Actual        string
	Message       string
}

func (d *Diagnostic) Error() string {
	location := string(d.Stage)
	if d.Function != "" {
		location += fmt.Sprintf(" function %q", d.Function)
	}
	if d.HintID != "" {
		location += fmt.Sprintf(" hint %q", d.HintID)
	}
	message := fmt.Sprintf("%s [%s]: %s", location, d.Code, d.Message)
	if d.RelatedHintID != "" {
		message += fmt.Sprintf(" (previous declaration hint %q)", d.RelatedHintID)
	}
	return message
}

// Diagnostics contains errors in deterministic function-name and analysis
// order. errors.As can retrieve this collection or an individual Diagnostic.
type Diagnostics []*Diagnostic

func (ds Diagnostics) Error() string {
	messages := make([]string, len(ds))
	for i, d := range ds {
		messages[i] = d.Error()
	}
	return strings.Join(messages, "\n")
}

func (ds Diagnostics) Unwrap() []error {
	errs := make([]error, len(ds))
	for i, d := range ds {
		errs[i] = d
	}
	return errs
}

// Reporter collects diagnostics for one analysis run. Report records an error;
// Err exposes the complete collection after analysis has finished.
type Reporter struct {
	Stage       AnalysisStage
	Function    Name
	diagnostics Diagnostics
}

func (r *Reporter) diagnostic(source Source, code DiagnosticCode, format string, args ...any) *Diagnostic {
	return &Diagnostic{Stage: r.Stage, Code: code, Function: r.Function, HintID: source.HintID, Message: fmt.Sprintf(format, args...)}
}

func (r *Reporter) Report(source Source, code DiagnosticCode, format string, args ...any) {
	r.diagnostics = append(r.diagnostics, r.diagnostic(source, code, format, args...))
}

func (r *Reporter) Mismatch(source Source, code DiagnosticCode, expected, actual, format string, args ...any) {
	d := r.diagnostic(source, code, format, args...)
	d.Expected, d.Actual = expected, actual
	r.diagnostics = append(r.diagnostics, d)
}

func (r *Reporter) Conflict(source Source, code DiagnosticCode, related HintID, format string, args ...any) {
	d := r.diagnostic(source, code, format, args...)
	d.RelatedHintID = related
	r.diagnostics = append(r.diagnostics, d)
}

func (r *Reporter) Err() error {
	if len(r.diagnostics) == 0 {
		return nil
	}
	slices.SortStableFunc(r.diagnostics, func(a, b *Diagnostic) int {
		return strings.Compare(string(a.Function), string(b.Function))
	})
	return r.diagnostics
}
