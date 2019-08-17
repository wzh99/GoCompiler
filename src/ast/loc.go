package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
)

type Loc struct {
	line, col int
}

func NewLocFromToken(token antlr.Token) *Loc {
	return &Loc{line: token.GetLine(), col: token.GetColumn()}
}

func NewLocFromContext(ctx antlr.ParserRuleContext) *Loc {
	return NewLocFromToken(ctx.GetStart())
}

func NewLocFromTerminal(node antlr.TerminalNode) *Loc {
	return NewLocFromToken(node.GetSymbol())
}

func (l *Loc) GetLine() int { return l.line }

func (l *Loc) GetColumn() int { return l.col }

func (l *Loc) ToString() string {
	return fmt.Sprintf("line %d, column %d", l.line, l.col)
}
