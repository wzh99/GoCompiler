package ast

import "fmt"

type SemaError struct {
	Loc *Loc
	Msg string
}

func NewSemaError(loc *Loc, msg string) *SemaError {
	return &SemaError{
		Loc: loc,
		Msg: msg,
	}
}

func (e *SemaError) Error() string {
	return fmt.Sprintf("%s: %s", e.Loc.ToString(), e.Msg)
}
