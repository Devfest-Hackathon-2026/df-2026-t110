// Package picolangbot defines the AST stages for a picolang game.Robot.
// Parsing produces ParsedProgram, variable indexing produces IndexedProgram,
// and type checking produces TypedProgram. Execution will consume TypedProgram.
package picolangbot

// HintID identifies the corresponding source node throughout all AST stages.
// UUID validity and program-wide uniqueness are checked before execution.
type HintID string

type Source struct {
	HintID HintID
}

// Name is a source-level variable or function name.
type Name string

// VariableIndex is a slot in a function's variable stack. Parameters occupy
// the first slots; slots for block-local variables can be reused after exit.
type VariableIndex uint

// Variable selects the representation of every variable in an AST stage.
type Variable interface {
	Name | VariableIndex
}

// Annotation distinguishes expressions awaiting type checking from checked ones.
type Annotation interface{ annotation() }

type Untyped struct{}

func (Untyped) annotation() {}

type ParsedProgram = Program[Name, Untyped]
type IndexedProgram = Program[VariableIndex, Untyped]
type TypedProgram = Program[VariableIndex, Type]

// Program uses the same variable representation and expression annotations
// recursively, so nodes from different stages cannot be mixed.
type Program[V Variable, A Annotation] struct {
	Functions map[Name]Function[V, A]
}

type Function[V Variable, A Annotation] struct {
	Source
	Params     []Param[V]
	ReturnType Type
	Body       Block[V, A]
}

type Param[V Variable] struct {
	Source
	Ident Ident[V]
	Type  Type
}

// Ident preserves the identifier's own hint, including in declarations,
// assignments, parameters, and for loops.
type Ident[V Variable] struct {
	Source
	Variable V
}

type Block[V Variable, A Annotation] struct {
	Source
	Stmts []Stmt[V, A]
}

// Stmt and Expr are closed node families. Their marker methods carry both
// stage parameters to prevent cross-stage assignment through interfaces.
type Stmt[V Variable, A Annotation] interface {
	stmt(V, A)
}

type Expr[V Variable, A Annotation] interface {
	Stmt[V, A]
	expr(V, A)
}

type LiteralExpr[V Variable, A Annotation] struct {
	Source
	Annotation A
	Value      Literal
}

type IdentExpr[V Variable, A Annotation] struct {
	Ident[V]
	Annotation A
}

// Args are evaluated from left to right. Functions remain global names;
// only variables are replaced with stack indices during indexing.
type CallExpr[V Variable, A Annotation] struct {
	Source
	Annotation A
	Func       Name
	Args       []Expr[V, A]
}

func (LiteralExpr[V, A]) expr(V, A) {}
func (IdentExpr[V, A]) expr(V, A)   {}
func (CallExpr[V, A]) expr(V, A)    {}
func (LiteralExpr[V, A]) stmt(V, A) {}
func (IdentExpr[V, A]) stmt(V, A)   {}
func (CallExpr[V, A]) stmt(V, A)    {}

type Declaration[V Variable, A Annotation] struct {
	Source
	Type  Type
	Ident Ident[V]
	Value Expr[V, A]
}

type Assignment[V Variable, A Annotation] struct {
	Source
	Ident Ident[V]
	Value Expr[V, A]
}

type If[V Variable, A Annotation] struct {
	Source
	Condition Expr[V, A]
	Then      Block[V, A]
	Else      *Block[V, A] // nil means there is no else clause.
}

type While[V Variable, A Annotation] struct {
	Source
	Condition Expr[V, A]
	Body      Block[V, A]
}

type For[V Variable, A Annotation] struct {
	Source
	Ident Ident[V]
	Start Expr[V, A]
	End   Expr[V, A]
	Body  Block[V, A]
}

type Return[V Variable, A Annotation] struct {
	Source
	Value Expr[V, A]
}

type Break[V Variable, A Annotation] struct{ Source }
type Continue[V Variable, A Annotation] struct{ Source }

func (Declaration[V, A]) stmt(V, A) {}
func (Assignment[V, A]) stmt(V, A)  {}
func (If[V, A]) stmt(V, A)          {}
func (While[V, A]) stmt(V, A)       {}
func (For[V, A]) stmt(V, A)         {}
func (Return[V, A]) stmt(V, A)      {}
func (Break[V, A]) stmt(V, A)       {}
func (Continue[V, A]) stmt(V, A)    {}
