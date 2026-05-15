package golang

import (
	"testing"

	"github.com/sqlc-dev/sqlc-gen-go/internal/opts"
)

// In split-package mode an override may point at a type defined inside the
// models package itself. The models file must NOT import its own package
// (would be a cycle), but the queries file still needs the import.
func TestBuildImports_SkipsSelfImportOverrideInModelsFile(t *testing.T) {
	const modelsPath = "example.com/proj/internal/models"

	options := &opts.Options{
		ModelsPackageImportPath: modelsPath,
		OutputModelsPackage:     "models",
		Overrides: []opts.Override{
			{
				ShimOverride: &opts.ShimOverride{
					DbType: "uuid",
					GoType: &opts.ShimGoType{
						ImportPath: modelsPath,
						Package:    "models",
						TypeName:   "models.UUID",
					},
				},
			},
		},
	}
	uses := func(name string) bool { return name == "models.UUID" }

	t.Run("OutputFileModel skips self-import", func(t *testing.T) {
		_, pkg := buildImports(options, nil, OutputFileModel, uses)
		for spec := range pkg {
			if spec.Path == modelsPath {
				t.Fatalf("models file imported its own package %q (cycle): %+v", modelsPath, pkg)
			}
		}
	})

	t.Run("OutputFileQuery still emits the import", func(t *testing.T) {
		_, pkg := buildImports(options, nil, OutputFileQuery, uses)
		var found bool
		for spec := range pkg {
			if spec.Path == modelsPath {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("queries file did not import override package %q: %+v", modelsPath, pkg)
		}
	})
}
