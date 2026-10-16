package apiv3

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"
)

// The Code constants and the spec's error codes match both ways, so a new code
// can't slip in unnoticed and a dropped one doesn't linger. Code stays an open string; this only keeps the named set
// complete.
func TestCodesCoverTheSpecsErrorCodes(t *testing.T) {
	declared := stringConstants(t, "errors.go", "Code")
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join("..", "internal", "gen", "gen.go"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	inSpec := map[string]bool{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			if typ, ok := vs.Type.(*ast.Ident); !ok || typ.Name != "ErrorBodyCode" {
				continue
			}
			for _, v := range vs.Values {
				code := strings.Trim(v.(*ast.BasicLit).Value, `"`)
				found++
				inSpec[code] = true
				if !declared[code] {
					t.Errorf("the spec declares error code %q but apiv3 has no Code constant for it", code)
				}
			}
		}
	}
	if found == 0 {
		t.Fatal("found no ErrorBodyCode values in gen.go; has the generated name changed?")
	}
	// The other way too: a Code the spec no longer declares is a constant callers
	// would compare against in vain. fault_merged is the exception: the client
	// assigns it to a merged fault's 301, which has no body to carry a code.
	for code := range declared {
		if !inSpec[code] && code != "fault_merged" {
			t.Errorf("apiv3 has a Code constant for %q, which the spec no longer declares", code)
		}
	}
}

// stringConstants returns the values of the string constants of the given type
// declared in a file of this package.
func stringConstants(t *testing.T, filename, typeName string) map[string]bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]bool{}
	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			if typ, ok := vs.Type.(*ast.Ident); !ok || typ.Name != typeName {
				continue
			}
			for _, v := range vs.Values {
				if lit, ok := v.(*ast.BasicLit); ok {
					values[strings.Trim(lit.Value, `"`)] = true
				}
			}
		}
	}
	return values
}
