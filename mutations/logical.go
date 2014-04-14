package mutations

import (
	"go/ast"
	"go/token"
)

var logical_operators = map[token.Token][]token.Token{
	token.LAND: []token.Token{token.LOR},
	token.LOR:  []token.Token{token.LAND},
}

func MutateLogicalOperators(node ast.Node, writeMutation func()) {
	if exp, ok := node.(*ast.BinaryExpr); ok {
		if replacement_operators, ok := logical_operators[exp.Op]; ok {
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
