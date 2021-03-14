package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gobwas/glob"
	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/imports"
)

var version = "0.0.0-dev"

func main() {

	var ver bool
	var help bool
	var h bool
	var tpl string
	var doTpl bool
	var dst string
	var skips string
	var suffix string
	var delimLeft string
	var delimRight string
	flag.BoolVar(&ver, "version", false, "show version")
	flag.BoolVar(&help, "help", false, "show help")
	flag.BoolVar(&h, "h", false, "show help")
	flag.BoolVar(&doTpl, "t", true, "render output files using golang template engine")
	flag.StringVar(&skips, "skip", "", "comma separated list of glob to exclude files from the template package")
	flag.StringVar(&tpl, "template", "", "package path to the template package")
	flag.StringVar(&dst, "dst", ".", "package path of the destination package")
	flag.StringVar(&suffix, "suffix", "", "output file suffix")
	flag.StringVar(&delimLeft, "dl", "{{", "left delimiter of the template parser")
	flag.StringVar(&delimRight, "dr", "}}", "right delimiter of the template parser")
	flag.Parse()

	if ver {
		fmt.Printf("%v %v\n", "jenjen", version)
		os.Exit(0)
	}
	if help || h {
		fmt.Printf("%v %v\n", "jenjen", version)
		fmt.Println()
		fmt.Println("golang code generator")
		fmt.Println()
		fmt.Printf("%v [-h|-help|-t|-skip=..|-template=..|-dst=..|-dl=..|-dr=..] [directives] [-]\n", "jenjen")
		fmt.Println()
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("  [directives]")
		fmt.Println("	Directives is a comma separated list of replacements to apply such as [context:]search=>replace(,)+.")
		fmt.Println("	The format of a directive is [context:]search=>replace")
		fmt.Println("	Where search is a case sensitive string matching a type name existing within ast.Ident nodes of the template package.")
		fmt.Println("	Where replace is a valid string value for an ast.Ident node,")
		fmt.Println("	or a dash (-) to signify deletion of the node if it is a function, a method or a type.")
		fmt.Println("	If replace is a fully qualified type path (package/path.type),")
		fmt.Println("	the package path and its type component are identified,")
		fmt.Println("	the package path will be added to the import lists.")
		fmt.Println("	Where context is a case sensitive string matching a type or a function declaration as ast.Decl nodes.")
		fmt.Println("	When context contains a dot, it match a method using the type.method notation.")
		fmt.Println("	Each directive is applied sequentially on the loaded program AST.")
		fmt.Println()
		fmt.Println("  [-]")
		fmt.Println("	do not write on disk, print on stdout")
		fmt.Println()
		fmt.Println("example")
		fmt.Println()
		fmt.Println(`  jenjen -template=github.com/clementauger/jenjen/_examples/mymap - \`)
		fmt.Println(`   "NewMyMap => NewFloat32Map, MyMap => Float32Map, Float32Map:int => float32, Float32Map.Rm:string => bytes.Buffer"`)
		fmt.Println()
		fmt.Println("  This example uses 4 directives:")
		fmt.Println()
		fmt.Println("  - NewMyMap => NewFloat32Map")
		fmt.Println("	Rename the function NewMyMap to NewFloat32Map")
		fmt.Println("  - MyMap => Float32Map")
		fmt.Println("	Rename the type MyMap to Float32Map")
		fmt.Println("  - Float32Map:int => float32")
		fmt.Println("	Within the type Float32Map replace all int by float32")
		fmt.Println("  - Float32Map.Rm:string => bytes.Buffer")
		fmt.Println("	Within the method Float32Map.Rm replace all string by bytes.Buffer")
		os.Exit(0)
	}

	if tpl == "" {
		log.Fatal("-template is required")
	}

	isStdout := false
	directives := []string{}
	for _, a := range flag.Args() {
		if a == "-" {
			isStdout = true
			continue
		}
		directives = append(directives, a)
	}
	srs := parseDirectives(directives)

	var cli string
	for _, a := range os.Args {
		if strings.Contains(a, " ") {
			a = fmt.Sprintf("%q", a)
		}
		cli += a + " "
	}

	var err error
	dst, err = filepath.Abs(dst)
	if err != nil {
		log.Fatal(err)
	}
	dstPkgName := getPkgName(dst)

	var conf loader.Config
	conf.ParserMode = parser.ParseComments
	conf.Import(tpl)
	prog, err := conf.Load()
	if err != nil {
		log.Fatal(err)
	}

	pkg := prog.Package(tpl)
	tplPkgName := filepath.Base(tpl)
	if len(pkg.Files) > 0 {
		tplPkgName = pkg.Files[0].Name.Name
	}

	files := selectFiles(prog, pkg, strings.Split(skips, ","))

	for _, f := range files {
		for _, sr := range srs {
			if sr.replacePkgPath != "" {
				if filepath.Base(sr.replacePkgPath) != sr.replacePkgName {
					astutil.AddNamedImport(prog.Fset, f, sr.replacePkgName, sr.replacePkgPath)
				} else {
					astutil.AddImport(prog.Fset, f, sr.replacePkgPath)
				}
			}

			nodes := selectNodes(sr.searchCtx, f)
			for _, node := range nodes {
				if sr.replace != "-" {
					astutil.Apply(node, rewriteIdent(sr.search, sr.replace), nil)
				}
			}
			for _, node := range nodes {
				if sr.replace == "-" {
					astutil.Apply(node, rmNode(sr.search), nil)
				}
			}

			if dstPkgName != "" {
				f.Name.Name = dstPkgName
			}
		}
	}

	funcMap := map[string]interface{}{
		"env": os.Getenv,
	}
	dataTpl := map[string]interface{}{
		"now":     time.Now().Format(time.RFC850),
		"cli":     cli,
		"version": fmt.Sprintf("%v %v", "jenjen", version),
	}

	for _, f := range files {
		outfp := fmt.Sprintf("jenjen_%v.go", tplPkgName)
		fp := prog.Fset.File(f.Pos()).Name()
		if len(files) > 1 {
			outfp = fmt.Sprintf("jenjen_%v_%v", tplPkgName, filepath.Base(fp))
		}
		outfp = filepath.Join(filepath.Dir(dst), outfp)
		if suffix != "" {
			ext := filepath.Ext(outfp)
			f := strings.TrimSuffix(outfp, ext)
			outfp = fmt.Sprintf("%v%v%v", f, suffix, ext)
		}
		var b bytes.Buffer
		printer.Fprint(&b, prog.Fset, f)
		data := b.Bytes()
		data, err = imports.Process(fp, data, nil)
		if err != nil {
			log.Fatal(err)
		}

		var outTpl *template.Template
		if doTpl {
			outTpl, err = template.New("").
				Delims(delimLeft, delimRight).
				Funcs(funcMap).
				Parse(string(data))
			if err != nil {
				log.Fatal(err)
			}
		}

		if isStdout {
			fmt.Fprintf(os.Stdout, "// source        %v\n", fp)
			fmt.Fprintf(os.Stdout, "// destination   %v\n", outfp)
			if doTpl {
				err = outTpl.Execute(os.Stdout, dataTpl)
			} else {
				_, err = os.Stdout.Write(data)
			}
			fmt.Fprintln(os.Stdout)
			if err != nil {
				log.Fatal(err)
			}

		} else {
			outf, err := os.OpenFile(outfp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
			if doTpl {
				err = outTpl.Execute(outf, dataTpl)
			} else {
				_, err = outf.Write(data)
			}
			outf.Close()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

type globs []glob.Glob

func (l globs) MatchAny(s string) bool {
	for _, g := range l {
		if g.Match(s) {
			return true
		}
	}
	return false
}

func getPkgName(path string) (pkgName string) {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if pkgName != "" {
			return nil
		}
		if info.IsDir() {
			return filepath.SkipDir
		}
		d, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(bytes.NewReader(d))
		for sc.Scan() {
			l := sc.Text()
			if strings.HasPrefix(l, "package ") {
				pkgName = l[len("package "):]
				pkgName = strings.TrimSpace(pkgName)
				return nil
			}
		}
		return nil
	})
	if pkgName == "" {
		pkgName = filepath.Base(path)
	}
	return
}

func selectFiles(prog *loader.Program, pkg *loader.PackageInfo, skips []string) []*ast.File {
	var globs globs
	for _, s := range skips {
		globs = append(globs, glob.MustCompile(s))
	}
	globs = append(globs, glob.MustCompile("jenjen_*"))
	files := []*ast.File{}
	for _, f := range pkg.Files {
		fname := prog.Fset.File(f.Pos()).Name()
		fname = filepath.Base(fname)
		if globs.MatchAny(fname) {
			continue
		}
		files = append(files, f)
	}
	return files
}

type sr struct {
	search         string
	replace        string
	replacePkgName string
	replacePkgPath string
	searchCtx      string
}

func parseDirective(search, replace string) (ret sr) {
	ret.search = search
	if strings.Contains(search, ":") {
		k := strings.Split(search, ":")
		ret.searchCtx = strings.TrimSpace(k[0])
		ret.search = strings.TrimSpace(k[1])
	}
	ret.replace = replace
	indx := strings.LastIndex(replace, ".")
	if indx < 0 {
		ret.replace = replace
	} else {
		rs := strings.SplitN(replace, ".", indx-1)
		ret.replacePkgPath = rs[0]
		ret.replacePkgName = filepath.Base(ret.replacePkgPath)
		ret.replacePkgName = strings.Replace(ret.replacePkgName, "-", "", -1)
		ret.replace = ret.replacePkgName + "." + rs[1]
	}

	return ret
}
func parseDirectives(args []string) []sr {
	ret := []sr{}
	directives := []string{}
	for _, arg := range args {
		directives = append(directives, strings.Split(arg, ",")...)
	}
	for _, directive := range directives {
		directive = strings.TrimSpace(directive)
		k := strings.Split(directive, "=>")
		if len(k) < 2 {
			log.Fatalf("invalid directive %q in %v", directive, strings.Join(directives, ","))
		}
		k[0] = strings.TrimSpace(k[0])
		k[1] = strings.TrimSpace(k[1])
		ret = append(ret, parseDirective(k[0], k[1]))
	}
	return ret
}

func selectNodes(search string, f *ast.File) []ast.Node {
	if search == "" {
		return []ast.Node{f}
	}
	searchLevels := strings.Split(search, ".")
	search = searchLevels[0]
	nodes := []ast.Node{}
	for _, n := range f.Decls {
		switch x := n.(type) {
		case *ast.GenDecl:
			if len(searchLevels) == 1 {
				if len(x.Specs) > 0 {
					if y, ok := x.Specs[0].(*ast.TypeSpec); ok {
						if y.Name.Name == search {
							nodes = append(nodes, n)
						}
					}
				}
			}
		case *ast.FuncDecl:
			if x.Recv == nil && x.Name.Name == search {
				nodes = append(nodes, n)
			} else if x.Recv != nil && len(x.Recv.List) > 0 {
				if k, ok := x.Recv.List[0].Type.(*ast.StarExpr); ok {
					if j, ok := k.X.(*ast.Ident); ok {
						if j.Name == search {
							if len(searchLevels) == 1 {
								nodes = append(nodes, n)
							} else if x.Name.Name == searchLevels[1] {
								nodes = append(nodes, n)
							}
						}
					}
				} else if k, ok := x.Recv.List[0].Type.(*ast.Ident); ok {
					if k.Name == search {
						if len(searchLevels) == 1 {
							nodes = append(nodes, n)
						} else if x.Name.Name == searchLevels[1] {
							nodes = append(nodes, n)
						}
					}
				}
			}
		}
	}

	return nodes
}

func rewriteIdent(search, replace string) func(*astutil.Cursor) bool {
	return func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		if x, ok := n.(*ast.Ident); ok {
			if x.Name == search {
				x.Name = replace
			}
		}
		return true
	}
}

func rmNode(search string) func(*astutil.Cursor) bool {
	return func(cursor *astutil.Cursor) bool {
		n := cursor.Node()
		if x, ok := n.(*ast.GenDecl); ok {
			if len(x.Specs) > 0 {
				if y, ok := x.Specs[0].(*ast.TypeSpec); ok {
					if y.Name.Name == search {
						cursor.Delete()
						return false
					}
				}
			}
		}
		if x, ok := n.(*ast.FuncDecl); ok {
			if x.Recv == nil && x.Name.Name == search {
				cursor.Delete()
				return false

			} else if x.Recv != nil && len(x.Recv.List) > 0 {
				if k, ok := x.Recv.List[0].Type.(*ast.StarExpr); ok {
					if j, ok := k.X.(*ast.Ident); ok {
						if j.Name == search {
							cursor.Delete()
							return false
						}
					}
				} else if k, ok := x.Recv.List[0].Type.(*ast.Ident); ok {
					if k.Name == search {
						cursor.Delete()
						return false
					}
				}
			}
		}
		return true
	}
}
