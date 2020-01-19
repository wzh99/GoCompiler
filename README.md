# GoCompiler

## Introduction

A toy compiler written in Go that attempts to compile a subset of [The Go Programming Language](https://golang.org). 

To reduce complexity of implementation, it does not strictly satisfies [the language specification](https://golang.org/ref/spec). 

## Parsing

This project uses [ANTLR](https://www.antlr.org) as the parser generator. The grammar is modified from the one in [antlr/grammars-v4](https://github.com/antlr/grammars-v4/blob/master/golang/GoParser.g4).

##Semantics

The semantic analysis is divided into two passes. In the first pass, the compiler visits the parse tree and construct a immature version of AST. Scope tree and symbol tables are also constructed along with the AST. Some obvious semantic errors are reported. In the second pass, the compiler visited the AST, resolving symbols and check the remaining potential errors. 

Some language features supported in this step:

* Type inference
* Lambdas
* Multiple assignment

## IR

After semantic analysis, the program is converted into mid-level intermediate representation. IR used in this compiler is a CFG IR with register-to-register memory model. The design of IR is inspired by [LLVM](https://www.llvm.org/docs/LangRef.html). 

## Optimization

At present, only machine-independent optimization on IR is implemented. All the optimizations are applied to the SSA form of IR. 

Supported optimizations are listed below, the names in bold type are strong ones, while the others are trivial ones:

* **Global Value Numbering**
* **Sparse Conditional Constant Propagation**
* **Partial Redundancy Eliminination**
* Dead Code Elimination
* Copy Propagation

## Code Generation

To be done...