package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	. "parser"
	"testing"
)

const source = `
package main

var a int

type Foo struct {
	bar1 uint
	bar2 float32
}

func foo1(a int) (ret int) {}

func foo2(foo Foo) {}
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
	ast.ToStringTree()
}
