# AST Design


We are going to take this program


```
main :: void () {
    String input = stdio.input("Enter: ")
}
```

and turn it into an AST

```
Keyword "main"
    -> Operator "::"
    -> Keyword "void"
    -> Parenthetical Container "()"
    -> Curly Brace Container
        -> Keyword "String"
        -> Identifier "input"
        -> Operator "="
        -> Identifier "stdio"
        -> Operator "."
        -> Identifier "input"
        -> Parenthetical Container
            -> String Container "Enter: "
```

which further reduces to
```
Function
    -> name: main
    -> returnValues: [void]
    -> parameters: []
    -> instructions: [
        VariableDeclaration
            -> name: input
            -> type: string
            -> valueAssign: [
                externalCall: stdio.input
                    -> parameters: ["Enter: "]
            ]
    ]    
```


fancy version

```
FunctionDecl
    name: main
    returnType: void
    parameters: []
    body: [
        VarDecl
            name: input
            type: String
            value:
                CallExpr
                    callee:
                        MemberExpr
                            object: Identifier "stdio"
                            member: input
                    arguments: [
                        StringLiteral "Enter: "
                    ]
    ]
```