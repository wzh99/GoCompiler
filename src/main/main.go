package main

import (
	"ast"
	"github.com/antlr/antlr4/runtime/Go/antlr"
	"io/ioutil"
	"ir"
	"os"
	"parse"
)

func main() {
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
	irBuilder := ir.NewBuilder()
	irPrg := irBuilder.VisitProgram(asTree).(*ir.Program)
	ssa := ir.NewSSAOpt(&ir.GVNOpt{}, &ir.SCCPOpt{} /*, &ir.PREOpt{}*/)
	ssa.Optimize(irPrg)
	irFile, _ := os.Create("_out.ir")
	printer := ir.NewPrinter(irFile)
	printer.VisitProgram(irPrg)
}
