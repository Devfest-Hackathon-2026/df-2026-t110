package picolangbot

// ParseFunc describes the parsing boundary. Implementations validate the
// versioned JSON syntax and source hints before producing a named AST. Duplicate
// function keys must be rejected here, before they are collapsed into a map.
type ParseFunc func(source []byte) (ParsedProgram, error)

// IndexFunc describes variable indexing and name resolution. Implementations
// reject builtin collisions, shadowing, and unbound names.
type IndexFunc func(ParsedProgram) (IndexedProgram, error)

// TypeCheckFunc describes type checking. Implementations check calls, conditions,
// declarations, assignments, returns, and loop control before annotating every
// expression. Only its successful output is intended for VM construction.
type TypeCheckFunc func(IndexedProgram) (TypedProgram, error)
