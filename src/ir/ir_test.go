package ir

import (
	"ast"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"os"
	"parse"
	"testing"
)

const source = `
package main

func test3() bool {
	a := 9
	b := 10
	p := a < b || b > 9 && a < 0
	return p
}
`

func TestIRBuild(t *testing.T) {
	input := antlr.NewInputStream(source)
	lexer := parse.NewGolangLexer(input)
	tokens := antlr.NewCommonTokenStream(lexer, antlr.LexerDefaultTokenChannel)
	parser := parse.NewGolangParser(tokens)
	tree := parser.SourceFile()
	visitor := ast.NewASTBuilder()
	asTree := visitor.VisitSourceFile(tree.(*parse.SourceFileContext)).(*ast.ProgramNode)
	sema := ast.NewSemaChecker()
	sema.VisitProgram(asTree)
	irBuilder := NewBuilder()
	irPrg := irBuilder.VisitProgram(asTree).(*Program)
	printer := NewPrinter(os.Stdout)
	printer.VisitProgram(irPrg)
}
