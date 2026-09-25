package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/metadata"
	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

func TestValidateGeneratedNames(t *testing.T) {
	tests := []struct {
		name    string
		options opts.Options
		query   Query
		wantErr bool
	}{
		{
			name:    "regular query",
			options: opts.Options{Package: "db"},
			query:   Query{Cmd: metadata.CmdOne, MethodName: "GetUser", ConstantName: "getUser"},
		},
		{
			name:    "method name collides",
			options: opts.Options{Package: "db"},
			query:   Query{Cmd: metadata.CmdOne, MethodName: "Expect", ConstantName: "expect"},
			wantErr: true,
		},
		{
			name:    "constant name collides",
			options: opts.Options{Package: "db"},
			query:   Query{Cmd: metadata.CmdMany, MethodName: "OneQuery", ConstantName: "oneQuery"},
			wantErr: true,
		},
		{
			name:    "copyfrom query type collides",
			options: opts.Options{Package: "db"},
			query:   Query{Cmd: metadata.CmdCopyFrom, MethodName: "Exec", ConstantName: "exec"},
			wantErr: true,
		},
		{
			name:    "separate queries package",
			options: opts.Options{Package: "db", OutputQueriesPackage: "queries"},
			query:   Query{Cmd: metadata.CmdOne, MethodName: "New", ConstantName: "new"},
		},
		{
			name:    "copyfrom stays in db package",
			options: opts.Options{Package: "db", OutputQueriesPackage: "queries"},
			query:   Query{Cmd: metadata.CmdCopyFrom, MethodName: "New", ConstantName: "new"},
			wantErr: true,
		},
		{
			name:    "batch stays in db package",
			options: opts.Options{Package: "db", OutputQueriesPackage: "queries"},
			query:   Query{Cmd: metadata.CmdBatchExec, MethodName: "DB", ConstantName: "dB"},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateGeneratedNames(&tc.options, []Query{tc.query})
			if (err != nil) != tc.wantErr {
				t.Errorf("validateGeneratedNames() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateGeneratedNames_ScannerCollision(t *testing.T) {
	queries := []Query{
		{Cmd: metadata.CmdOne, MethodName: "GetUser", ConstantName: "getUser", ScannerName: "scanUser"},
		{Cmd: metadata.CmdExec, MethodName: "ScanUser", ConstantName: "scanUser"},
	}
	if err := validateGeneratedNames(&opts.Options{Package: "db"}, queries); err == nil {
		t.Error("expected constant/scanner collision error, got nil")
	}
}

func TestValidateGeneratedNames_QueryTypeCollision(t *testing.T) {
	queries := []Query{
		{Cmd: metadata.CmdBatchOne, MethodName: "BatchGetUsers", ConstantName: "batchGetUsers"},
		{Cmd: metadata.CmdOne, MethodName: "BatchGetUsersQuery", ConstantName: "batchGetUsersQuery"},
	}
	if err := validateGeneratedNames(&opts.Options{Package: "db"}, queries); err == nil {
		t.Error("expected constant/query type collision error, got nil")
	}
}

func TestAssignScanners(t *testing.T) {
	user := &Struct{Name: "User"}
	queries := []Query{
		{Cmd: metadata.CmdMany, MethodName: "ListUsers", SourceName: "b.sql", Ret: QueryValue{Struct: user}},
		{Cmd: metadata.CmdOne, MethodName: "GetUser", SourceName: "a.sql", Ret: QueryValue{Struct: user}},
		{Cmd: metadata.CmdOne, MethodName: "CountUsers", SourceName: "a.sql", Ret: QueryValue{Typ: "int64"}},
		{Cmd: metadata.CmdBatchOne, MethodName: "BatchGetUsers", SourceName: "a.sql", Ret: QueryValue{Struct: user}},
		{Cmd: metadata.CmdMany, MethodName: "ListTags", SourceName: "a.sql", Ret: QueryValue{Name: "tags", Typ: "[]string", SQLDriver: opts.SQLDriverLibPQ}},
		{Cmd: metadata.CmdMany, MethodName: "ListPgxTags", SourceName: "a.sql", Ret: QueryValue{Name: "tags", Typ: "[]string", SQLDriver: opts.SQLDriverPGXV5}},
	}

	scanners := assignScanners(queries)

	if len(scanners) != 2 || scanners[0].Name != "scanUser" || scanners[0].SourceName != "b.sql" || scanners[1].Name != "scanListTags" {
		t.Fatalf("scanners = %+v, want scanUser owned by b.sql and scanListTags", scanners)
	}
	want := []string{"scanUser", "scanUser", "", "", "scanListTags", ""}
	for i, q := range queries {
		if q.ScannerName != want[i] {
			t.Errorf("%s.ScannerName = %q, want %q", q.MethodName, q.ScannerName, want[i])
		}
	}
}
