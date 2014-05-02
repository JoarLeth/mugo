package mutations

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	// "reflect"
)

func OmitFunctionCalls(node ast.Node, f *ast.File, writeMutation func()) {
	if ce, ok := node.(*ast.CallExpr); ok {
		var ident *ast.Ident

		if se, ok := ce.Fun.(*ast.SelectorExpr); ok {
			ident = se.Sel
		} else if ident, ok = ce.Fun.(*ast.Ident); ok {
		}

		if ident != nil && ident.Obj != nil {
			if ident.Obj.Decl != nil {
				if fun_decl, ok := ident.Obj.Decl.(*ast.FuncDecl); ok {
					original_func_name := ident.Name

					fmt.Println(ident.Name)

					fake_func_name := original_func_name + "FakeMugoFunction"
					ident.Name = fake_func_name

					fake_func_ident := ast.NewIdent(fake_func_name)

					var numReturnValues uint
					var returnTypes []ast.Expr

					if fun_decl.Type.Results.NumFields() > 0 {
						for _, field := range fun_decl.Type.Results.List {
							if len(field.Names) == 0 {
								numReturnValues++
								returnTypes = append(returnTypes, field.Type)
							} else {
								for _, _ = range field.Names {
									numReturnValues++
									returnTypes = append(returnTypes, field.Type)
								}
							}
						}
					}

					// Anropa originalmetoden i fakemetoden och returnera alla utom ett som ersätts med default value (?)
					var stmt_list []ast.Stmt
					var returnVariables []string

					for i, rt := range returnTypes {
						var returnVariable string

						returnVariable = fmt.Sprintf("rv%d", i+1)
						ident := ast.NewIdent(returnVariable)

						if se, ok := rt.(*ast.StarExpr); ok {
							// Make it a non-star expression
							rt = se.X
							// Return the address of the variable
							returnVariable = fmt.Sprintf("&rv%d", i+1)
						}

						returnVariables = append(returnVariables, returnVariable)

						vs := ast.ValueSpec{Names: []*ast.Ident{ident}, Type: rt}
						gd := ast.GenDecl{Tok: token.VAR, Specs: []ast.Spec{&vs}}

						ds := ast.DeclStmt{Decl: &gd}

						stmt_list = append(stmt_list, &ds)
					}

					var returnExpressions []ast.Expr

					for _, rv := range returnVariables {
						expr, _ := parser.ParseExpr(rv)
						returnExpressions = append(returnExpressions, expr)
					}

					rs := ast.ReturnStmt{Results: returnExpressions}

					stmt_list = append(stmt_list, &rs)

					//stmt_list = append(stmt_list, &params_to_blank_asgn_stmt)
					// Body is a list of statements
					body := ast.BlockStmt{List: stmt_list}

					fake_func_decl := ast.FuncDecl{Name: fake_func_ident, Type: fun_decl.Type, Body: &body, Recv: fun_decl.Recv}

					f.Decls = append(f.Decls, &fake_func_decl)

					defer func() {
						// Return function name of call expression
						ident.Name = original_func_name
						// Remove the inserted function
						d := len(f.Decls) - 1
						f.Decls = append(f.Decls[:d], f.Decls[d+1:]...)
					}()

					writeMutation()
				}
			}
		}
	}
}
