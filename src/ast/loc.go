package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type Location struct {
	line, col int
}

func NewLocationFromToken(token antlr.Token) *Location {
	return &Location{line: token.GetLine(), col: token.GetColumn()}
}

func NewLocationFromContext(ctx antlr.ParserRuleContext) *Location {
	return NewLocationFromToken(ctx.GetStart())
}

func NewLocationFromTerminal(node antlr.TerminalNode) *Location {
	return NewLocationFromToken(node.GetSymbol())
}

func (l *Location) GetLine() int { return l.line }

func (l *Location) GetColumn() int { return l.col }

func (l *Location) ToString() string {
	return fmt.Sprintf("Line %d, Column %d", l.line, l.col)
}
