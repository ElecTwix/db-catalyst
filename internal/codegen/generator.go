package codegen

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

type Options struct {
	Package      string
	EmitJSONTags bool
}

type Generator struct {
	opts Options
}

type File struct {
	Path    string
	Content []byte
}

func New(opts Options) *Generator {
	return &Generator{opts: opts}
}

type tableModel struct {
	tableName string
	typeName  string
	fields    []modelField
	needsSQL  bool
}

type modelField struct {
	columnName string
	fieldName  string
	goType     string
	jsonTag    string
}

type queryInfo struct {
	methodName string
	constName  string
	fileName   string
	sqlLiteral string
	command    block.Command
	docComment string
	params     []paramSpec
	returnType string
	returnZero string
	rowType    string
	helper     *helperSpec
	args       []string
}

type paramSpec struct {
	name   string
	goType string
}

type helperSpec struct {
	rowTypeName string
	funcName    string
	fields      []helperField
}

type helperField struct {
	name   string
	goType string
}

func (g *Generator) Generate(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	packageName := g.opts.Package
	if packageName == "" {
		packageName = "db"
	}

	tableModels := g.collectTableModels(catalog, analyses)
	queries := g.buildQueries(analyses)

	helperPtrs := make([]*helperSpec, 0)
	for i := range queries {
		if queries[i].helper != nil {
			helperPtrs = append(helperPtrs, queries[i].helper)
		}
	}
	slices.SortFunc(helperPtrs, func(a, b *helperSpec) int {
		return strings.Compare(a.rowTypeName, b.rowTypeName)
	})

	files := make([]File, 0, 1+len(queries))
	if len(tableModels) > 0 {
		modelsFile, err := g.buildModelsFile(packageName, tableModels)
		if err != nil {
			return nil, err
		}
		files = append(files, modelsFile)
	}

	querierFile, err := g.buildQuerierFile(packageName, queries)
	if err != nil {
		return nil, err
	}
	files = append(files, querierFile)

	if len(helperPtrs) > 0 {
		helperFile, err := g.buildHelpersFile(packageName, helperPtrs)
		if err != nil {
			return nil, err
		}
		files = append(files, helperFile)
	}

	queryFiles, err := g.buildQueryFiles(packageName, queries)
	if err != nil {
		return nil, err
	}
	files = append(files, queryFiles...)

	slices.SortFunc(files, func(a, b File) int {
		if a.Path == b.Path {
			return 0
		}
		if a.Path < b.Path {
			return -1
		}
		return 1
	})

	return files, nil
}

func (g *Generator) collectTableModels(catalog *model.Catalog, analyses []analyzer.Result) []*tableModel {
	if catalog == nil {
		return nil
	}
	referenced := make(map[string]*model.Table)
	for _, res := range analyses {
		for _, col := range res.Columns {
			if col.Table == "" {
				continue
			}
			if tbl, ok := catalog.Tables[col.Table]; ok {
				referenced[tbl.Name] = tbl
				continue
			}
			for name, tbl := range catalog.Tables {
				if strings.EqualFold(name, col.Table) {
					referenced[name] = tbl
					break
				}
			}
		}
	}
	if len(referenced) == 0 {
		return nil
	}
	names := make([]string, 0, len(referenced))
	for name := range referenced {
		names = append(names, name)
	}
	slices.Sort(names)
	models := make([]*tableModel, 0, len(names))
	for _, name := range names {
		tbl := referenced[name]
		model := g.buildTableModel(tbl)
		models = append(models, model)
	}
	return models
}

func (g *Generator) buildTableModel(tbl *model.Table) *tableModel {
	used := make(map[string]int)
	fields := make([]modelField, 0, len(tbl.Columns))
	needsSQL := false
	for _, col := range tbl.Columns {
		goName := ExportedIdentifier(col.Name)
		if goName == "" {
			goName = ExportedIdentifier("column")
		}
		if _, exists := used[goName]; exists {
			goName = UniqueName(goName, used)
		} else {
			used[goName] = 1
		}
		typeInfo := resolveType(analyzer.SQLiteTypeToGo(col.Type), !col.NotNull)
		if typeInfo.UsesSQLNull {
			needsSQL = true
		}
		field := modelField{
			columnName: col.Name,
			fieldName:  goName,
			goType:     typeInfo.GoType,
		}
		if g.opts.EmitJSONTags {
			field.jsonTag = fmt.Sprintf("`json:\"%s\"`", col.Name)
		}
		fields = append(fields, field)
	}
	typeName := ExportedIdentifier(tbl.Name)
	if typeName == "" {
		typeName = "Model"
	}
	return &tableModel{
		tableName: tbl.Name,
		typeName:  typeName,
		fields:    fields,
		needsSQL:  needsSQL,
	}
}

func (g *Generator) buildQueries(analyses []analyzer.Result) []queryInfo {
	queries := make([]queryInfo, 0, len(analyses))
	for _, res := range analyses {
		methodName := ExportedIdentifier(res.Query.Block.Name)
		if methodName == "" {
			methodName = "Query"
		}
		constName := "query" + methodName
		fileName := fmt.Sprintf("query_%s.go", FileName(res.Query.Block.Name))
		params := g.buildParams(res.Params)
		args := make([]string, 0, len(params))
		for _, p := range params {
			args = append(args, p.name)
		}
		info := queryInfo{
			methodName: methodName,
			constName:  constName,
			fileName:   fileName,
			sqlLiteral: res.Query.Block.SQL,
			command:    res.Query.Block.Command,
			docComment: res.Query.Block.Doc,
			params:     params,
			args:       args,
		}

		switch res.Query.Block.Command {
		case block.CommandOne:
			helper := g.buildHelper(methodName, res.Columns)
			info.helper = helper
			info.rowType = helper.rowTypeName
			info.returnType = helper.rowTypeName
			info.returnZero = helper.rowTypeName + "{}"
		case block.CommandMany:
			helper := g.buildHelper(methodName, res.Columns)
			info.helper = helper
			info.rowType = helper.rowTypeName
			info.returnType = "[]" + helper.rowTypeName
			info.returnZero = "nil"
		case block.CommandExec:
			info.returnType = "sql.Result"
			info.returnZero = "nil"
		case block.CommandExecResult:
			info.returnType = "QueryResult"
			info.returnZero = "QueryResult{}"
		}

		queries = append(queries, info)
	}

	slices.SortFunc(queries, func(a, b queryInfo) int {
		return strings.Compare(a.methodName, b.methodName)
	})

	return queries
}

func (g *Generator) buildParams(params []analyzer.ResultParam) []paramSpec {
	result := make([]paramSpec, 0, len(params))
	used := map[string]int{"ctx": 1}
	for idx, p := range params {
		name := UnexportedIdentifier(p.Name)
		if name == "" {
			name = fmt.Sprintf("arg%d", idx+1)
		}
		if _, exists := used[name]; exists {
			name = UniqueName(name, used)
		} else {
			used[name] = 1
		}
		typeInfo := resolveType(p.GoType, p.Nullable)
		result = append(result, paramSpec{name: name, goType: typeInfo.GoType})
	}
	return result
}

func (g *Generator) buildHelper(methodName string, columns []analyzer.ResultColumn) *helperSpec {
	rowTypeName := methodName + "Row"
	funcName := "scan" + methodName + "Row"
	fields := make([]helperField, 0, len(columns))
	used := make(map[string]int)
	for idx, col := range columns {
		baseName := col.Name
		if baseName == "" {
			baseName = fmt.Sprintf("column_%d", idx+1)
		}
		fieldName := ExportedIdentifier(baseName)
		if fieldName == "" {
			fieldName = fmt.Sprintf("Column%d", idx+1)
		}
		if _, exists := used[fieldName]; exists {
			fieldName = UniqueName(fieldName, used)
		} else {
			used[fieldName] = 1
		}
		typeInfo := resolveType(col.GoType, col.Nullable)
		fields = append(fields, helperField{name: fieldName, goType: typeInfo.GoType})
	}
	return &helperSpec{rowTypeName: rowTypeName, funcName: funcName, fields: fields}
}

func (g *Generator) buildModelsFile(pkg string, models []*tableModel) (File, error) {
	fset := token.NewFileSet()
	file := &ast.File{Name: ast.NewIdent(pkg)}
	decls := make([]ast.Decl, 0, len(models))
	for _, mdl := range models {
		fields := make([]*ast.Field, 0, len(mdl.fields))
		for _, fld := range mdl.fields {
			expr, err := parser.ParseExpr(fld.goType)
			if err != nil {
				return File{}, err
			}
			field := &ast.Field{
				Names: []*ast.Ident{ast.NewIdent(fld.fieldName)},
				Type:  expr,
			}
			if fld.jsonTag != "" {
				field.Tag = &ast.BasicLit{Kind: token.STRING, Value: fld.jsonTag}
			}
			fields = append(fields, field)
		}
		structType := &ast.StructType{Fields: &ast.FieldList{List: fields}}
		typeSpec := &ast.TypeSpec{Name: ast.NewIdent(mdl.typeName), Type: structType}
		decl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{typeSpec}}
		decls = append(decls, decl)
	}
	file.Decls = decls

	content, err := formatFile(fset, file)
	if err != nil {
		return File{}, err
	}
	return File{Path: "models.go", Content: content}, nil
}

func (g *Generator) buildQuerierFile(pkg string, queries []queryInfo) (File, error) {
	fset := token.NewFileSet()
	file := &ast.File{Name: ast.NewIdent(pkg)}

	interfaceFields := make([]*ast.Field, 0, len(queries))
	for _, q := range queries {
		params := []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: selector("context", "Context")}}
		for _, p := range q.params {
			expr, err := parser.ParseExpr(p.goType)
			if err != nil {
				return File{}, err
			}
			params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent(p.name)}, Type: expr})
		}
		results := []*ast.Field{}
		if q.returnType != "" {
			expr, err := parser.ParseExpr(q.returnType)
			if err != nil {
				return File{}, err
			}
			results = append(results, &ast.Field{Type: expr})
		}
		errorType, _ := parser.ParseExpr("error")
		results = append(results, &ast.Field{Type: errorType})

		interfaceFields = append(interfaceFields, &ast.Field{
			Names: []*ast.Ident{ast.NewIdent(q.methodName)},
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: params},
				Results: &ast.FieldList{List: results},
			},
		})
	}

	querierType := &ast.TypeSpec{Name: ast.NewIdent("Querier"), Type: &ast.InterfaceType{Methods: &ast.FieldList{List: interfaceFields}}}
	querierDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{querierType}}

	dbtxMethods := []*ast.Field{
		{Names: []*ast.Ident{ast.NewIdent("ExecContext")}, Type: funcType([]*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("any")}},
		}, []*ast.Field{{Type: selector("sql", "Result")}, {Type: ast.NewIdent("error")}})},
		{Names: []*ast.Ident{ast.NewIdent("QueryContext")}, Type: funcType([]*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("any")}},
		}, []*ast.Field{{Type: selector("sql", "Rows")}, {Type: ast.NewIdent("error")}})},
		{Names: []*ast.Ident{ast.NewIdent("QueryRowContext")}, Type: funcType([]*ast.Field{
			{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*ast.Ident{ast.NewIdent("query")}, Type: ast.NewIdent("string")},
			{Names: []*ast.Ident{ast.NewIdent("args")}, Type: &ast.Ellipsis{Elt: ast.NewIdent("any")}},
		}, []*ast.Field{{Type: selector("sql", "Row")}})},
	}
	dbtxType := &ast.TypeSpec{Name: ast.NewIdent("DBTX"), Type: &ast.InterfaceType{Methods: &ast.FieldList{List: dbtxMethods}}}
	dbtxDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{dbtxType}}

	queriesStruct := &ast.TypeSpec{Name: ast.NewIdent("Queries"), Type: &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("db")}, Type: ast.NewIdent("DBTX")}}}}}
	queriesDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{queriesStruct}}

	newFunc := &ast.FuncDecl{
		Name: ast.NewIdent("New"),
		Type: &ast.FuncType{
			Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("db")}, Type: ast.NewIdent("DBTX")}}},
			Results: &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: ast.NewIdent("Queries")}}}},
		},
		Body: &ast.BlockStmt{List: []ast.Stmt{
			&ast.ReturnStmt{Results: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{Type: ast.NewIdent("Queries"), Elts: []ast.Expr{
				&ast.KeyValueExpr{Key: ast.NewIdent("db"), Value: ast.NewIdent("db")},
			}}}}},
		}},
	}

	resultStruct := &ast.TypeSpec{Name: ast.NewIdent("QueryResult"), Type: &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{
		{Names: []*ast.Ident{ast.NewIdent("LastInsertID")}, Type: ast.NewIdent("int64")},
		{Names: []*ast.Ident{ast.NewIdent("RowsAffected")}, Type: ast.NewIdent("int64")},
	}}}}
	resultDecl := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{resultStruct}}

	file.Decls = []ast.Decl{querierDecl, dbtxDecl, queriesDecl, newFunc, resultDecl}

	content, err := formatFile(fset, file)
	if err != nil {
		return File{}, err
	}
	return File{Path: "querier.go", Content: content}, nil
}

func (g *Generator) buildHelpersFile(pkg string, helpers []*helperSpec) (File, error) {
	fset := token.NewFileSet()
	file := &ast.File{Name: ast.NewIdent(pkg)}
	decls := make([]ast.Decl, 0, len(helpers)*2)
	for _, helper := range helpers {
		fields := make([]*ast.Field, 0, len(helper.fields))
		for _, fld := range helper.fields {
			expr, err := parser.ParseExpr(fld.goType)
			if err != nil {
				return File{}, err
			}
			fields = append(fields, &ast.Field{Names: []*ast.Ident{ast.NewIdent(fld.name)}, Type: expr})
		}
		rowType := &ast.StructType{Fields: &ast.FieldList{List: fields}}
		rowSpec := &ast.TypeSpec{Name: ast.NewIdent(helper.rowTypeName), Type: rowType}
		decls = append(decls, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{rowSpec}})

		stmts := []ast.Stmt{
			mustParseStmt("var item " + helper.rowTypeName),
		}
		scanArgs := make([]string, 0, len(helper.fields))
		for _, fld := range helper.fields {
			scanArgs = append(scanArgs, "&item."+fld.name)
		}
		scanStmt := fmt.Sprintf("if err := rows.Scan(%s); err != nil {\nreturn item, err\n}", strings.Join(scanArgs, ", "))
		stmts = append(stmts, mustParseStmt(scanStmt))
		stmts = append(stmts, mustParseStmt("return item, nil"))

		funcDecl := &ast.FuncDecl{
			Name: ast.NewIdent(helper.funcName),
			Type: &ast.FuncType{
				Params:  &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("rows")}, Type: selector("sql", "Rows")}}},
				Results: &ast.FieldList{List: []*ast.Field{{Type: ast.NewIdent(helper.rowTypeName)}, {Type: ast.NewIdent("error")}}},
			},
			Body: &ast.BlockStmt{List: stmts},
		}
		decls = append(decls, funcDecl)
	}
	file.Decls = decls

	content, err := formatFile(fset, file)
	if err != nil {
		return File{}, err
	}
	return File{Path: "_helpers.go", Content: content}, nil
}

func (g *Generator) buildQueryFiles(pkg string, queries []queryInfo) ([]File, error) {
	files := make([]File, 0, len(queries))
	for _, q := range queries {
		fset := token.NewFileSet()
		file := &ast.File{Name: ast.NewIdent(pkg)}

		constSpec := &ast.ValueSpec{
			Names:  []*ast.Ident{ast.NewIdent(q.constName)},
			Type:   ast.NewIdent("string"),
			Values: []ast.Expr{stringLiteral(q.sqlLiteral)},
		}
		constDecl := &ast.GenDecl{Tok: token.CONST, Specs: []ast.Spec{constSpec}}

		funcDecl, err := g.buildQueryFunc(q)
		if err != nil {
			return nil, err
		}

		file.Decls = []ast.Decl{constDecl, funcDecl}

		content, err := formatFile(fset, file)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: q.fileName, Content: content})
	}
	return files, nil
}

func (g *Generator) buildQueryFunc(q queryInfo) (*ast.FuncDecl, error) {
	params := []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("ctx")}, Type: selector("context", "Context")}}
	for _, p := range q.params {
		expr, err := parser.ParseExpr(p.goType)
		if err != nil {
			return nil, err
		}
		params = append(params, &ast.Field{Names: []*ast.Ident{ast.NewIdent(p.name)}, Type: expr})
	}

	results := []*ast.Field{}
	if q.returnType != "" {
		expr, err := parser.ParseExpr(q.returnType)
		if err != nil {
			return nil, err
		}
		results = append(results, &ast.Field{Type: expr})
	}
	errorType, _ := parser.ParseExpr("error")
	results = append(results, &ast.Field{Type: errorType})

	body := make([]ast.Stmt, 0)

	switch q.command {
	case block.CommandExec:
		args := append([]string{"ctx", q.constName}, q.args...)
		body = append(body, mustParseStmt(fmt.Sprintf("return q.db.ExecContext(%s)", strings.Join(args, ", "))))
	case block.CommandExecResult:
		args := append([]string{"ctx", q.constName}, q.args...)
		body = append(body, mustParseStmt(fmt.Sprintf("res, err := q.db.ExecContext(%s)", strings.Join(args, ", "))))
		body = append(body, mustParseStmt("if err != nil {\nreturn QueryResult{}, err\n}"))
		body = append(body, mustParseStmt("result := QueryResult{}"))
		body = append(body, mustParseStmt("if v, err := res.LastInsertId(); err == nil {\nresult.LastInsertID = v\n}"))
		body = append(body, mustParseStmt("if v, err := res.RowsAffected(); err == nil {\nresult.RowsAffected = v\n}"))
		body = append(body, mustParseStmt("return result, nil"))
	default:
		args := append([]string{"ctx", q.constName}, q.args...)
		body = append(body, mustParseStmt(fmt.Sprintf("rows, err := q.db.QueryContext(%s)", strings.Join(args, ", "))))
		zero := q.returnZero
		if zero == "" {
			zero = "nil"
		}
		body = append(body, mustParseStmt("if err != nil {\nreturn "+zero+", err\n}"))
		body = append(body, mustParseStmt("defer rows.Close()"))

		if q.command == block.CommandOne {
			body = append(body, mustParseStmt("if !rows.Next() {\nif err := rows.Err(); err != nil {\nreturn "+zero+", err\n}\nreturn "+zero+", sql.ErrNoRows\n}"))
			body = append(body, mustParseStmt("item, err := "+q.helper.funcName+"(rows)"))
			body = append(body, mustParseStmt("if err != nil {\nreturn item, err\n}"))
			body = append(body, mustParseStmt("if err := rows.Err(); err != nil {\nreturn item, err\n}"))
			body = append(body, mustParseStmt("return item, nil"))
		} else {
			body = append(body, mustParseStmt("items := make([]"+q.rowType+", 0)"))
			loop := &ast.ForStmt{
				Cond: &ast.CallExpr{Fun: &ast.SelectorExpr{X: ast.NewIdent("rows"), Sel: ast.NewIdent("Next")}},
				Body: &ast.BlockStmt{List: []ast.Stmt{
					mustParseStmt("item, err := " + q.helper.funcName + "(rows)"),
					mustParseStmt("if err != nil {\nreturn nil, err\n}"),
					mustParseStmt("items = append(items, item)"),
				}},
			}
			body = append(body, loop)
			body = append(body, mustParseStmt("if err := rows.Err(); err != nil {\nreturn nil, err\n}"))
			body = append(body, mustParseStmt("return items, nil"))
		}
	}

	funcDecl := &ast.FuncDecl{
		Recv: &ast.FieldList{List: []*ast.Field{{Names: []*ast.Ident{ast.NewIdent("q")}, Type: &ast.StarExpr{X: ast.NewIdent("Queries")}}}},
		Name: ast.NewIdent(q.methodName),
		Type: &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: results}},
		Body: &ast.BlockStmt{List: body},
	}

	if q.docComment != "" {
		funcDecl.Doc = &ast.CommentGroup{List: []*ast.Comment{{Text: "// " + q.docComment}}}
	}

	return funcDecl, nil
}

func selector(pkg, name string) *ast.SelectorExpr {
	return &ast.SelectorExpr{X: ast.NewIdent(pkg), Sel: ast.NewIdent(name)}
}

func funcType(params []*ast.Field, results []*ast.Field) ast.Expr {
	return &ast.FuncType{Params: &ast.FieldList{List: params}, Results: &ast.FieldList{List: results}}
}

func stringLiteral(value string) ast.Expr {
	if !strings.Contains(value, "`") {
		return &ast.BasicLit{Kind: token.STRING, Value: "`" + value + "`"}
	}
	return &ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(value)}
}

func mustParseStmt(code string) ast.Stmt {
	src := "package p\nfunc _() {\n" + code + "\n}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		panic(err)
	}
	if len(file.Decls) == 0 {
		panic("no declarations parsed")
	}
	fn, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || fn.Body == nil || len(fn.Body.List) == 0 {
		panic("parsed function missing body")
	}
	return fn.Body.List[0]
}

func formatFile(fset *token.FileSet, file *ast.File) ([]byte, error) {
	var buf bytes.Buffer
	cfg := &printer.Config{Mode: printer.TabIndent | printer.UseSpaces, Tabwidth: 8}
	if err := cfg.Fprint(&buf, fset, file); err != nil {
		return nil, err
	}
	formatted, err := imports.Process("", buf.Bytes(), nil)
	if err != nil {
		return nil, err
	}
	return formatted, nil
}
