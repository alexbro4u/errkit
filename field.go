package errkit

import "log/slog"

// FieldType represents the type of a metadata field value.
type FieldType int

const (
	FieldTypeString FieldType = iota
	FieldTypeInt
	FieldTypeInt64
	FieldTypeBool
	FieldTypeFloat64
	FieldTypeAny
)

// Field is a structured key-value pair attached to an error.
type Field struct {
	Key    string
	Type   FieldType
	Str    string
	Int    int64
	Float  float64
	Bool   bool
	AnyVal any
}

// String creates a string metadata field.
func String(key, val string) Field {
	return Field{Key: key, Type: FieldTypeString, Str: val}
}

// Int creates an int metadata field.
func Int(key string, val int) Field {
	return Field{Key: key, Type: FieldTypeInt, Int: int64(val)}
}

// Int64 creates an int64 metadata field.
func Int64(key string, val int64) Field {
	return Field{Key: key, Type: FieldTypeInt64, Int: val}
}

// Bool creates a bool metadata field.
func Bool(key string, val bool) Field {
	return Field{Key: key, Type: FieldTypeBool, Bool: val}
}

// Float64 creates a float64 metadata field.
func Float64(key string, val float64) Field {
	return Field{Key: key, Type: FieldTypeFloat64, Float: val}
}

// Any creates an arbitrary metadata field.
func Any(key string, val any) Field {
	return Field{Key: key, Type: FieldTypeAny, AnyVal: val}
}

// Value returns the underlying value of the field.
func (f Field) Value() any {
	switch f.Type {
	case FieldTypeString:
		return f.Str
	case FieldTypeInt, FieldTypeInt64:
		return f.Int
	case FieldTypeBool:
		return f.Bool
	case FieldTypeFloat64:
		return f.Float
	case FieldTypeAny:
		return f.AnyVal
	default:
		return nil
	}
}

// SlogAttr converts the field to a slog.Attr.
func (f Field) SlogAttr() slog.Attr {
	switch f.Type {
	case FieldTypeString:
		return slog.String(f.Key, f.Str)
	case FieldTypeInt, FieldTypeInt64:
		return slog.Int64(f.Key, f.Int)
	case FieldTypeBool:
		return slog.Bool(f.Key, f.Bool)
	case FieldTypeFloat64:
		return slog.Float64(f.Key, f.Float)
	case FieldTypeAny:
		return slog.Any(f.Key, f.AnyVal)
	default:
		return slog.Any(f.Key, nil)
	}
}
