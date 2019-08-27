package ir

import (
	"ast"
	"fmt"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"parse"
	"testing"
)

const source = `
package main

func foo1() {
	k, pi := foo2(1, 2)
	f := func() {
		g := func() {
			e := pi
		}
	}
}

func foo2(i int, j int) (a int, pi float64) {
	return i, 3.14
}
`

func TestIRBuild(t *testing.T) {
	input := antlr.NewInputStream(source)
	lexer := parse.NewGolangLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.LexerDefaultTokenChannel)
	parser := parse.NewGolangParser(tokens)
	tree := parser.SourceFile()
	fmt.Println(tree.ToStringTree(nil, parser))
	visitor := ast.NewASTBuilder()
	asTree := visitor.VisitSourceFile(tree.(*parse.SourceFileContext)).(*ast.ProgramNode)
	sema := ast.NewSemaChecker()
	sema.VisitProgram(asTree)
	irBuilder := NewBuilder()
	irPrg := irBuilder.VisitProgram(asTree).(*Program)
	fmt.Println(irPrg)
}
