package compiler

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/parser"
	"github.com/bufbuild/protocompile/reporter"
	"github.com/vedadiyan/protov/internal/system/install"
	"go.lsp.dev/protocol"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type report struct {
	diagnostics         []protocol.Diagnostic
	syntaxMissing       map[string]bool
	pathToUnusedImports map[string]map[string]bool
}

// Error implements reporter.Handler for *diagnostics.
func (r *report) Error(err reporter.ErrorWithPos) error {
	r.diagnostics = append(r.diagnostics, newDiagnostic(err, false))
	return nil
}

// Warning implements reporter.Handler for *diagnostics.
func (r *report) Warning(err reporter.ErrorWithPos) {
	r.diagnostics = append(r.diagnostics, newDiagnostic(err, true))

	if err.Unwrap() == parser.ErrNoSyntax {
		if r.syntaxMissing == nil {
			r.syntaxMissing = make(map[string]bool)
		}
		r.syntaxMissing[err.GetPosition().Filename] = true
	} else if unusedImport, ok := err.Unwrap().(linker.ErrorUnusedImport); ok {
		if r.pathToUnusedImports == nil {
			r.pathToUnusedImports = make(map[string]map[string]bool)
		}

		path := err.GetPosition().Filename
		unused, ok := r.pathToUnusedImports[path]
		if !ok {
			unused = map[string]bool{}
			r.pathToUnusedImports[path] = unused
		}

		unused[unusedImport.UnusedImport()] = true
	}
}

func newDiagnostic(err reporter.ErrorWithPos, isWarning bool) protocol.Diagnostic {
	pos := protocol.Position{
		Line:      uint32(err.GetPosition().Line - 1),
		Character: uint32(err.GetPosition().Col - 1),
	}

	diagnostic := protocol.Diagnostic{
		// TODO: The compiler currently does not record spans for diagnostics. This is
		// essentially a bug that will result in worse diagnostics until fixed.
		Range:    protocol.Range{Start: pos, End: pos},
		Severity: protocol.DiagnosticSeverityError,
		Message:  err.Unwrap().Error(),
		Source:   "",
	}

	if isWarning {
		diagnostic.Severity = protocol.DiagnosticSeverityWarning
	}

	return diagnostic
}

// Format is the serialization format used to represent the default value.
type Format int
type ByteString string

const (
	_ Format = iota
	// Descriptor uses the serialization format that protoc uses with the
	// google.protobuf.FieldDescriptorProto.default_value field.
	Descriptor
	// GoTag uses the historical serialization format in Go struct field tags.
	GoTag
)

// Ignorables is a set of message names to ignore during compilation.
type Ignorables map[string]struct{}

// Contains checks if a value exists in the ignorables set.
func (i Ignorables) Contains(value string) bool {
	_, exists := i[value]
	return exists
}

// Add adds a value to the ignorables set.
func (i Ignorables) Add(value string) {
	i[value] = struct{}{}
}

// NewIgnorables creates a new Ignorables set.
func NewIgnorables() Ignorables {
	return make(Ignorables)
}

// Resolver resolves source files for protocol buffer compilation.
type Resolver struct {
	protocompile.SourceResolver
	Dir string
}

// NewResolver creates a new Resolver for the given directory.
func NewResolver(dir string) *Resolver {
	r := &Resolver{Dir: dir}
	r.Accessor = r.accessor
	return r
}

func (r *Resolver) accessor(f string) (io.ReadCloser, error) {
	// Normalize path separators
	normalizedPath := strings.ReplaceAll(f, "\\", "/")
	cleanPath := strings.TrimPrefix(normalizedPath, r.Dir)
	cleanPath = strings.TrimPrefix(cleanPath, "/")

	filePath := path.Join(r.Dir, cleanPath)

	data, err := os.ReadFile(filePath)
	if err != nil {
		// Fallback to standard protoc include directory
		protoPath, err := install.ProtoPath()
		if err != nil {
			return nil, err
		}
		fallbackPath := path.Join(protoPath, "include", cleanPath)
		data, err = os.ReadFile(fallbackPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s from %s or %s: %w", f, filePath, fallbackPath, err)
		}
	}

	return io.NopCloser(bytes.NewReader(data)), nil
}

func marshalTags(fd protoreflect.FieldDescriptor) string {
	var buf bytes.Buffer

	buf.WriteString("protobuf:")
	buf.WriteRune('"')
	buf.WriteString(buildTagString(fd, false))
	buf.WriteRune('"')

	if fd.HasJSONName() {
		buf.WriteString(` json:"`)
		buf.WriteString(fd.JSONName())
		buf.WriteRune('"')
	}

	if fd.IsMap() {
		buf.WriteString(` protobuf_key:"`)
		buf.WriteString(buildTagString(fd.MapKey(), true))
		buf.WriteString(`" protobuf_val:"`)
		buf.WriteString(buildTagString(fd.MapValue(), true))
		buf.WriteRune('"')
	}

	return buf.String()
}

func buildTagString(fd protoreflect.FieldDescriptor, skipSyntax bool) string {
	var tags []string

	// Wire type
	switch fd.Kind() {
	case protoreflect.BoolKind, protoreflect.EnumKind,
		protoreflect.Int32Kind, protoreflect.Uint32Kind,
		protoreflect.Int64Kind, protoreflect.Uint64Kind:
		tags = append(tags, "varint")
	case protoreflect.Sint32Kind:
		tags = append(tags, "zigzag32")
	case protoreflect.Sint64Kind:
		tags = append(tags, "zigzag64")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		tags = append(tags, "fixed32")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		tags = append(tags, "fixed64")
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.MessageKind:
		tags = append(tags, "bytes")
	case protoreflect.GroupKind:
		tags = append(tags, "group")
	}

	// Field number
	tags = append(tags, strconv.Itoa(int(fd.Number())))

	// Cardinality
	switch fd.Cardinality() {
	case protoreflect.Optional:
		tags = append(tags, "opt")
	case protoreflect.Required:
		tags = append(tags, "req")
	case protoreflect.Repeated:
		tags = append(tags, "rep")
	}

	if fd.IsPacked() {
		tags = append(tags, "packed")
	}

	// Name (group names need special handling)
	name := string(fd.Name())
	if fd.Kind() == protoreflect.GroupKind {
		name = string(fd.Message().Name())
	}
	tags = append(tags, "name="+name)

	// JSON name
	if jsonName := fd.JSONName(); jsonName != "" && jsonName != name && !fd.IsExtension() {
		tags = append(tags, "json="+jsonName)
	}

	// Proto3 syntax
	if !skipSyntax && fd.Syntax() == protoreflect.Proto3 && !fd.IsExtension() {
		tags = append(tags, "proto3")
	}

	// Enum type
	if fd.Kind() == protoreflect.EnumKind {
		tags = append(tags, "enum="+string(fd.Enum().FullName()))
	}

	// Oneof
	if fd.ContainingOneof() != nil {
		tags = append(tags, "oneof")
	}

	// Default value (must be last)
	if fd.HasDefault() {
		if def, err := marshalDefaultValue(fd.Default(), fd.DefaultEnumValue(), fd.Kind(), GoTag); err == nil {
			tags = append(tags, "def="+def)
		}
	}

	return strings.Join(tags, ",")
}

func marshalDefaultValue(v protoreflect.Value, ev protoreflect.EnumValueDescriptor, k protoreflect.Kind, f Format) (string, error) {
	switch k {
	case protoreflect.BoolKind:
		if f == GoTag {
			if v.Bool() {
				return "1", nil
			}
			return "0", nil
		}
		return strconv.FormatBool(v.Bool()), nil

	case protoreflect.EnumKind:
		if f == GoTag {
			return strconv.FormatInt(int64(v.Enum()), 10), nil
		}
		return string(ev.Name()), nil

	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(v.Int(), 10), nil

	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return strconv.FormatUint(v.Uint(), 10), nil

	case protoreflect.FloatKind, protoreflect.DoubleKind:
		f := v.Float()
		switch {
		case math.IsInf(f, -1):
			return "-inf", nil
		case math.IsInf(f, +1):
			return "inf", nil
		case math.IsNaN(f):
			return "nan", nil
		default:
			bitSize := 64
			if k == protoreflect.FloatKind {
				bitSize = 32
			}
			return strconv.FormatFloat(f, 'g', -1, bitSize), nil
		}

	case protoreflect.StringKind:
		return v.String(), nil

	case protoreflect.BytesKind:
		return marshalBytes(v.Bytes())
	}

	return "", fmt.Errorf("unsupported kind for default value: %v", k)
}

func marshalBytes(b []byte) (string, error) {
	var buf bytes.Buffer

	for _, c := range b {
		switch c {
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '"':
			buf.WriteString(`\"`)
		case '\'':
			buf.WriteString(`\'`)
		case '\\':
			buf.WriteString(`\\`)
		default:
			if c >= 0x20 && c <= 0x7e { // printable ASCII
				buf.WriteByte(c)
			} else {
				fmt.Fprintf(&buf, `\%03o`, c)
			}
		}
	}

	return buf.String(), nil
}
