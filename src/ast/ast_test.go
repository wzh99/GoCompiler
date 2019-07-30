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
	const k = (5. + 4) / 3 == 2
	c, d := 3, 4
	r := foo2(k)
	{
		const a = 2
		b := foo1(a)
		f := func(x int) {
			a := (c + d) / r
			{
				h := k
			}
			d := b
		}
	} 
	g := k
}

func foo2(foo Foo) int {}
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
