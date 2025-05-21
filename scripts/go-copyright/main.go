// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	copyright = `Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.`
)

var (
	excludes = []string{
		"test/conformance",
	}
)

func checkCopyright() ([]string, error) {
	var files []string
	err := filepath.Walk(".", func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip directories like ".git".
			if name := info.Name(); name != "." && strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only check Go files.
		if !strings.HasSuffix(filename, ".go") {
			return nil
		}
		for _, dir := range excludes {
			if isFileInDir(filename, dir) {
				return nil
			}
		}

		needsCopyright, err := checkFile(filename)
		if err != nil {
			return err
		}
		if needsCopyright {
			files = append(files, filename)
		}
		return nil
	})
	return files, err
}

func isFileInDir(filename, dir string) bool {
	for ; len(filename) >= len(dir); filename = path.Dir(filename) {
		if filename == dir {
			return true
		}
	}
	return false
}

func checkFile(filename string) (bool, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, filename, content, parser.ParseComments)
	if err != nil {
		return false, err
	}
	// Don't require headers on generated files.
	if isGenerated(fset, parsed) {
		return false, nil
	}
	shouldAddCopyright := true
	for _, c := range parsed.Comments {
		// The copyright should appear before the package declaration.
		if c.Pos() > parsed.Package {
			break
		}

		if strings.HasPrefix(c.Text(), copyright) {
			shouldAddCopyright = false
			break
		}
	}
	return shouldAddCopyright, nil
}

var generatedRx = regexp.MustCompile(`//.*DO NOT EDIT\.?`)

func isGenerated(fset *token.FileSet, file *ast.File) bool {
	for _, commentGroup := range file.Comments {
		for _, comment := range commentGroup.List {
			if matched := generatedRx.MatchString(comment.Text); !matched {
				continue
			}
			// Check if comment is at the beginning of the line in source.
			if pos := fset.Position(comment.Slash); pos.Column == 1 {
				return true
			}
		}
	}
	return false
}

func main() {
	files, err := checkCopyright()
	if err != nil {
		log.Fatal(err)
	}
	if len(files) > 0 {
		fmt.Printf("[ERROR] invalid copyright files (%d):\n", len(files))
		fmt.Println(strings.Join(files, "\n"))
		os.Exit(1)
		return
	}
	fmt.Println("Good files !")
}
