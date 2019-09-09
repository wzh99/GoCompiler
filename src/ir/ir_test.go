package ir

import (
	"ast"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"io/ioutil"
	"os"
	"parse"
	"testing"
)

func TestIRBuild(t *testing.T) {
	source, _ := ioutil.ReadFile("_source.go")
	input := antlr.NewInputStream(string(source))
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
