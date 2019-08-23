package ir

type IRError struct {
	Msg string
}

func NewIRError(msg string) *IRError {
	return &IRError{Msg: msg}
}

func (e *IRError) Error() string { return e.Msg }
