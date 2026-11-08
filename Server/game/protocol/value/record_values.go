package value

// RecordValue is a complete picolang value at the time of recording.
// Existing action-result primitives and terrain observations can be used
// directly. Stacks contain recursively typed values.
type RecordValue interface{ recordValue() }

type RotationRecordValue struct{ Rotation Rotation }
type StackRecordValue struct{ Elements []RecordValue }

func (BoolValue) recordValue()           {}
func (IntegerValue) recordValue()        {}
func (UnitValue) recordValue()           {}
func (LandTerrain) recordValue()         {}
func (RockTerrain) recordValue()         {}
func (SeaTerrain) recordValue()          {}
func (RotationRecordValue) recordValue() {}
func (StackRecordValue) recordValue()    {}
