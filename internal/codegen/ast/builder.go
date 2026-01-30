// Package ast provides types and logic for building Go ASTs.
package ast

import (
	"context"
	"errors"
	"fmt"
	goast "go/ast"
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/imports"

	"github.com/electwix/db-catalyst/internal/query/analyzer"
	"github.com/electwix/db-catalyst/internal/query/block"
	"github.com/electwix/db-catalyst/internal/schema/model"
)

// PreparedOptions captures prepared-query generation toggles.
type PreparedOptions struct {
	Enabled     bool
	EmitMetrics bool
	ThreadSafe  bool
}

// Options configures the AST builder.
type Options struct {
	Package             string
	EmitJSONTags        bool
	EmitEmptySlices     bool
	EmitPointersForNull bool
	Prepared            PreparedOptions
	TypeResolver        *TypeResolver
}

// File represents an AST file ready for rendering.
type File struct {
	Path string
	Node *goast.File
	Raw  []byte
}

// Builder constructs Go AST files for code generation outputs.
type Builder struct {
	opts Options
}

// New returns a builder configured with the provided options.
func New(options Options) *Builder {
	return &Builder{opts: options}
}

// Build produces the Go AST files for the provided catalog and analyses.
func (b *Builder) Build(ctx context.Context, catalog *model.Catalog, analyses []analyzer.Result) ([]File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	packageName := b.opts.Package
	if packageName == "" {
		packageName = "db"
	}

	tableModels, err := b.collectTableModels(catalog, analyses)
	if err != nil {
		return nil, err
	}
	queries, err := b.buildQueries(analyses)
	if err != nil {
		return nil, err
	}

	helperPtrs := make([]*helperSpec, 0, len(queries))
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
		node, err := b.buildModelsFile(packageName, tableModels)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: "models.gen.go", Node: node})
	}

	querierNode, err := b.buildQuerierFile(packageName, queries)
	if err != nil {
		return nil, err
	}
	files = append(files, File{Path: "querier.gen.go", Node: querierNode})

	if len(helperPtrs) > 0 {
		helperNode, err := b.buildHelpersFile(packageName, helperPtrs)
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: "_helpers.gen.go", Node: helperNode})
	}

	queryFiles, err := b.buildQueryFiles(packageName, queries)
	if err != nil {
		return nil, err
	}
	files = append(files, queryFiles...)

	if b.opts.Prepared.Enabled {
		preparedFile, err := b.buildPreparedFile(packageName, queries)
		if err != nil {
			return nil, err
		}
		files = append(files, preparedFile)
	}

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
	stmtField  string
	prepareFn  string
	metricsKey string
}

type paramSpec struct {
	name           string
	goType         string
	variadic       bool
	variadicCount  int
	sliceName      string
	argExpr        string
	isDynamicSlice bool
	marker         string
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

func (b *Builder) collectTableModels(catalog *model.Catalog, analyses []analyzer.Result) ([]*tableModel, error) {
	if catalog == nil {
		return nil, nil
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
		return nil, nil
	}
	names := make([]string, 0, len(referenced))
	for name := range referenced {
		names = append(names, name)
	}
	slices.Sort(names)
	models := make([]*tableModel, 0, len(names))
	for _, name := range names {
		tbl := referenced[name]
		model, err := b.buildTableModel(tbl)
		if err != nil {
			return nil, err
		}
		models = append(models, model)
	}
	return models, nil
}

func (b *Builder) buildTableModel(tbl *model.Table) (*tableModel, error) {
	used := make(map[string]int)
	fields := make([]modelField, 0, len(tbl.Columns))
	needsSQL := false
	for _, col := range tbl.Columns {
		goName := ExportedIdentifier(col.Name)
		if goName == "" {
			goName = ExportedIdentifier("column")
		}
		if _, exists := used[goName]; exists {
			var err error
			goName, err = UniqueName(goName, used)
			if err != nil {
				return nil, err
			}
		} else {
			used[goName] = 1
		}
		var typeInfo TypeInfo
		if b.opts.TypeResolver != nil {
			typeInfo = b.opts.TypeResolver.ResolveType(col.Type, !col.NotNull)
		} else {
			typeInfo = resolveType(analyzer.SQLiteTypeToGo(col.Type), !col.NotNull)
		}
		if typeInfo.UsesSQLNull {
			needsSQL = true
		}
		field := modelField{
			columnName: col.Name,
			fieldName:  goName,
			goType:     typeInfo.GoType,
		}
		if b.opts.EmitJSONTags {
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
	}, nil
}

func (b *Builder) buildQueries(analyses []analyzer.Result) ([]queryInfo, error) {
	queries := make([]queryInfo, 0, len(analyses))
	for _, res := range analyses {
		methodName := ExportedIdentifier(res.Query.Block.Name)
		if methodName == "" {
			methodName = "Query"
		}
		constName := "query" + methodName
		fileName := fmt.Sprintf("query_%s.go", FileName(res.Query.Block.Name))

		// Pre-process params to identify dynamic slices and prepare SQL replacement
		params, err := b.buildParams(res.Params)
		if err != nil {
			return nil, err
		}

		sqlLiteral := res.Query.Block.SQL
		// Replace dynamic slice macros with markers in the SQL string
		// We do this by iterating params in reverse offset order to avoid offset shifting
		// 3. Apply replacements to SQL.
		// 4. Update paramSpec with the used marker.

		// We need unique markers.
		for i := len(params) - 1; i >= 0; i-- {
			if params[i].isDynamicSlice {
				pp := res.Query.Params[i]
				marker := fmt.Sprintf("/*SLICE:%s*/", params[i].name)
				params[i].marker = marker
				// Replace in sqlLiteral
				// Ensure offsets are valid
				if pp.StartOffset >= 0 && pp.EndOffset <= len(sqlLiteral) && pp.StartOffset <= pp.EndOffset {
					sqlLiteral = sqlLiteral[:pp.StartOffset] + marker + sqlLiteral[pp.EndOffset:]
				}
			}
		}

		args := make([]string, 0, len(params))
		for _, p := range params {
			args = append(args, p.argExpr)
		}
		stmtField := "stmt" + methodName
		prepareFn := "prepare" + methodName
		info := queryInfo{
			methodName: methodName,
			constName:  constName,
			fileName:   fileName,
			sqlLiteral: sqlLiteral,
			command:    res.Query.Block.Command,
			docComment: res.Query.Block.Doc,
			params:     params,
			args:       args,
			stmtField:  stmtField,
			prepareFn:  prepareFn,
			metricsKey: methodName,
		}

		switch res.Query.Block.Command {
		case block.CommandOne:
			helper, err := b.buildHelper(methodName, res.Columns)
			if err != nil {
				return nil, err
			}
			info.helper = helper
			info.rowType = helper.rowTypeName
			info.returnType = helper.rowTypeName
			info.returnZero = helper.rowTypeName + "{}"
		case block.CommandMany:
			helper, err := b.buildHelper(methodName, res.Columns)
			if err != nil {
				return nil, err
			}
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
		default:
			// CommandUnknown or other unhandled commands - skip
		}

		queries = append(queries, info)
	}

	slices.SortFunc(queries, func(a, b queryInfo) int {
		return strings.Compare(a.methodName, b.methodName)
	})

	return queries, nil
}

func (b *Builder) buildParams(params []analyzer.ResultParam) ([]paramSpec, error) {
	result := make([]paramSpec, 0, len(params))
	used := map[string]int{"ctx": 1}
	for idx, p := range params {
		name := UnexportedIdentifier(p.Name)
		if name == "" {
			name = fmt.Sprintf("arg%d", idx+1)
		}
		if _, exists := used[name]; exists {
			var err error
			name, err = UniqueName(name, used)
			if err != nil {
				return nil, err
			}
		} else {
			used[name] = 1
		}

		var typeInfo TypeInfo
		if b.opts.TypeResolver != nil {
			typeInfo = b.opts.TypeResolver.ResolveType(p.GoType, p.Nullable)
		} else {
			typeInfo = resolveType(p.GoType, p.Nullable)
		}

		isDynamicSlice := p.IsVariadic && p.VariadicCount == 0

		spec := paramSpec{
			name:           name,
			goType:         typeInfo.GoType,
			variadic:       p.IsVariadic && !isDynamicSlice,
			variadicCount:  p.VariadicCount,
			argExpr:        name,
			isDynamicSlice: isDynamicSlice,
		}

		if isDynamicSlice {
			// For dynamic slices, the argument is just the slice name
			spec.argExpr = name
		} else if p.IsVariadic {
			sliceName := name + "Args"
			if _, exists := used[sliceName]; exists {
				var err error
				sliceName, err = UniqueName(sliceName, used)
				if err != nil {
					return nil, err
				}
			} else {
				used[sliceName] = 1
			}
			spec.sliceName = sliceName
			spec.argExpr = sliceName + "..."
		}

		result = append(result, spec)
	}
	return result, nil
}

func (b *Builder) buildHelper(methodName string, columns []analyzer.ResultColumn) (*helperSpec, error) {
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
			var err error
			fieldName, err = UniqueName(fieldName, used)
			if err != nil {
				return nil, err
			}
		} else {
			used[fieldName] = 1
		}
		var typeInfo TypeInfo
		if b.opts.TypeResolver != nil {
			typeInfo = b.opts.TypeResolver.ResolveType(col.GoType, col.Nullable)
		} else {
			typeInfo = resolveType(col.GoType, col.Nullable)
		}
		fields = append(fields, helperField{name: fieldName, goType: typeInfo.GoType})
	}
	return &helperSpec{rowTypeName: rowTypeName, funcName: funcName, fields: fields}, nil
}

func (b *Builder) buildModelsFile(pkg string, models []*tableModel) (*goast.File, error) {
	file := &goast.File{Name: goast.NewIdent(pkg)}

	// Collect imports needed for custom types
	importSet := make(map[string]struct{})
	if b.opts.TypeResolver != nil && b.opts.TypeResolver.transformer != nil {
		for _, mapping := range b.opts.TypeResolver.transformer.GetCustomTypes() {
			if importPath, _, err := b.opts.TypeResolver.transformer.GetImportsForCustomType(mapping); err == nil {
				importSet[importPath] = struct{}{}
			}
		}
	}

	// Add imports if needed
	if len(importSet) > 0 {
		importDecls := make([]goast.Spec, 0, len(importSet))
		for importPath := range importSet {
			importDecls = append(importDecls, &goast.ImportSpec{
				Path: &goast.BasicLit{Kind: token.STRING, Value: strconv.Quote(importPath)},
			})
		}
		importDecl := &goast.GenDecl{Tok: token.IMPORT, Specs: importDecls}
		file.Decls = append(file.Decls, importDecl)
	}

	decls := make([]goast.Decl, 0, len(models))
	for _, mdl := range models {
		fields := make([]*goast.Field, 0, len(mdl.fields))
		for _, fld := range mdl.fields {
			expr, err := parser.ParseExpr(fld.goType)
			if err != nil {
				return nil, err
			}
			field := &goast.Field{
				Names: []*goast.Ident{goast.NewIdent(fld.fieldName)},
				Type:  expr,
			}
			if fld.jsonTag != "" {
				field.Tag = &goast.BasicLit{Kind: token.STRING, Value: fld.jsonTag}
			}
			fields = append(fields, field)
		}
		structType := &goast.StructType{Fields: &goast.FieldList{List: fields}}
		typeSpec := &goast.TypeSpec{Name: goast.NewIdent(mdl.typeName), Type: structType}
		decl := &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{typeSpec}}
		decls = append(decls, decl)
	}
	file.Decls = append(file.Decls, decls...)
	return file, nil
}

func (b *Builder) buildQuerierFile(pkg string, queries []queryInfo) (*goast.File, error) {
	file := &goast.File{Name: goast.NewIdent(pkg)}

	interfaceFields := make([]*goast.Field, 0, len(queries))
	for _, q := range queries {
		params := []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("ctx")}, Type: selector("context", "Context")}}
		for _, p := range q.params {
			expr, err := parser.ParseExpr(p.goType)
			if err != nil {
				return nil, err
			}
			paramType := expr
			if p.variadic {
				paramType = &goast.Ellipsis{Elt: expr}
			}
			params = append(params, &goast.Field{Names: []*goast.Ident{goast.NewIdent(p.name)}, Type: paramType})
		}
		results := []*goast.Field{}
		if q.returnType != "" {
			expr, err := parser.ParseExpr(q.returnType)
			if err != nil {
				return nil, err
			}
			results = append(results, &goast.Field{Type: expr})
		}
		errorType, _ := parser.ParseExpr("error")
		results = append(results, &goast.Field{Type: errorType})

		interfaceFields = append(interfaceFields, &goast.Field{
			Names: []*goast.Ident{goast.NewIdent(q.methodName)},
			Type: &goast.FuncType{
				Params:  &goast.FieldList{List: params},
				Results: &goast.FieldList{List: results},
			},
		})
	}

	querierType := &goast.TypeSpec{Name: goast.NewIdent("Querier"), Type: &goast.InterfaceType{Methods: &goast.FieldList{List: interfaceFields}}}
	querierDecl := &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{querierType}}

	dbtxMethods := []*goast.Field{
		{Names: []*goast.Ident{goast.NewIdent("ExecContext")}, Type: funcType([]*goast.Field{
			{Names: []*goast.Ident{goast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*goast.Ident{goast.NewIdent("query")}, Type: goast.NewIdent("string")},
			{Names: []*goast.Ident{goast.NewIdent("args")}, Type: &goast.Ellipsis{Elt: goast.NewIdent("any")}},
		}, []*goast.Field{{Type: selector("sql", "Result")}, {Type: goast.NewIdent("error")}})},
		{Names: []*goast.Ident{goast.NewIdent("QueryContext")}, Type: funcType([]*goast.Field{
			{Names: []*goast.Ident{goast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*goast.Ident{goast.NewIdent("query")}, Type: goast.NewIdent("string")},
			{Names: []*goast.Ident{goast.NewIdent("args")}, Type: &goast.Ellipsis{Elt: goast.NewIdent("any")}},
		}, []*goast.Field{{Type: selector("sql", "Rows")}, {Type: goast.NewIdent("error")}})},
		{Names: []*goast.Ident{goast.NewIdent("QueryRowContext")}, Type: funcType([]*goast.Field{
			{Names: []*goast.Ident{goast.NewIdent("ctx")}, Type: selector("context", "Context")},
			{Names: []*goast.Ident{goast.NewIdent("query")}, Type: goast.NewIdent("string")},
			{Names: []*goast.Ident{goast.NewIdent("args")}, Type: &goast.Ellipsis{Elt: goast.NewIdent("any")}},
		}, []*goast.Field{{Type: selector("sql", "Row")}})},
	}
	dbtxType := &goast.TypeSpec{Name: goast.NewIdent("DBTX"), Type: &goast.InterfaceType{Methods: &goast.FieldList{List: dbtxMethods}}}
	dbtxDecl := &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{dbtxType}}

	queriesStruct := &goast.TypeSpec{Name: goast.NewIdent("Queries"), Type: &goast.StructType{Fields: &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("db")}, Type: goast.NewIdent("DBTX")}}}}}
	queriesDecl := &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{queriesStruct}}

	newFunc := &goast.FuncDecl{
		Name: goast.NewIdent("New"),
		Type: &goast.FuncType{
			Params:  &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("db")}, Type: goast.NewIdent("DBTX")}}},
			Results: &goast.FieldList{List: []*goast.Field{{Type: &goast.StarExpr{X: goast.NewIdent("Queries")}}}},
		},
		Body: &goast.BlockStmt{List: []goast.Stmt{
			&goast.ReturnStmt{Results: []goast.Expr{&goast.UnaryExpr{Op: token.AND, X: &goast.CompositeLit{Type: goast.NewIdent("Queries"), Elts: []goast.Expr{
				&goast.KeyValueExpr{Key: goast.NewIdent("db"), Value: goast.NewIdent("db")},
			}}}}},
		}},
	}

	resultStruct := &goast.TypeSpec{Name: goast.NewIdent("QueryResult"), Type: &goast.StructType{Fields: &goast.FieldList{List: []*goast.Field{
		{Names: []*goast.Ident{goast.NewIdent("LastInsertID")}, Type: goast.NewIdent("int64")},
		{Names: []*goast.Ident{goast.NewIdent("RowsAffected")}, Type: goast.NewIdent("int64")},
	}}}}
	resultDecl := &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{resultStruct}}

	file.Decls = []goast.Decl{querierDecl, dbtxDecl, queriesDecl, newFunc, resultDecl}
	return file, nil
}

func (b *Builder) buildHelpersFile(pkg string, helpers []*helperSpec) (*goast.File, error) {
	file := &goast.File{Name: goast.NewIdent(pkg)}
	decls := make([]goast.Decl, 0, len(helpers)*2)
	for _, helper := range helpers {
		fields := make([]*goast.Field, 0, len(helper.fields))
		for _, fld := range helper.fields {
			expr, err := parser.ParseExpr(fld.goType)
			if err != nil {
				return nil, err
			}
			fields = append(fields, &goast.Field{Names: []*goast.Ident{goast.NewIdent(fld.name)}, Type: expr})
		}
		rowType := &goast.StructType{Fields: &goast.FieldList{List: fields}}
		rowSpec := &goast.TypeSpec{Name: goast.NewIdent(helper.rowTypeName), Type: rowType}
		decls = append(decls, &goast.GenDecl{Tok: token.TYPE, Specs: []goast.Spec{rowSpec}})

		stmts := make([]goast.Stmt, 0, 3)
		stmt, err := parseStmt("var item " + helper.rowTypeName)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		scanArgs := make([]string, 0, len(helper.fields))
		for _, fld := range helper.fields {
			scanArgs = append(scanArgs, "&item."+fld.name)
		}
		scanStmt := fmt.Sprintf("if err := rows.Scan(%s); err != nil {\nreturn item, err\n}", strings.Join(scanArgs, ", "))
		stmt, err = parseStmt(scanStmt)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		stmt, err = parseStmt("return item, nil")
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)

		funcDecl := &goast.FuncDecl{
			Name: goast.NewIdent(helper.funcName),
			Type: &goast.FuncType{
				Params:  &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("rows")}, Type: selector("sql", "Rows")}}},
				Results: &goast.FieldList{List: []*goast.Field{{Type: goast.NewIdent(helper.rowTypeName)}, {Type: goast.NewIdent("error")}}},
			},
			Body: &goast.BlockStmt{List: stmts},
		}
		decls = append(decls, funcDecl)
	}
	file.Decls = decls
	return file, nil
}

func (b *Builder) buildQueryFiles(pkg string, queries []queryInfo) ([]File, error) {
	files := make([]File, 0, len(queries))
	for _, q := range queries {
		file := &goast.File{Name: goast.NewIdent(pkg)}

		constSpec := &goast.ValueSpec{
			Names:  []*goast.Ident{goast.NewIdent(q.constName)},
			Type:   goast.NewIdent("string"),
			Values: []goast.Expr{stringLiteral(q.sqlLiteral)},
		}
		constDecl := &goast.GenDecl{Tok: token.CONST, Specs: []goast.Spec{constSpec}}

		funcDecl, err := b.buildQueryFunc(q)
		if err != nil {
			return nil, err
		}

		file.Decls = []goast.Decl{constDecl, funcDecl}
		files = append(files, File{Path: q.fileName, Node: file})
	}
	return files, nil
}

func (b *Builder) buildPreparedFile(pkg string, queries []queryInfo) (File, error) {
	importSet := map[string]struct{}{
		"context":      {},
		"database/sql": {},
	}
	if b.opts.Prepared.ThreadSafe {
		importSet["sync"] = struct{}{}
	}
	if b.opts.Prepared.EmitMetrics {
		importSet["time"] = struct{}{}
	}

	keys := make([]string, 0, len(importSet))
	for key := range importSet {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var buf strings.Builder
	fmt.Fprintf(&buf, "package %s\n\n", pkg)
	if len(keys) > 0 {
		fmt.Fprintf(&buf, "import (\n")
		for _, imp := range keys {
			fmt.Fprintf(&buf, "\t\"%s\"\n", imp)
		}
		fmt.Fprintf(&buf, ")\n\n")
	}

	fmt.Fprintf(&buf, "type PrepareDB interface {\n")
	fmt.Fprintf(&buf, "\tDBTX\n")
	fmt.Fprintf(&buf, "\tPrepareContext(ctx context.Context, query string) (*sql.Stmt, error)\n")
	fmt.Fprintf(&buf, "}\n\n")

	if b.opts.Prepared.EmitMetrics {
		fmt.Fprintf(&buf, "type PreparedMetricsRecorder interface {\n")
		fmt.Fprintf(&buf, "\tObservePreparedQuery(ctx context.Context, name string, duration time.Duration, err error)\n")
		fmt.Fprintf(&buf, "}\n\n")
		fmt.Fprintf(&buf, "type PreparedConfig struct {\n")
		fmt.Fprintf(&buf, "\tMetrics PreparedMetricsRecorder\n")
		fmt.Fprintf(&buf, "}\n\n")
	} else {
		fmt.Fprintf(&buf, "type PreparedConfig struct{}\n\n")
	}

	fmt.Fprintf(&buf, "type PreparedQueries struct {\n")
	fmt.Fprintf(&buf, "\tqueries *Queries\n")
	fmt.Fprintf(&buf, "\tdb PrepareDB\n")
	if b.opts.Prepared.EmitMetrics {
		fmt.Fprintf(&buf, "\tmetrics PreparedMetricsRecorder\n")
	}
	if b.opts.Prepared.ThreadSafe {
		fmt.Fprintf(&buf, "\tcloseOnce sync.Once\n")
		fmt.Fprintf(&buf, "\tcloseErr error\n")
	} else {
		fmt.Fprintf(&buf, "\tclosed bool\n")
	}
	for _, q := range queries {
		fmt.Fprintf(&buf, "\t%s *sql.Stmt\n", q.stmtField)
		if b.opts.Prepared.ThreadSafe {
			fmt.Fprintf(&buf, "\t%[1]sMu sync.Mutex\n", q.stmtField)
		}
	}
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func (p *PreparedQueries) Raw() *Queries {\n")
	fmt.Fprintf(&buf, "\treturn p.queries\n")
	fmt.Fprintf(&buf, "}\n\n")

	fmt.Fprintf(&buf, "func Prepare(ctx context.Context, db PrepareDB, cfg PreparedConfig) (*PreparedQueries, error) {\n")
	fmt.Fprintf(&buf, "\tpq := &PreparedQueries{\n")
	fmt.Fprintf(&buf, "\t\tqueries: New(db),\n")
	fmt.Fprintf(&buf, "\t\tdb:       db,\n")
	if b.opts.Prepared.EmitMetrics {
		fmt.Fprintf(&buf, "\t\tmetrics: cfg.Metrics,\n")
	}
	fmt.Fprintf(&buf, "\t}\n")
	if !b.opts.Prepared.ThreadSafe {
		fmt.Fprintf(&buf, "\tprepared := make([]*sql.Stmt, 0, %d)\n", len(queries))
		for _, q := range queries {
			fmt.Fprintf(&buf, "\tstmt, err := db.PrepareContext(ctx, %s)\n", q.constName)
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\tfor _, preparedStmt := range prepared {\n")
			fmt.Fprintf(&buf, "\t\t\tpreparedStmt.Close()\n")
			fmt.Fprintf(&buf, "\t\t}\n")
			fmt.Fprintf(&buf, "\t\treturn nil, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tprepared = append(prepared, stmt)\n")
			fmt.Fprintf(&buf, "\tpq.%s = stmt\n", q.stmtField)
		}
	}
	fmt.Fprintf(&buf, "\treturn pq, nil\n")
	fmt.Fprintf(&buf, "}\n\n")

	if b.opts.Prepared.ThreadSafe {
		fmt.Fprintf(&buf, "func (p *PreparedQueries) Close() error {\n")
		fmt.Fprintf(&buf, "\tp.closeOnce.Do(func() {\n")
		fmt.Fprintf(&buf, "\t\tvar err error\n")
		for _, q := range queries {
			fmt.Fprintf(&buf, "\t\tp.%[1]sMu.Lock()\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\tif p.%[1]s != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\t\tif closeErr := p.%[1]s.Close(); err == nil && closeErr != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\t\t\terr = closeErr\n")
			fmt.Fprintf(&buf, "\t\t\t}\n")
			fmt.Fprintf(&buf, "\t\t\tp.%[1]s = nil\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\t}\n")
			fmt.Fprintf(&buf, "\t\tp.%[1]sMu.Unlock()\n", q.stmtField)
		}
		fmt.Fprintf(&buf, "\t\tp.closeErr = err\n")
		fmt.Fprintf(&buf, "\t})\n")
		fmt.Fprintf(&buf, "\treturn p.closeErr\n")
		fmt.Fprintf(&buf, "}\n\n")
	} else {
		fmt.Fprintf(&buf, "func (p *PreparedQueries) Close() error {\n")
		fmt.Fprintf(&buf, "\tif p.closed {\n")
		fmt.Fprintf(&buf, "\t\treturn nil\n")
		fmt.Fprintf(&buf, "\t}\n")
		fmt.Fprintf(&buf, "\tp.closed = true\n")
		fmt.Fprintf(&buf, "\tvar err error\n")
		for _, q := range queries {
			fmt.Fprintf(&buf, "\tif p.%[1]s != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\tif closeErr := p.%[1]s.Close(); err == nil && closeErr != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\t\terr = closeErr\n")
			fmt.Fprintf(&buf, "\t\t}\n")
			fmt.Fprintf(&buf, "\t\tp.%[1]s = nil\n", q.stmtField)
			fmt.Fprintf(&buf, "\t}\n")
		}
		fmt.Fprintf(&buf, "\treturn err\n")
		fmt.Fprintf(&buf, "}\n\n")
	}

	if b.opts.Prepared.ThreadSafe {
		for _, q := range queries {
			fmt.Fprintf(&buf, "func (p *PreparedQueries) %s(ctx context.Context) (*sql.Stmt, error) {\n", q.prepareFn)
			fmt.Fprintf(&buf, "\tif stmt := p.%s; stmt != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\treturn stmt, nil\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tp.%[1]sMu.Lock()\n", q.stmtField)
			fmt.Fprintf(&buf, "\tdefer p.%[1]sMu.Unlock()\n", q.stmtField)
			fmt.Fprintf(&buf, "\tif stmt := p.%s; stmt != nil {\n", q.stmtField)
			fmt.Fprintf(&buf, "\t\treturn stmt, nil\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tstmt, err := p.db.PrepareContext(ctx, %s)\n", q.constName)
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn nil, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tp.%s = stmt\n", q.stmtField)
			fmt.Fprintf(&buf, "\treturn stmt, nil\n")
			fmt.Fprintf(&buf, "}\n\n")
		}
	}

	for _, q := range queries {
		if q.docComment != "" {
			fmt.Fprintf(&buf, "// %s\n", q.docComment)
		}
		fmt.Fprintf(&buf, "func (p *PreparedQueries) %s(ctx context.Context", q.methodName)
		for _, param := range q.params {
			if param.variadic {
				fmt.Fprintf(&buf, ", %s ...%s", param.name, param.goType)
				continue
			}
			fmt.Fprintf(&buf, ", %s %s", param.name, param.goType)
		}
		fmt.Fprintf(&buf, ") (")
		if q.returnType != "" {
			fmt.Fprintf(&buf, "%s, ", q.returnType)
		}
		fmt.Fprintf(&buf, "error) {\n")
		if b.opts.Prepared.ThreadSafe {
			fmt.Fprintf(&buf, "\tstmt, err := p.%s(ctx)\n", q.prepareFn)
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn %s, err\n", q.returnZero)
			fmt.Fprintf(&buf, "\t}\n")
		} else {
			fmt.Fprintf(&buf, "\tstmt := p.%s\n", q.stmtField)
		}

		if b.opts.Prepared.EmitMetrics {
			fmt.Fprintf(&buf, "\trecorder := p.metrics\n")
			fmt.Fprintf(&buf, "\tvar start time.Time\n")
			fmt.Fprintf(&buf, "\tif recorder != nil {\n")
			fmt.Fprintf(&buf, "\t\tstart = time.Now()\n")
			fmt.Fprintf(&buf, "\t}\n")
		}

		for _, param := range q.params {
			if !param.variadic {
				continue
			}
			fmt.Fprintf(&buf, "\t%[1]s := make([]any, len(%[2]s))\n", param.sliceName, param.name)
			fmt.Fprintf(&buf, "\tfor i := range %[1]s {\n", param.name)
			fmt.Fprintf(&buf, "\t\t%[1]s[i] = %[2]s[i]\n", param.sliceName, param.name)
			fmt.Fprintf(&buf, "\t}\n")
		}

		switch q.command {
		case block.CommandExec:
			fmt.Fprintf(&buf, "\tres, err := stmt.ExecContext(ctx")
			for _, arg := range q.args {
				fmt.Fprintf(&buf, ", %s", arg)
			}
			fmt.Fprintf(&buf, ")\n")
			if b.opts.Prepared.EmitMetrics {
				fmt.Fprintf(&buf, "\tif recorder != nil {\n")
				fmt.Fprintf(&buf, "\t\trecorder.ObservePreparedQuery(ctx, %q, time.Since(start), err)\n", q.metricsKey)
				fmt.Fprintf(&buf, "\t}\n")
			}
			fmt.Fprintf(&buf, "\treturn res, err\n")
		case block.CommandExecResult:
			fmt.Fprintf(&buf, "\tres, err := stmt.ExecContext(ctx")
			for _, arg := range q.args {
				fmt.Fprintf(&buf, ", %s", arg)
			}
			fmt.Fprintf(&buf, ")\n")
			if b.opts.Prepared.EmitMetrics {
				fmt.Fprintf(&buf, "\tif recorder != nil {\n")
				fmt.Fprintf(&buf, "\t\trecorder.ObservePreparedQuery(ctx, %q, time.Since(start), err)\n", q.metricsKey)
				fmt.Fprintf(&buf, "\t}\n")
			}
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn %s, err\n", q.returnZero)
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tresult := QueryResult{}\n")
			fmt.Fprintf(&buf, "\tif v, err := res.LastInsertId(); err == nil {\n")
			fmt.Fprintf(&buf, "\t\tresult.LastInsertID = v\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tif v, err := res.RowsAffected(); err == nil {\n")
			fmt.Fprintf(&buf, "\t\tresult.RowsAffected = v\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\treturn result, nil\n")
		case block.CommandOne:
			fmt.Fprintf(&buf, "\trows, err := stmt.QueryContext(ctx")
			for _, arg := range q.args {
				fmt.Fprintf(&buf, ", %s", arg)
			}
			fmt.Fprintf(&buf, ")\n")
			if b.opts.Prepared.EmitMetrics {
				fmt.Fprintf(&buf, "\tif recorder != nil {\n")
				fmt.Fprintf(&buf, "\t\trecorder.ObservePreparedQuery(ctx, %q, time.Since(start), err)\n", q.metricsKey)
				fmt.Fprintf(&buf, "\t}\n")
			}
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn %s, err\n", q.returnZero)
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tdefer rows.Close()\n")
			fmt.Fprintf(&buf, "\tif !rows.Next() {\n")
			fmt.Fprintf(&buf, "\t\tif err := rows.Err(); err != nil {\n")
			fmt.Fprintf(&buf, "\t\t\treturn %s, err\n", q.returnZero)
			fmt.Fprintf(&buf, "\t\t}\n")
			fmt.Fprintf(&buf, "\t\treturn %s, sql.ErrNoRows\n", q.returnZero)
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\titem, err := %s(rows)\n", q.helper.funcName)
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn item, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tif err := rows.Err(); err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn item, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\treturn item, nil\n")
		case block.CommandMany:
			fmt.Fprintf(&buf, "\trows, err := stmt.QueryContext(ctx")
			for _, arg := range q.args {
				fmt.Fprintf(&buf, ", %s", arg)
			}
			fmt.Fprintf(&buf, ")\n")
			if b.opts.Prepared.EmitMetrics {
				fmt.Fprintf(&buf, "\tif recorder != nil {\n")
				fmt.Fprintf(&buf, "\t\trecorder.ObservePreparedQuery(ctx, %q, time.Since(start), err)\n", q.metricsKey)
				fmt.Fprintf(&buf, "\t}\n")
			}
			fmt.Fprintf(&buf, "\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn nil, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tdefer rows.Close()\n")
			if b.opts.EmitEmptySlices {
				fmt.Fprintf(&buf, "\titems := make([]%s, 0)\n", q.rowType)
			} else {
				fmt.Fprintf(&buf, "\tvar items []%s\n", q.rowType)
			}
			fmt.Fprintf(&buf, "\tfor rows.Next() {\n")
			fmt.Fprintf(&buf, "\t\titem, err := %s(rows)\n", q.helper.funcName)
			fmt.Fprintf(&buf, "\t\tif err != nil {\n")
			fmt.Fprintf(&buf, "\t\t\treturn nil, err\n")
			fmt.Fprintf(&buf, "\t\t}\n")
			fmt.Fprintf(&buf, "\t\titems = append(items, item)\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\tif err := rows.Err(); err != nil {\n")
			fmt.Fprintf(&buf, "\t\treturn nil, err\n")
			fmt.Fprintf(&buf, "\t}\n")
			fmt.Fprintf(&buf, "\treturn items, nil\n")
		}
		fmt.Fprintf(&buf, "}\n\n")
	}

	source := buf.String()
	formatted, err := imports.Process("", []byte(source), nil)
	if err != nil {
		return File{}, err
	}
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", formatted, parser.ParseComments)
	if err != nil {
		return File{}, err
	}
	return File{Path: "prepared.gen.go", Node: node, Raw: formatted}, nil
}

func (b *Builder) buildQueryFunc(q queryInfo) (*goast.FuncDecl, error) {
	params := []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("ctx")}, Type: selector("context", "Context")}}
	for _, p := range q.params {
		expr, err := parser.ParseExpr(p.goType)
		if err != nil {
			return nil, err
		}
		paramType := expr
		if p.variadic {
			paramType = &goast.Ellipsis{Elt: expr}
		}
		params = append(params, &goast.Field{Names: []*goast.Ident{goast.NewIdent(p.name)}, Type: paramType})
	}

	results := []*goast.Field{}
	if q.returnType != "" {
		expr, err := parser.ParseExpr(q.returnType)
		if err != nil {
			return nil, err
		}
		results = append(results, &goast.Field{Type: expr})
	}
	errorType, _ := parser.ParseExpr("error")
	results = append(results, &goast.Field{Type: errorType})

	body := make([]goast.Stmt, 0)

	// Handle dynamic slices
	hasDynamic := false
	for _, p := range q.params {
		if p.isDynamicSlice {
			hasDynamic = true
			break
		}
	}

	if hasDynamic {
		body = append(body, mustParseStmt("query := "+q.constName))

		// Generate slice expansion strings
		// query = strings.Replace(query, "/*SLICE:ids*/", strings.Repeat("?, ", len(ids))[:len(strings.Repeat("?, ", len(ids)))-2], 1)
		// Or simpler:
		// var queryOutput strings.Builder
		// ...
		// But replacing marker is cleaner for AST generation.

		for _, p := range q.params {
			if !p.isDynamicSlice {
				continue
			}
			// strings.Repeat("?,", len(ids))
			// We need to strip trailing comma.
			// strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
			// Generate: query = strings.Replace(query, marker, strings.TrimRight(strings.Repeat("?,", len(p.name)), ","), 1)

			replaceStmt := fmt.Sprintf(`query = strings.Replace(query, "%s", strings.TrimRight(strings.Repeat("?,", len(%s)), ","), 1)`, p.marker, p.name)
			body = append(body, mustParseStmt(replaceStmt))
		}
	}

	// Handle variadic args expansion (for explicit IN (?,?) support)
	for _, p := range q.params {
		if !p.variadic {
			continue
		}
		makeStmt := mustParseStmt(fmt.Sprintf("%s := make([]any, len(%s))", p.sliceName, p.name))
		loopStmt := mustParseStmt(fmt.Sprintf("for i := range %s {\n%s[i] = %s[i]\n}", p.name, p.sliceName, p.name))
		body = append(body, makeStmt, loopStmt)
	}

	// Collect arguments for query call
	// If dynamic, we need to build args slice dynamically
	callArgsName := ""
	if hasDynamic {
		callArgsName = "args"
		body = append(body, mustParseStmt("args := []any{}"))
		for _, p := range q.params {
			switch {
			case p.isDynamicSlice:
				body = append(body, mustParseStmt(fmt.Sprintf("for _, v := range %s {\nargs = append(args, v)\n}", p.name)))
			case p.variadic:
				body = append(body, mustParseStmt(fmt.Sprintf("args = append(args, %s...)", p.sliceName)))
			default:
				body = append(body, mustParseStmt(fmt.Sprintf("args = append(args, %s)", p.name)))
			}
		}
	}

	switch q.command {
	case block.CommandExec:
		if hasDynamic {
			body = append(body, mustParseStmt(fmt.Sprintf("return q.db.ExecContext(ctx, query, %s...)", callArgsName)))
		} else {
			args := append([]string{"ctx", q.constName}, q.args...)
			body = append(body, mustParseStmt(fmt.Sprintf("return q.db.ExecContext(%s)", strings.Join(args, ", "))))
		}
	case block.CommandExecResult:
		if hasDynamic {
			body = append(body, mustParseStmt(fmt.Sprintf("res, err := q.db.ExecContext(ctx, query, %s...)", callArgsName)))
		} else {
			args := append([]string{"ctx", q.constName}, q.args...)
			body = append(body, mustParseStmt(fmt.Sprintf("res, err := q.db.ExecContext(%s)", strings.Join(args, ", "))))
		}
		body = append(body, mustParseStmt("if err != nil {\nreturn QueryResult{}, err\n}"))
		body = append(body, mustParseStmt("result := QueryResult{}"))
		body = append(body, mustParseStmt("if v, err := res.LastInsertId(); err == nil {\nresult.LastInsertID = v\n}"))
		body = append(body, mustParseStmt("if v, err := res.RowsAffected(); err == nil {\nresult.RowsAffected = v\n}"))
		body = append(body, mustParseStmt("return result, nil"))
	default:
		if hasDynamic {
			body = append(body, mustParseStmt(fmt.Sprintf("rows, err := q.db.QueryContext(ctx, query, %s...)", callArgsName)))
		} else {
			args := append([]string{"ctx", q.constName}, q.args...)
			body = append(body, mustParseStmt(fmt.Sprintf("rows, err := q.db.QueryContext(%s)", strings.Join(args, ", "))))
		}
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
			if b.opts.EmitEmptySlices {
				body = append(body, mustParseStmt("items := make([]"+q.rowType+", 0)"))
			} else {
				body = append(body, mustParseStmt("var items []"+q.rowType))
			}
			loop := &goast.ForStmt{
				Cond: &goast.CallExpr{Fun: &goast.SelectorExpr{X: goast.NewIdent("rows"), Sel: goast.NewIdent("Next")}},
				Body: &goast.BlockStmt{List: []goast.Stmt{
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

	funcDecl := &goast.FuncDecl{
		Recv: &goast.FieldList{List: []*goast.Field{{Names: []*goast.Ident{goast.NewIdent("q")}, Type: &goast.StarExpr{X: goast.NewIdent("Queries")}}}},
		Name: goast.NewIdent(q.methodName),
		Type: &goast.FuncType{Params: &goast.FieldList{List: params}, Results: &goast.FieldList{List: results}},
		Body: &goast.BlockStmt{List: body},
	}

	if q.docComment != "" {
		funcDecl.Doc = &goast.CommentGroup{List: []*goast.Comment{{Text: "// " + q.docComment}}}
	}

	return funcDecl, nil
}

func selector(pkg, name string) *goast.SelectorExpr {
	return &goast.SelectorExpr{X: goast.NewIdent(pkg), Sel: goast.NewIdent(name)}
}

func funcType(params []*goast.Field, results []*goast.Field) goast.Expr {
	return &goast.FuncType{Params: &goast.FieldList{List: params}, Results: &goast.FieldList{List: results}}
}

func stringLiteral(value string) goast.Expr {
	if !strings.Contains(value, "`") {
		return &goast.BasicLit{Kind: token.STRING, Value: "`" + value + "`"}
	}
	return &goast.BasicLit{Kind: token.STRING, Value: strconv.Quote(value)}
}

func parseStmt(code string) (goast.Stmt, error) {
	src := "package p\nfunc _() {\n" + code + "\n}\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, fmt.Errorf("parse statement: %w", err)
	}
	if len(file.Decls) == 0 {
		return nil, errors.New("no declarations parsed")
	}
	fn, ok := file.Decls[0].(*goast.FuncDecl)
	if !ok || fn.Body == nil || len(fn.Body.List) == 0 {
		return nil, errors.New("parsed function missing body")
	}
	return fn.Body.List[0], nil
}

// mustParseStmt parses a Go statement from code string.
// It panics on error because all code strings are hardcoded templates
// that should always be valid Go syntax. A panic indicates a programming
// error in the code generator itself, not a user input error.
func mustParseStmt(code string) goast.Stmt {
	stmt, err := parseStmt(code)
	if err != nil {
		panic(fmt.Errorf("failed to parse generated code %q: %w", code, err))
	}
	return stmt
}
