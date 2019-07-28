package ast

import (
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	. "parser"
	"testing"
)

const source = `
package main

var a, b int

type Foo struct {
	bar1 uint
	bar2 float32
}

func (f Bar) foo1(a float32) (ret int) {
	var k int
	{} 
}

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
