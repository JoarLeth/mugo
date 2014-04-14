package mutations

import (
	"go/ast"
	"go/token"
)

var arithmetic_operators = map[token.Token][]token.Token{
	token.ADD: []token.Token{token.SUB, token.MUL, token.QUO},
	token.SUB: []token.Token{token.ADD, token.MUL, token.QUO},
	token.MUL: []token.Token{token.QUO, token.ADD, token.SUB},
	token.QUO: []token.Token{token.MUL, token.ADD, token.SUB},
}

func MutateArithmeticOperators(node ast.Node, writeMutation func()) {
	if exp, ok := node.(*ast.BinaryExpr); ok {
		// TODO: If token.ADD: skip if opetands are strings

		if replacement_operators, ok := arithmetic_operators[exp.Op]; ok {
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
