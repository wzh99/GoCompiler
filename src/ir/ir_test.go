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

func test3(n int) int {
	i, j := 1, 1
	for j > n {
		if i % 2 == 0 {
			i++
			j++
		} else {
			i += 3
			j += 3
		}
	}
	return j
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
	ssa := NewSSAOpt()
	ssa.Optimize(irPrg)
	printer.VisitProgram(irPrg)
}
