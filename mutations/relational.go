package mutations

import (
	"go/ast"
	"go/token"
)

var relational_operators = map[token.Token][]token.Token{
	token.EQL: []token.Token{token.LSS, token.GTR, token.NEQ, token.LEQ, token.GEQ},
	token.LSS: []token.Token{token.EQL, token.GTR, token.NEQ, token.LEQ, token.GEQ},
	token.GTR: []token.Token{token.EQL, token.LSS, token.NEQ, token.LEQ, token.GEQ},
	token.NEQ: []token.Token{token.EQL, token.LSS, token.GTR, token.LEQ, token.GEQ},
	token.LEQ: []token.Token{token.EQL, token.LSS, token.GTR, token.NEQ, token.GEQ},
	token.GEQ: []token.Token{token.EQL, token.LSS, token.GTR, token.NEQ, token.LEQ},
}

func MutateRelationalOperators(node ast.Node, writeMutation func()) {
	if exp, ok := node.(*ast.BinaryExpr); ok {
		if replacement_operators, ok := relational_operators[exp.Op]; ok {
			for _, new_op := range replacement_operators {
				oldOp := exp.Op
				exp.Op = new_op
				defer func() {
					exp.Op = oldOp
				}()

				writeMutation()
			}
		}
	}
}
