// Package main содержит проверку документации экспортированных Go-идентификаторов.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	roots := os.Args[1:]
	if len(roots) == 0 {
		roots = []string{"."}
	}

	if err := run(roots); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(roots []string) error {
	checker := docChecker{
		fileSet:     token.NewFileSet(),
		packages:    make(map[string]bool),
		packageDocs: make(map[string]bool),
	}

	for _, root := range roots {
		if err := filepath.WalkDir(root, checker.checkPath); err != nil {
			return err
		}
	}

	for packagePath := range checker.packages {
		if !checker.packageDocs[packagePath] {
			checker.report(packagePath, 1, "missing package doc")
		}
	}

	if checker.missing > 0 {
		return fmt.Errorf("doccheck: found %d undocumented exported declarations", checker.missing)
	}
	return nil
}

type docChecker struct {
	fileSet     *token.FileSet
	packages    map[string]bool
	packageDocs map[string]bool
	missing     int
}

func (c *docChecker) checkPath(path string, entry os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if entry.IsDir() {
		if shouldSkipDir(entry.Name()) {
			return filepath.SkipDir
		}
		return nil
	}
	if shouldSkipFile(path) {
		return nil
	}

	file, err := parser.ParseFile(c.fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return err
	}
	if isGenerated(path, file) {
		return nil
	}

	packagePath := filepath.Dir(path)
	c.packages[packagePath] = true
	if file.Doc != nil && strings.TrimSpace(file.Doc.Text()) != "" {
		c.packageDocs[packagePath] = true
	}

	for _, declaration := range file.Decls {
		c.checkDeclaration(path, declaration)
	}
	return nil
}

func (c *docChecker) checkDeclaration(path string, declaration ast.Decl) {
	switch declaration := declaration.(type) {
	case *ast.FuncDecl:
		if declaration.Name.IsExported() && !hasNamedDoc(declaration.Doc, declaration.Name.Name) {
			c.report(path, c.fileSet.Position(declaration.Pos()).Line, "missing doc for func/method "+declaration.Name.Name)
		}
	case *ast.GenDecl:
		c.checkGenDeclaration(path, declaration)
	}
}

func (c *docChecker) checkGenDeclaration(path string, declaration *ast.GenDecl) {
	for _, spec := range declaration.Specs {
		switch spec := spec.(type) {
		case *ast.TypeSpec:
			if spec.Name.IsExported() && !hasNamedDoc(declaration.Doc, spec.Name.Name) && !hasNamedDoc(spec.Doc, spec.Name.Name) {
				c.report(path, c.fileSet.Position(spec.Pos()).Line, "missing doc for type "+spec.Name.Name)
			}
		case *ast.ValueSpec:
			for _, name := range spec.Names {
				if name.IsExported() && !hasNamedDoc(declaration.Doc, name.Name) && !hasNamedDoc(spec.Doc, name.Name) {
					c.report(path, c.fileSet.Position(name.Pos()).Line, "missing doc for value "+name.Name)
				}
			}
		}
	}
}

func (c *docChecker) report(path string, line int, message string) {
	fmt.Printf("%s:%d: %s\n", path, line, message)
	c.missing++
}

func shouldSkipDir(name string) bool {
	switch name {
	case ".git", ".idea", ".vscode", "bin", "storage", "tmp", "vendor":
		return true
	default:
		return false
	}
}

func shouldSkipFile(path string) bool {
	return !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go")
}

func isGenerated(path string, file *ast.File) bool {
	if strings.Contains(filepath.ToSlash(path), "/pkg/api/generated/") {
		return true
	}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.Contains(comment.Text, "Code generated") || strings.Contains(comment.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

func hasNamedDoc(comments *ast.CommentGroup, name string) bool {
	if comments == nil {
		return false
	}

	text := strings.TrimSpace(comments.Text())
	return strings.HasPrefix(text, name+" ") || strings.HasPrefix(text, name+".")
}
