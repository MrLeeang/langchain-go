package tools

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// SchemaFromType builds a JSON Schema object for OpenAI function parameters from a Go type.
func SchemaFromType(t reflect.Type) map[string]any {
	if t == nil {
		return defaultObjectSchema()
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	schema := schemaForType(t)
	if schema == nil {
		return defaultObjectSchema()
	}
	if schema["type"] != "object" {
		return map[string]any{
			"type":       "object",
			"properties": map[string]any{"value": schema},
			"required":   []string{"value"},
		}
	}
	return schema
}

func defaultObjectSchema() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func schemaForType(t reflect.Type) map[string]any {
	if t == nil {
		return nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.Struct:
		return schemaForStruct(t)
	case reflect.Map:
		if t.Key().Kind() == reflect.String {
			return map[string]any{
				"type":                 "object",
				"additionalProperties": schemaForType(t.Elem()),
			}
		}
		return map[string]any{"type": "object"}
	case reflect.Slice, reflect.Array:
		return map[string]any{
			"type":  "array",
			"items": schemaForType(t.Elem()),
		}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Interface:
		return defaultObjectSchema()
	default:
		return map[string]any{"type": "string"}
	}
}

func schemaForStruct(t reflect.Type) map[string]any {
	props := make(map[string]any)
	var required []string

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Anonymous {
			embed := schemaForStruct(f.Type)
			if embed != nil {
				if ep, ok := embed["properties"].(map[string]any); ok {
					for k, v := range ep {
						props[k] = v
					}
				}
				if er, ok := embed["required"].([]string); ok {
					required = append(required, er...)
				}
			}
			continue
		}

		jsonName, omit, req := jsonFieldMeta(f)
		if omit {
			continue
		}

		fieldSchema := schemaForType(f.Type)
		if fieldSchema == nil {
			continue
		}
		if desc := fieldDescription(f); desc != "" {
			fieldSchema["description"] = desc
		}
		if enum := parseEnumTag(f.Tag.Get("enum")); len(enum) > 0 {
			fieldSchema["enum"] = enum
		}
		props[jsonName] = fieldSchema
		if req {
			required = append(required, jsonName)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func jsonFieldMeta(f reflect.StructField) (name string, omit bool, required bool) {
	tag := f.Tag.Get("json")
	if tag == "-" {
		return "", true, false
	}
	name = f.Name
	if tag != "" {
		parts := strings.Split(tag, ",")
		name = parts[0]
		if name == "" {
			name = f.Name
		}
		for _, p := range parts[1:] {
			switch p {
			case "omitempty":
				// not required
			default:
				// ignore unknown json flags
			}
		}
	}
	if f.Tag.Get("required") == "true" {
		required = true
	}
	if strings.Contains(f.Tag.Get("jsonschema"), "required") {
		required = true
	}
	return name, false, required
}

func fieldDescription(f reflect.StructField) string {
	if d := f.Tag.Get("description"); d != "" {
		return d
	}
	if d := f.Tag.Get("desc"); d != "" {
		return d
	}
	// jsonschema:"description=..."
	if tag := f.Tag.Get("jsonschema"); tag != "" {
		for _, part := range strings.Split(tag, ",") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "description=") {
				return strings.TrimPrefix(part, "description=")
			}
		}
	}
	return ""
}

func parseEnumTag(tag string) []any {
	if tag == "" {
		return nil
	}
	parts := strings.Split(tag, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// DecodeInput unmarshals a tool call argument map into a value of type t (pointer or struct).
func DecodeInput(t reflect.Type, input any) (reflect.Value, error) {
	if t == nil {
		return reflect.Value{}, nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	val := reflect.New(t)
	if input == nil {
		return val.Elem(), nil
	}

	switch v := input.(type) {
	case map[string]any:
		return decodeFromMap(t, val, v)
	default:
		return reflect.Value{}, fmt.Errorf("unsupported tool input type %T", input)
	}
}

func decodeFromMap(t reflect.Type, dest reflect.Value, m map[string]any) (reflect.Value, error) {
	if t.Kind() == reflect.Map && t.Key().Kind() == reflect.String {
		mv := reflect.MakeMap(t)
		elemType := t.Elem()
		for k, v := range m {
			ev, err := assignJSONValue(elemType, v)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("field %q: %w", k, err)
			}
			mv.SetMapIndex(reflect.ValueOf(k), ev)
		}
		return mv, nil
	}

	// struct: use JSON round-trip for nested types
	b, err := json.Marshal(m)
	if err != nil {
		return reflect.Value{}, err
	}
	if err := json.Unmarshal(b, dest.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("decode arguments: %w", err)
	}
	return dest.Elem(), nil
}

func assignJSONValue(t reflect.Type, v any) (reflect.Value, error) {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		val := reflect.New(t)
		b, err := json.Marshal(v)
		if err != nil {
			return reflect.Value{}, err
		}
		if err := json.Unmarshal(b, val.Interface()); err != nil {
			return reflect.Value{}, err
		}
		return val.Elem(), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return reflect.Value{}, err
	}
	val := reflect.New(t)
	if err := json.Unmarshal(b, val.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return val.Elem(), nil
}
