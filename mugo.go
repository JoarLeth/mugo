package main

import (
	//"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/format"
	"go/parser"
	//"go/printer"
	"flag"
	"go/token"
	"io"
	//"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/joarleth/mugo/mutations"
	"github.com/joarleth/mugo/testcase"
)

type Visitor struct {
	file            *ast.File
	fset            *token.FileSet
	origDir         string
	fileName        string
	tmpDir          string
	mutationID      uint
	liveMutationIDs []uint
}

type DeclarationVisitor struct {
	topIdentNames []string
	mutationID    uint
}

func (dv *DeclarationVisitor) Visit(node ast.Node) ast.Visitor {
	if ident, ok := node.(*ast.Ident); ok {
		if stringInSlice(ident.Name, dv.topIdentNames) {
			ident.Name = fmt.Sprintf("%s%s%d", ident.Name, "Mutant", dv.mutationID)
		}
	}
	return dv
}

func (v *Visitor) Visit(node ast.Node) ast.Visitor {
	mutations.MutateRelationalOperators(node, v.writeMutation)
	mutations.MutateLogicalOperators(node, v.writeMutation)
	mutations.MutateArithmeticOperators(node, v.writeMutation)
	mutations.MutateBitwiseOperators(node, v.writeMutation)

	mutations.OmitFunctionCalls(node, v.file, v.writeMutation)
	return v
}

func nodeIndex(stmt ast.Stmt, stmt_list []ast.Stmt) int {
	for i, s := range stmt_list {
		if stmt == s {
			fmt.Printf("Returning node index %v\n", i)
			return i
		}
	}

	return -1
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

var package_name string
var mutation_dir string

func init() {
	flag.StringVar(&package_name, "pkg", "", "The package to mutate")
	flag.StringVar(&mutation_dir, "dst", "mugo", "The destination. The specified directory will be created and used for mutation. This should be a directory on your GOPATH.")
}

func main() {

	flag.Parse()
	if package_name == "" {
		fmt.Println("Must specify package.")
		flag.Usage()
		os.Exit(0)
	}
	//name := "strings"
	pkg, err := build.Import(package_name, "", 0)
	if err != nil {
		fmt.Printf("%v", fmt.Errorf("could not import %s: %s", package_name, err))
	}

	//tmpDir, err := ioutil.TempDir("", "mugo")
	if !path.IsAbs(mutation_dir) {
		working_dir, err := os.Getwd()
		if err != nil {
			fmt.Errorf("%s", err)
			os.Exit(1)
		}
		mutation_dir = filepath.Join(working_dir, mutation_dir)
	}

	mkerr := os.Mkdir(mutation_dir, os.ModePerm)

	if mkerr != nil {
		fmt.Printf("%v", fmt.Errorf("Could not create directory for for mutation: %s", mkerr))
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("%v", fmt.Errorf("could not create temporary directory: %s", err))
	}

	origDir := path.Join(mutation_dir, "original")
	mkerr = os.Mkdir(origDir, os.ModePerm)

	if mkerr != nil {
		fmt.Printf("%v", fmt.Errorf("could not create directory original: %s", mkerr))
	}

	fmt.Fprintf(os.Stderr, "using %s as a temporary directory\n", mutation_dir)
	if err := copyDir(pkg.Dir, origDir); err != nil {
		fmt.Printf("%v", fmt.Errorf("could not copy package directory: %s", err))
	}

	v := &Visitor{origDir: origDir, tmpDir: mutation_dir, mutationID: 1}

	// TODO: Consider using parser.ParseDir instead to porse all go files in a directory.
	for _, f := range pkg.GoFiles {
		v.fileName = f
		mutateFile(v)

		writeTestCases(v)
	}
}

func writeTestCases(v *Visitor) {
	for _, d := range v.file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Name.Name != "main" { // init is not package wide. Each file is allowed one.

				file, fset := testcase.GenerateTestCase(fd, v.liveMutationIDs, v.file.Name.Name)

				extension := filepath.Ext(v.fileName)
				name := strings.TrimRight(v.fileName, extension)

				testCasesPath := path.Join(v.tmpDir, "testcases", name)

				err := os.MkdirAll(testCasesPath, os.ModePerm)

				if err != nil {
					fmt.Printf("%v", fmt.Errorf("could not create directory testcases: %s", err))
				}

				testCaseFileName := fmt.Sprintf("%s_%s_%s%s", name, fd.Name.Name, "mugo_test", extension)

				testCaseFilePath := filepath.Join(testCasesPath, testCaseFileName)

				printAST(testCaseFilePath, fset, file)

				//if fd.Name.Name == "Multiply" {
				symLinkName := filepath.Join(v.origDir, testCaseFileName)

				os.Symlink(testCaseFilePath, symLinkName)
				//}
			}
		}
	}

}

func (v *Visitor) writeMutation() {
	dirName := fmt.Sprintf("mutant%d", v.mutationID)

	/*if err := copyDir(v.origDir, mutDir); err != nil {
		fmt.Printf("%v", fmt.Errorf("could not copy original directory: %s", err))
	}*/

	srcFile := filepath.Join(v.origDir, v.fileName)

	//fset := token.NewFileSet()
	printAST(srcFile, v.fset, v.file)

	var mutDir string

	if killed := runTests(v.origDir, v.mutationID, v); killed {
		mutDir = path.Join(v.tmpDir, "killed", dirName)
		mkerr := os.MkdirAll(mutDir, os.ModePerm)

		if mkerr != nil {
			fmt.Printf("%v", fmt.Errorf("could not create directory for mutation: %s", mkerr))
		}

		copyFile(srcFile, mutDir)
	} else {
		mutDir = path.Join(v.tmpDir, "live", dirName)
		mkerr := os.MkdirAll(mutDir, os.ModePerm)

		if mkerr != nil {
			fmt.Printf("%v", fmt.Errorf("could not create directory for mutation: %s", mkerr))
		}

		mutFile := filepath.Join(mutDir, v.fileName)

		extension := filepath.Ext(v.fileName)
		name := strings.TrimRight(v.fileName, extension)
		mutFileName := fmt.Sprintf("%s%s%06d%s", name, "mutant", v.mutationID, extension)

		symLinkName := filepath.Join(v.origDir, mutFileName)

		// TODO: Fińd out why this causes failed symlinks...
		os.Symlink(mutFile, symLinkName)
		fmt.Println(mutFile)
		fmt.Println(symLinkName)

		copyFile(srcFile, mutDir)
	}

	renameGlobalFuncs(mutDir, v.fileName, v.mutationID)

	v.mutationID++
}

func renameGlobalFuncs(mutDir, fileName string, mutationID uint) {
	dv := DeclarationVisitor{mutationID: mutationID}

	srcFile := filepath.Join(mutDir, fileName)

	fset := token.NewFileSet()

	file, err := parser.ParseFile(fset, srcFile, nil, parser.ParseComments)

	if err != nil {
		fmt.Print(fmt.Errorf("could not parse %s: %s", srcFile, err))
	}

	for _, d := range file.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok {
			if fd.Name.Name != "init" { // init is not package wide. Each file is allowed one.
				dv.topIdentNames = append(dv.topIdentNames, fd.Name.Name)
			}

		} /*else {
			t := reflect.TypeOf(d)
			println(t.Elem().Name())
		}*/

		if gd, ok := d.(*ast.GenDecl); ok {
			if ts, ok := gd.Specs[0].(*ast.TypeSpec); ok {
				ts.Name.Name = fmt.Sprintf("%s%s%d", ts.Name.Name, "Mutant", mutationID)
			} else if vs, ok := gd.Specs[0].(*ast.ValueSpec); ok {
				for i, name := range vs.Names {
					vs.Names[i].Name = fmt.Sprintf("%s%s%d", name.Name, "Mutant", mutationID)
				}
			}
		}
	}

	ast.Walk(&dv, file)

	printAST(srcFile, fset, file)
}

func runTests(dir string, id uint, v *Visitor) (killed bool) {
	args := []string{"test"}
	//args = append(args, testFlags...)
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	co, err := cmd.CombinedOutput()

	fmt.Println(string(co))

	if err == nil {
		message := fmt.Sprintf("Mutant %d still alive", id)
		v.liveMutationIDs = append(v.liveMutationIDs, id)
		fmt.Println(message)

		return false
	} else if _, ok := err.(*exec.ExitError); ok {
		message := fmt.Sprintf("Mutant %d was killed or crashed", id)
		fmt.Println(message)
		return true

	} else {
		fmt.Errorf("\nMutant %d failed to run tests: %s\n", id, err)
		return false
	}
}

func mutateFile(v *Visitor) {
	srcFile := filepath.Join(v.origDir, v.fileName)
	fmt.Printf("%v\n", srcFile)

	v.fset = token.NewFileSet()
	file, err := parser.ParseFile(v.fset, srcFile, nil, parser.ParseComments)
	v.file = file

	if err != nil {
		fmt.Print(fmt.Errorf("could not parse %s: %s", srcFile, err))
	}

	ast.Walk(v, v.file)

	// Restore original file
	printAST(srcFile, v.fset, v.file)
}

func printAST(path string, fset *token.FileSet, node interface{}) error {
	out, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0666)

	if err != nil {
		return fmt.Errorf("could not create file: %s", err)
	}
	defer out.Close()

	// if err := printer.Fprint(out, fset, node); err != nil {
	// 	return fmt.Errorf("could not print %s: %s", path, err)
	// }

	if err := format.Node(out, fset, node); err != nil {
		return fmt.Errorf("could not print %s: %s", path, err)
	}
	return nil
}

// copyDir non-recursively copies the contents of the directory src to the directory dst
func copyDir(src, dst string) error {
	dir, err := os.Open(src)
	if err != nil {
		return err
	}

	contents, err := dir.Readdir(0)
	if err != nil {
		return err
	}

	for _, f := range contents {
		if f.IsDir() || f.Mode()&os.ModeType > 0 {
			continue
		}
		if err := copyFile(filepath.Join(src, f.Name()), dst); err != nil {
			return err
		}
	}

	return nil
}

// copyFile copies the file given by src to the directory dir
func copyFile(src, dir string) error {
	name := filepath.Base(src)
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func MutationID(pos token.Position) string {
	pos.Filename = filepath.Base(pos.Filename)
	return pos.String()
}
