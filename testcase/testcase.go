package testcase

import (
	//"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	//"go/printer"
	"go/token"
	"strings"
)

type parameters struct {
	list []parameter
}
type parameter struct {
	name string
	t    string
}

func (ps parameters) getCommaSeparatedNames() string {
	var names []string

	for _, p := range ps.list {
		names = append(names, p.name)
	}

	return strings.Join(names, ", ")
}

func (ps parameters) getCommaSeparatedTypes() string {
	var types []string

	for _, p := range ps.list {
		types = append(types, p.t)
	}

	return strings.Join(types, ", ")
}

func (ps parameters) getCommaSeparatedNameTypePairs() string {
	var pairs []string

	for _, p := range ps.list {
		pairs = append(pairs, fmt.Sprintf("%s %s", p.name, p.t))
	}

	return strings.Join(pairs, ", ")
}

func GenerateTestCase(funcDecl *ast.FuncDecl, mutationIDs []uint, packageName string) (*ast.File, *token.FileSet) {
	// funcName := funcDecl.Name.Name

	ps := parameters{}
	//var paramTypes []string

	if funcDecl.Type.Params.NumFields() > 0 {
		for _, field := range funcDecl.Type.Params.List {
			for _, ident := range field.Names {

				if se, ok := field.Type.(*ast.StarExpr); ok {
					ps.list = append(ps.list, parameter{name: ident.Name, t: fmt.Sprintf("*%s", se.X)})
				} else {
					ps.list = append(ps.list, parameter{name: ident.Name, t: fmt.Sprintf("%s", field.Type)})
				}
			}
		}
	}

	var numReturnValues uint
	var returnTypes []string

	if funcDecl.Type.Results.NumFields() > 0 {
		//returnValueNum = 1
		for _, field := range funcDecl.Type.Results.List {
			// ast.NewIdent(name)

			if len(field.Names) == 0 {
				numReturnValues++
				returnTypes = append(returnTypes, fmt.Sprintf("%s", field.Type))
			} else {
				for _, _ = range field.Names {
					numReturnValues++
					returnTypes = append(returnTypes, fmt.Sprintf("%s", field.Type))
				}
			}
		}
	}

	var originalResults, mutatedResults []string

	for i := 1; i <= int(numReturnValues); i++ {
		originalResults = append(originalResults, fmt.Sprintf("origRV%d", i))
		mutatedResults = append(mutatedResults, fmt.Sprintf("mutRV%d", i))
	}

	originalResultVars := strings.Join(originalResults, ", ")
	mutatedResultVars := strings.Join(mutatedResults, ", ")

	originalResultVarsAssignment := originalResultVars + " := "
	mutatedResultVarsAssignment := mutatedResultVars + " := "

	paramTypes := ps.getCommaSeparatedTypes()
	returnTypesString := strings.Join(returnTypes, ", ")

	fmt.Println("Printing params")
	fmt.Println(ps.getCommaSeparatedNameTypePairs())

	paramNameTypePairs := ps.getCommaSeparatedNameTypePairs()

	params := ps.getCommaSeparatedNames()

	returnValueCheckParameters := originalResultVars + ", " + mutatedResultVars + ", " + params

	var returnValuesParamTypePairs string

	for _, resultParam := range originalResults {
		returnValuesParamTypePairs += fmt.Sprintf("%s %s, ", resultParam, returnTypes[0])
	}

	for i, resultParam := range mutatedResults {
		returnValuesParamTypePairs += fmt.Sprintf("%s %s", resultParam, returnTypes[0])

		if i != len(mutatedResults)-1 {
			returnValuesParamTypePairs += ", "
		}
	}

	returnValuesParamTypePairs += ", " + paramNameTypePairs

	var mutationFunctions string

	for _, id := range mutationIDs {
		mutationFunctions += fmt.Sprintf("%d: %sMutant%d,\n", id, funcDecl.Name.Name, id)
	}

	var checkReturnValues string

	for i, _ := range originalResults {
		checkReturnValues += `if !reflect.DeepEqual(` + originalResults[i] + `, ` + mutatedResults[i] + `) {
			mt.mutantKilled("", false)
		}
		`
	}

	src := `package ` + packageName + `

	import (
		"fmt"
		"testing"
		"reflect"
	)

	func (mt *MutationTester) setupCalls() {
		// Please initialize the necessare parameters here.
		// Note that this can be done in a loop

		mt.CallOriginalAndMutants(` + params + `)
	}

	func (mt *MutationTester) checkReturnValues(` + returnValuesParamTypePairs + `) {
		` + checkReturnValues + `
	}

	func Test` + funcDecl.Name.Name + `AgainstLiveMutants(t *testing.T) {
		mutantFuncMap := map[uint]func(` + paramTypes + `) (` + returnTypesString + `){
			0: ` + funcDecl.Name.Name + `,
			` + mutationFunctions + `
		}
		mt := MutationTester{t: t, mutantFuncMap: mutantFuncMap}
		mt.setupCalls()
		mt.printLiveMutants()
	}

	func (mt *MutationTester) mutantKilled(msg string, crashed bool) {
		if mt.mutant == 0 {
			if crashed == true {
				mt.t.Errorf("Original implementation crashed. %s\n", msg)
			} else {
				mt.t.Fatalf("Two separate executions of the original caused different results. This makes it invalid as an oracle. %s", msg)
			}
		} else {
			delete(mt.mutantFuncMap, mt.mutant)
			fmt.Printf("Successfully killed mutant %d. %s\n", mt.mutant, msg)
		}
	}

	func (mt *MutationTester) printLiveMutants() {
		fmt.Println("The following mutants survived:")
		for id, _ := range mt.mutantFuncMap {
			if id != 0 {
				fmt.Printf("\t%d\n", id)
			}
		}
	}

	func (mt *MutationTester) mutantAlive(msg string) {
		if mt.mutant != 0 {
			mt.t.Errorf("Failed to kill mutant %d. %s", mt.mutant, msg)
		}
	}

	type MutationTester struct {
		t      *testing.T
		mutant uint
		mutantFuncMap map[uint]func(` + paramTypes + `) (` + returnTypesString + `)
	}

	

	func (mt *MutationTester) handlePanic() {
		if r := recover(); r != nil {
			mt.mutantKilled(fmt.Sprintf("Crashed: %s", r), true)
		}
	}

	func (mt *MutationTester) CallOriginalAndMutants() {
		defer mt.handlePanic()
		mt.mutant = 0
		` + originalResultVarsAssignment + `mt.mutantFuncMap[0](` + params + `)

		for mutant, fun := range mt.mutantFuncMap {
			func() {
				defer mt.handlePanic()

				mt.mutant = mutant
				` + mutatedResultVarsAssignment + `fun(` + params + `)
				mt.checkReturnValues(` + returnValueCheckParameters + `)
			}()
		}
	}`

	fset := token.NewFileSet()

	file, _ := parser.ParseFile(fset, "", src, parser.ParseComments)

	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Name.Name == "CallOriginalAndMutants" { // init is not package wide. Each file is allowed one.
				fd.Type.Params = funcDecl.Type.Params

				break
			}
		}
	}

	return file, fset
}
