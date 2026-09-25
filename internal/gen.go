package golang

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/plugin-sdk-go/plugin"
	"github.com/sqlc-dev/plugin-sdk-go/sdk"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

type tmplCtx struct {
	Q           string
	Package     string
	SQLDriver   opts.SQLDriver
	Enums       []Enum
	Structs     []Struct
	GoQueries   []Query
	Scanners    []rowScanner
	SqlcVersion string

	// TODO: Race conditions
	SourceName string

	EmitJSONTags        bool
	JsonTagsIDUppercase bool
	EmitDBTags          bool
	EmitEmptySlices     bool
	EmitEnumValidMethod bool
	EmitAllEnumValues   bool
	EmitMockExecutor    bool
	UsesCopyFrom        bool
	UsesBatch           bool
	OmitSqlcVersion     bool
	BuildTags           string

	// Package qualifiers for query struct pattern
	PackageQualifier       string
	ModelsPackageQualifier string
}

func (t *tmplCtx) OutputQuery(sourceName string) bool {
	return t.SourceName == sourceName
}

func (t *tmplCtx) codegenQueryMethod(q Query) string {
	db := "q.db"

	switch q.Cmd {
	case ":one":
		return db + ".QueryRowContext"

	case ":many":
		return db + ".QueryContext"

	default:
		return db + ".ExecContext"
	}
}

func (t *tmplCtx) codegenQueryRetval(q Query) (string, error) {
	switch q.Cmd {
	case ":one":
		return "row :=", nil
	case ":many":
		return "rows, err :=", nil
	case ":exec":
		return "_, err :=", nil
	case ":execrows", ":execlastid":
		return "result, err :=", nil
	case ":execresult":
		return "return", nil
	default:
		return "", fmt.Errorf("unhandled q.Cmd case %q", q.Cmd)
	}
}

func Generate(ctx context.Context, req *plugin.GenerateRequest) (*plugin.GenerateResponse, error) {
	options, err := opts.Parse(req)
	if err != nil {
		return nil, err
	}

	if err := opts.ValidateOpts(options); err != nil {
		return nil, err
	}

	enums := buildEnums(req, options)
	structs := buildStructs(req, options)
	queries, err := buildQueries(req, options, structs)
	if err != nil {
		return nil, err
	}

	if options.OmitUnusedStructs {
		enums, structs = filterUnusedStructs(options, enums, structs, queries)
	}

	scanners := assignScanners(queries)

	if err := validate(options, enums, structs, queries); err != nil {
		return nil, err
	}

	return generate(req, options, enums, structs, queries, scanners)
}

func validate(options *opts.Options, enums []Enum, structs []Struct, queries []Query) error {
	enumNames := make(map[string]struct{})
	for _, enum := range enums {
		enumNames[enum.Name] = struct{}{}
		enumNames["Null"+enum.Name] = struct{}{}
	}
	structNames := make(map[string]struct{})
	for _, struckt := range structs {
		if _, ok := enumNames[struckt.Name]; ok {
			return fmt.Errorf("struct name conflicts with enum name: %s", struckt.Name)
		}
		structNames[struckt.Name] = struct{}{}
	}
	if err := validateGeneratedNames(options, queries); err != nil {
		return err
	}
	if !options.EmitExportedQueries {
		return nil
	}
	for _, query := range queries {
		if _, ok := enumNames[query.ConstantName]; ok {
			return fmt.Errorf("query constant name conflicts with enum name: %s", query.ConstantName)
		}
		if _, ok := structNames[query.ConstantName]; ok {
			return fmt.Errorf("query constant name conflicts with struct name: %s", query.ConstantName)
		}
	}
	return nil
}

// runtimeNames are package-level identifiers declared by the pgx and stdlib dbCode.tmpl templates.
var runtimeNames = map[string]struct{}{
	"DBTX": {}, "DBTXWithTx": {}, "Query": {}, "ExecFunc": {}, "QueryExecutor": {}, "DB": {}, "New": {},
	"Row": {}, "RowScanner": {}, "ScanValue": {}, "Statement": {}, "NewStatement": {},
	"Executor": {}, "NewExecutor": {}, "Step": {}, "Expect": {},
	"StubExecutor": {}, "NewStubExecutor": {}, "ErrBatchAlreadyClosed": {},
	"oneQuery": {}, "manyQuery": {}, "execQuery": {}, "execResultQuery": {}, "execLastIDQuery": {},
}

// queryTypeName is the per-query type generated for :copyfrom and :batch* queries.
func queryTypeName(q Query) string {
	if q.Cmd == metadata.CmdCopyFrom || usesBatch([]Query{q}) {
		return sdk.LowerTitle(q.MethodName) + "Query"
	}
	return ""
}

func validateGeneratedNames(options *opts.Options, queries []Query) error {
	scannerNames := map[string]struct{}{}
	queryTypeNames := map[string]struct{}{}
	for _, q := range queries {
		if q.ScannerName != "" {
			scannerNames[q.ScannerName] = struct{}{}
		}
		if name := queryTypeName(q); name != "" {
			queryTypeNames[name] = struct{}{}
		}
	}
	separateQueries := options.OutputQueriesPackage != "" && options.OutputQueriesPackage != options.Package
	for _, q := range queries {
		if _, ok := scannerNames[q.ConstantName]; ok {
			return fmt.Errorf("query %s: generated name %q conflicts with row scanner", q.MethodName, q.ConstantName)
		}
		typeName := queryTypeName(q)
		inDBPackage := !separateQueries || typeName != ""
		if !inDBPackage {
			continue
		}
		if _, ok := queryTypeNames[q.ConstantName]; ok {
			return fmt.Errorf("query %s: generated name %q conflicts with generated query type", q.MethodName, q.ConstantName)
		}
		for _, name := range []string{q.MethodName, q.ConstantName, typeName} {
			if _, ok := runtimeNames[name]; ok {
				return fmt.Errorf("query %s: generated name %q conflicts with runtime identifier", q.MethodName, name)
			}
		}
	}
	return nil
}

// rowScanner is a shared row scanner, emitted once in the query file named by SourceName.
type rowScanner struct {
	Name       string
	SourceName string
	Ret        QueryValue
}

// assignScanners sets ScannerName on :one and :many queries whose rows ScanValue cannot scan
// (structs and pq.Array scalars) and returns one scanner per row type.
func assignScanners(queries []Query) []rowScanner {
	var scanners []rowScanner
	seen := map[string]struct{}{}
	for i := range queries {
		q := &queries[i]
		if q.Cmd != metadata.CmdOne && q.Cmd != metadata.CmdMany {
			continue
		}
		switch {
		case q.Ret.IsStruct():
			q.ScannerName = "scan" + q.Ret.Struct.Name
		case strings.Contains(q.Ret.Scan(), "pq.Array"):
			q.ScannerName = "scan" + q.MethodName
		default:
			continue
		}
		if _, ok := seen[q.ScannerName]; ok {
			continue
		}
		seen[q.ScannerName] = struct{}{}
		scanners = append(scanners, rowScanner{Name: q.ScannerName, SourceName: q.SourceName, Ret: q.Ret})
	}
	return scanners
}

func generate(req *plugin.GenerateRequest, options *opts.Options, enums []Enum, structs []Struct, queries []Query, scanners []rowScanner) (*plugin.GenerateResponse, error) {
	i := &importer{
		Options:  options,
		Queries:  queries,
		Enums:    enums,
		Structs:  structs,
		Scanners: scanners,
	}

	// Package qualifiers for query struct templates
	packageQualifier := ""
	modelsPackageQualifier := ""
	if options.OutputQueriesPackage != "" && options.OutputQueriesPackage != options.Package {
		packageQualifier = options.Package + "."
	}
	if options.ModelsPackageImportPath != "" {
		modelsPackageQualifier = options.OutputModelsPackage + "."
	}

	tctx := tmplCtx{
		EmitJSONTags:           options.EmitJsonTags,
		JsonTagsIDUppercase:    options.JsonTagsIdUppercase,
		EmitDBTags:             options.EmitDbTags,
		EmitEmptySlices:        options.EmitEmptySlices,
		EmitEnumValidMethod:    options.EmitEnumValidMethod,
		EmitAllEnumValues:      options.EmitAllEnumValues,
		EmitMockExecutor:       options.EmitMockExecutor,
		UsesCopyFrom:           usesCopyFrom(queries),
		UsesBatch:              usesBatch(queries),
		SQLDriver:              parseDriver(options.SqlPackage),
		Q:                      "`",
		Package:                options.Package,
		Enums:                  enums,
		Structs:                structs,
		SqlcVersion:            req.SqlcVersion,
		BuildTags:              options.BuildTags,
		OmitSqlcVersion:        options.OmitSqlcVersion,
		PackageQualifier:       packageQualifier,
		ModelsPackageQualifier: modelsPackageQualifier,
		Scanners:               scanners,
	}

	if tctx.UsesCopyFrom && !tctx.SQLDriver.IsPGX() && options.SqlDriver != string(opts.SQLDriverGoSQLDriverMySQL) {
		return nil, errors.New(":copyfrom is only supported by pgx and github.com/go-sql-driver/mysql")
	}

	if tctx.UsesCopyFrom && options.SqlDriver == string(opts.SQLDriverGoSQLDriverMySQL) {
		if err := checkNoTimesForMySQLCopyFrom(queries); err != nil {
			return nil, err
		}
		tctx.SQLDriver = opts.SQLDriverGoSQLDriverMySQL
	}

	if tctx.UsesBatch && !tctx.SQLDriver.IsPGX() {
		return nil, errors.New(":batch* commands are only supported by pgx")
	}

	funcMap := template.FuncMap{
		"lowerTitle": sdk.LowerTitle,
		"comment":    sdk.DoubleSlashComment,
		"escape":     sdk.EscapeBacktick,
		"imports":    i.Imports,
		"hasImports": i.HasImports,
		"hasPrefix":  strings.HasPrefix,
		"trimPrefix": strings.TrimPrefix,

		// These methods are Go specific, they do not belong in the codegen package
		// (as that is language independent)
		"queryMethod": tctx.codegenQueryMethod,
		"queryRetval": tctx.codegenQueryRetval,
	}

	tmpl := template.Must(
		template.New("table").
			Funcs(funcMap).
			ParseFS(
				templates,
				"templates/*.tmpl",
				"templates/*/*.tmpl",
			),
	)

	output := map[string]string{}

	execute := func(name, packageName, templateName string) error {
		imports := i.Imports(name)
		replacedQueries := replaceConflictedArg(imports, queries)

		var b bytes.Buffer
		w := bufio.NewWriter(&b)
		tctx.SourceName = name
		tctx.GoQueries = replacedQueries
		tctx.Package = packageName
		err := tmpl.ExecuteTemplate(w, templateName, &tctx)
		if err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
		code, err := format.Source(b.Bytes())
		if err != nil {
			lines := strings.Split(b.String(), "\n")
			start := max(0, 420)
			end := min(len(lines), 435)
			for i := start; i < end; i++ {
				fmt.Fprintf(os.Stderr, "%d: %s\n", i+1, lines[i])
			}
			return fmt.Errorf("source error: %w", err)
		}

		if templateName == "queryFile" {
			if options.OutputQueryFilesDirectory != "" {
				name = filepath.Join(options.OutputQueryFilesDirectory, name)
			}
			if options.OutputFilesSuffix != "" {
				name += options.OutputFilesSuffix
			}
		}

		if !strings.HasSuffix(name, ".go") {
			name += ".go"
		}
		output[name] = string(code)
		return nil
	}

	dbFileName := "db.go"
	if options.OutputDbFileName != "" {
		dbFileName = options.OutputDbFileName
	}
	modelsFileName := "models.go"
	if options.OutputModelsFileName != "" {
		modelsFileName = options.OutputModelsFileName
	}
	copyfromFileName := "copyfrom.go"
	if options.OutputCopyfromFileName != "" {
		copyfromFileName = options.OutputCopyfromFileName
	}

	batchFileName := "batch.go"
	if options.OutputBatchFileName != "" {
		batchFileName = options.OutputBatchFileName
	}

	modelsPackageName := options.Package
	if options.OutputModelsPackage != "" {
		modelsPackageName = options.OutputModelsPackage
	}

	queriesPackageName := options.Package
	if options.OutputQueriesPackage != "" {
		queriesPackageName = options.OutputQueriesPackage
	}

	if err := execute(dbFileName, options.Package, "dbFile"); err != nil {
		return nil, err
	}
	if err := execute(modelsFileName, modelsPackageName, "modelsFile"); err != nil {
		return nil, err
	}
	if tctx.UsesCopyFrom {
		if err := execute(copyfromFileName, options.Package, "copyfromFile"); err != nil {
			return nil, err
		}
	}
	if tctx.UsesBatch {
		if err := execute(batchFileName, options.Package, "batchFile"); err != nil {
			return nil, err
		}
	}

	files := map[string]struct{}{}
	for _, gq := range queries {
		files[gq.SourceName] = struct{}{}
	}

	for source := range files {
		if err := execute(source, queriesPackageName, "queryFile"); err != nil {
			return nil, err
		}
	}
	resp := plugin.GenerateResponse{}

	for filename, code := range output {
		resp.Files = append(resp.Files, &plugin.File{
			Name:     filename,
			Contents: []byte(code),
		})
	}

	return &resp, nil
}

func usesCopyFrom(queries []Query) bool {
	for _, q := range queries {
		if q.Cmd == metadata.CmdCopyFrom {
			return true
		}
	}
	return false
}

func usesBatch(queries []Query) bool {
	for _, q := range queries {
		for _, cmd := range []string{metadata.CmdBatchExec, metadata.CmdBatchMany, metadata.CmdBatchOne} {
			if q.Cmd == cmd {
				return true
			}
		}
	}
	return false
}

func checkNoTimesForMySQLCopyFrom(queries []Query) error {
	for _, q := range queries {
		if q.Cmd != metadata.CmdCopyFrom {
			continue
		}
		for _, f := range q.Arg.CopyFromMySQLFields() {
			if f.Type == "time.Time" {
				return fmt.Errorf("values with a timezone are not yet supported")
			}
		}
	}
	return nil
}

func filterUnusedStructs(options *opts.Options, enums []Enum, structs []Struct, queries []Query) ([]Enum, []Struct) {
	keepTypes := make(map[string]struct{})

	for _, query := range queries {
		if !query.Arg.isEmpty() {
			keepTypes[query.Arg.Type()] = struct{}{}
			if query.Arg.IsStruct() {
				for _, field := range query.Arg.Struct.Fields {
					keepTypes[field.Type] = struct{}{}
				}
			}
		}
		if query.hasRetType() {
			keepTypes[query.Ret.Type()] = struct{}{}
			if query.Ret.IsStruct() {
				for _, field := range query.Ret.Struct.Fields {
					keepTypes[field.Type] = struct{}{}
					for _, embedField := range field.EmbedFields {
						keepTypes[embedField.Type] = struct{}{}
					}
				}
			}
		}
	}

	qualify := func(name string) string {
		if options.ModelsPackageImportPath != "" {
			return options.OutputModelsPackage + "." + name
		}
		return name
	}
	keepEnums := make([]Enum, 0, len(enums))
	for _, enum := range enums {
		_, keep := keepTypes[qualify(enum.Name)]
		_, keepNull := keepTypes[qualify("Null"+enum.Name)]
		_, keepPtr := keepTypes["*"+qualify(enum.Name)]
		if keep || keepNull || keepPtr {
			keepEnums = append(keepEnums, enum)
		}
	}

	keepStructs := make([]Struct, 0, len(structs))
	for _, st := range structs {
		if _, ok := keepTypes[st.Type()]; ok {
			keepStructs = append(keepStructs, st)
		}
	}

	return keepEnums, keepStructs
}
