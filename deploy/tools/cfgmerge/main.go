// Command cfgmerge 按表合并 Luban 生成的配置表 Go 文件：
//
//	item.go      Item（记录结构）+ TbItem（表类）
//	weapon.go    Weapon + TbWeapon
//	tables.go    Tables / NewTables / JsonLoader（汇总）
//
// 用法：go run ./deploy/tools/cfgmerge share\cfg
//
// 生成文件每次执行 gen-config.bat 都会重建，因此合并必须由脚本完成，勿手工改名。
package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 手写文件与合并产物本身不参与合并。
var skip = map[string]bool{
	"cfg.go":            true,
	"doc.go":            true,
	"cfg_smoke_test.go": true,
	"records.go":        true,
	"tables.go":         true,
}

// tablesGroup 汇总表（Tables.go）的分组键。
const tablesGroup = "@tables"

func main() {
	dir := "share/cfg"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fatal(err)
	}

	// 按表分组：Weapon.go 与 TbWeapon.go 同属 weapon；Tables.go 单独成组。
	groups := make(map[string][]string)
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || skip[name] {
			continue
		}
		base := strings.TrimSuffix(name, ".go")
		switch {
		case base == "Tables":
			groups[tablesGroup] = append(groups[tablesGroup], name)
		case strings.HasPrefix(base, "Tb"):
			key := strings.ToLower(strings.TrimPrefix(base, "Tb"))
			groups[key] = append(groups[key], name)
		default:
			groups[strings.ToLower(base)] = append(groups[strings.ToLower(base)], name)
		}
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	outputs := make(map[string][]byte, len(groups))
	var inputs []string
	for _, key := range keys {
		names := groups[key]
		// 记录结构作为合并基底（含 import），表类文件排在后面。
		sort.Slice(names, func(i, j int) bool {
			ti, tj := strings.HasPrefix(names[i], "Tb"), strings.HasPrefix(names[j], "Tb")
			if ti != tj {
				return !ti
			}
			return names[i] < names[j]
		})
		data, err := merge(dir, names)
		if err != nil {
			fatal(err)
		}
		out := key + ".go"
		if key == tablesGroup {
			out = "tables.go"
		}
		outputs[out] = data
		inputs = append(inputs, names...)
	}

	// 先删原始文件再写产物：Windows 文件名不区分大小写，避免 tables.go 与 Tables.go 互相覆盖。
	for _, name := range inputs {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			fatal(err)
		}
	}
	for out, data := range outputs {
		if err := os.WriteFile(filepath.Join(dir, out), data, 0o644); err != nil {
			fatal(err)
		}
	}
	fmt.Printf("cfgmerge: tables=%d files=%d\n", len(groups), len(outputs))
}

// merge 按给定顺序把同表文件合并成一个 AST 文件并返回格式化后的源码。
// 后续文件的 import 必须被基底文件覆盖（Luban 模板中表类文件通常无 import）。
func merge(dir string, names []string) ([]byte, error) {
	fset := token.NewFileSet()
	var merged *ast.File
	var baseImports map[string]bool
	for i, name := range names {
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		paths := importPaths(f)
		if i == 0 {
			merged = f
			baseImports = paths
			continue
		}
		if !subset(paths, baseImports) {
			return nil, fmt.Errorf("%s: imports not covered by %s", name, names[0])
		}
		// 只合并声明，跳过重复的 import 块（import 必须位于其他声明之前）。
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.IMPORT {
				continue
			}
			merged.Decls = append(merged.Decls, d)
		}
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, merged); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func importPaths(f *ast.File) map[string]bool {
	paths := make(map[string]bool, len(f.Imports))
	for _, imp := range f.Imports {
		paths[imp.Path.Value] = true
	}
	return paths
}

func subset(a, b map[string]bool) bool {
	for p := range a {
		if !b[p] {
			return false
		}
	}
	return true
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "cfgmerge:", err)
	os.Exit(1)
}
