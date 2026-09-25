package golang

import (
	"testing"

	"github.com/sqlc-dev/plugin-sdk-go/plugin"
)

func TestPairs_RepeatedParamOnce(t *testing.T) {
	v := QueryValue{
		Name: "arg",
		Struct: &Struct{Fields: []Field{
			{Name: "Value", Type: "string", Column: &plugin.Column{}},
			{Name: "Value", Type: "string", Column: &plugin.Column{}},
		}},
	}

	pairs := v.Pairs()

	if len(pairs) != 1 || pairs[0].Name != "value" {
		t.Fatalf("Pairs() = %+v, want one value pair", pairs)
	}
	if got := v.Params(); got != "value,value" {
		t.Errorf("Params() = %q, want %q", got, "value,value")
	}
}
