package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	. "parser"
	"testing"
)

const source = `
package main

func foo1() (d int) {
	f := Foo{b: g}
	a := 3 + g
	for i := 0; i < 4; i++ {
		if b := i * i; b > 4 {
			a += b
		} else {
			a -= b
		}
		continue
	}
	return a
}

type Foo struct {
	b Bar
}

type Bar int

const g = 4
`

func TestASTBuild(t *testing.T) {
	input := antlr.NewInputStream(source)
	lexer := NewGolangLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.LexerDefaultTokenChannel)
	parser := NewGolangParser(tokens)
	tree := parser.SourceFile()
	fmt.Println(tree.ToStringTree(nil, parser))
	visitor := NewASTBuilder()
	ast := visitor.VisitSourceFile(tree.(*SourceFileContext)).(*ProgramNode)
	sema := NewSemaChecker()
	sema.VisitProgram(ast)
}
