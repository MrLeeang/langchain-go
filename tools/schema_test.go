package tools

import (
	"reflect"
	"testing"
)

type searchArgs struct {
	Query string `json:"query" description:"search query" required:"true"`
	Limit int    `json:"limit" description:"max results"`
}

func TestSchemaFromType(t *testing.T) {
	schema := SchemaFromType(reflect.TypeOf(searchArgs{}))
	if schema["type"] != "object" {
		t.Fatalf("type = %v", schema["type"])
	}
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	if _, ok := props["query"]; !ok {
		t.Fatal("missing query")
	}
	req, ok := schema["required"].([]string)
	if !ok || len(req) != 1 || req[0] != "query" {
		t.Fatalf("required = %v", schema["required"])
	}
}

func TestDecodeInput(t *testing.T) {
	in := map[string]any{"query": "hello", "limit": 3}
	v, err := DecodeInput(reflect.TypeOf(searchArgs{}), in)
	if err != nil {
		t.Fatal(err)
	}
	args := v.Interface().(searchArgs)
	if args.Query != "hello" || args.Limit != 3 {
		t.Fatalf("got %+v", args)
	}
}
