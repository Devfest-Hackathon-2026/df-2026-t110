package value

// Rotation is a relative quarter-turn, matching picolang's rotation type.
type Rotation string

const (
	Left  Rotation = "left"
	Right Rotation = "right"
)
