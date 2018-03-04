package log




//ProcessingState represents log processing state
type ProcessingState struct {
	Line     int
	Position int
}

//Update updates processed position and line number
func (s *ProcessingState) Update(position, lineNumber int) (string, int) {
	s.Line = lineNumber
	s.Position += position
	return "", 0
}

//Reset resets processing state
func (s *ProcessingState) Reset() {
	s.Line = 0
	s.Position = 0
}

