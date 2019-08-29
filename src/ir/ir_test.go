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
	for i := 0; i < 10; i++ {
		if a < 12 {
			a++
			continue
		} else {
			a--
			break
		}
	}
	return a < 20
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
