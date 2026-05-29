# This is my devblog!

Every change/idea i have will be listed here


## 5/28/2026

Today I implemented a simple lexer, it was made using AI. This lexer isn't permanent
but it allows me to move onto the parser and AST. Something which I'm very excited to do.

A simple jist of the planned compiler structure:
```
Lexer
  ↓
Parser
  ↓
AST
  ↓
Name resolution
  ↓
Type checking
  ↓
High-level IR / typed IR
  ↓
Lowering
  ↓
LLVM IR
  ↓
Object file / executable
```

I want to get started on the parser/AST implementation tomorrow, and hopefully be able to write a couple simple programs.