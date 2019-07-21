package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type Location struct {
	line, col int
}

func NewLocation(token antlr.Token) *Location {
	return &Location{line: token.GetLine(), col: token.GetColumn()}
}

func NewLocationFromContext(ctx antlr.ParserRuleContext) *Location {
	return NewLocation(ctx.GetStart())
}

func NewLocationFromTerminal(node antlr.TerminalNode) *Location {
	return NewLocation(node.GetSymbol())
}

func (l *Location) GetLine() int { return l.line }

func (l *Location) GetColumn() int { return l.col }

func (l *Location) ToString() string {
	return fmt.Sprintf("Line %d, Column %d", l.line, l.col)
}
