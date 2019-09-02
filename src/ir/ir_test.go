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

func test3() int {
	i, j, k := 1, 1, 0
	for k < 100 {
		if j < 20 {
			j = i
			k++
		} else {
			j = k
			k += 2
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
