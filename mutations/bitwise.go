package mutations

import (
	"go/ast"
	"go/token"
)

var bitwise_operators = map[token.Token][]token.Token{
	token.AND: []token.Token{token.OR, token.XOR, token.SHL, token.SHR},
	token.OR:  []token.Token{token.AND, token.XOR, token.SHL, token.SHR},
	token.XOR: []token.Token{token.AND, token.OR, token.SHL, token.SHR},
	token.SHL: []token.Token{token.SHR, token.AND, token.OR, token.XOR},
	token.SHR: []token.Token{token.SHL, token.AND, token.OR, token.XOR},
}

func MutateBitwiseOperators(node ast.Node, writeMutation func()) {
	if exp, ok := node.(*ast.BinaryExpr); ok {
		if replacement_operators, ok := bitwise_operators[exp.Op]; ok {
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
