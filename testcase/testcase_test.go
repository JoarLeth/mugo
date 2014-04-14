package testcase

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"testing"
)

func TestGenerateTestCase(t *testing.T) {
	fset := token.NewFileSet()
	srcFile := "/home/joar/go/src/github.com/joarleth/slask/slask.go"
	file, err := parser.ParseFile(fset, srcFile, nil, parser.ParseComments)

	if err != nil {
		t.Error(err)
	}

	mutationIDs := []uint{3, 7}

	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Name.Name == "Join" {
				f, fset := GenerateTestCase(fd, mutationIDs, "slask")

				var buf bytes.Buffer

				printer.Fprint(&buf, fset, f)

				s := buf.String()

				fmt.Printf(s)
				break
			}
		}
	}

}
