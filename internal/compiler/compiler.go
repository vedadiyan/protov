package compiler

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Format is the serialization format used to represent the default value.
type Format int

const (
	_ Format = iota

	// Descriptor uses the serialization format that protoc uses with the
	// google.protobuf.FieldDescriptorProto.default_value field.
	Descriptor

	// GoTag uses the historical serialization format in Go struct field tags.
	GoTag
)

type (
	Ignorables []string
	Resolver   struct {
		protocompile.SourceResolver
		Dir string
	}
	Field struct {
		Field         string
		Type          string
		Tags          []string
		MarshalledTag string
	}
	EnumValue struct {
		Name   string
		Number int
	}
	Enum struct {
		Name   string
		Values []*EnumValue
	}
	Message struct {
		Name       string
		Fields     []*Field
		Ignorables Ignorables
		Descriptor string
	}
	File struct {
		Messages []*Message
		Enums    []*Enum
	}
	AST struct {
		Files []*File
	}
)

func (r *Resolver) accessor(f string) (io.ReadCloser, error) {
	f = strings.ReplaceAll(strings.ReplaceAll(f, "\\", "/"), r.Dir, "")
	file := path.Join(r.Dir, f)
	data, err := os.ReadFile(file)
	if err != nil {
		return func(f string) (io.ReadCloser, error) {
			data, err := os.ReadFile(f)
			if err != nil {
				return nil, err
			}
			return io.NopCloser(bytes.NewBuffer(data)), nil
		}(path.Join("C:\\protoc\\include", f))
	}
	return io.NopCloser(bytes.NewBuffer(data)), nil
}

func NewResolver(dir string) *Resolver {
	r := new(Resolver)
	r.Dir = dir
	r.Accessor = r.accessor
	return r
}

func Compile(file string) (*AST, error) {
	dir := fmt.Sprintf("%s/", path.Dir(strings.ReplaceAll(file, "\\", "/")))
	var report report
	var symbols linker.Symbols
	compiler := protocompile.Compiler{
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations,
		Resolver:       NewResolver(dir),
		Symbols:        &symbols,
		Reporter:       &report,
	}
	linker, err := compiler.Compile(context.TODO(), file)
	if err != nil {
		return nil, err
	}
	out := new(AST)
	out.Files = make([]*File, len(linker))
	for i, linker := range linker {
		ast, err := GetFile(linker)
		if err != nil {
			return nil, err
		}
		out.Files[i] = ast
	}

	return out, nil
}

func GetFile(file linker.File) (*File, error) {
	out := new(File)
	messages, err := GetMessages(file.Messages(), nil)
	if err != nil {
		return nil, err
	}
	out.Messages = messages
	enums, err := GetEnums(file.Enums())
	if err != nil {
		return nil, err
	}
	out.Enums = enums
	return out, nil
}

func GetMessages(md protoreflect.MessageDescriptors, ignoreList Ignorables) ([]*Message, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}
	out := make([]*Message, 0)

	for i := range l {
		messageDescriptor := md.Get(i)
		name := messageDescriptor.Name()
		if ignoreList.Contains(string(name)) {
			continue
		}
		message, err := GetMessage(name, messageDescriptor.Fields())
		if err != nil {
			return nil, err
		}
		xxx := protoutil.ProtoFromMessageDescriptor(messageDescriptor)
		zzz, err := protojson.Marshal(xxx)
		if err != nil {
			return nil, err
		}
		message.Descriptor = base64.StdEncoding.EncodeToString(zzz)

		out = append(out, message)
		nestedMessagesLength := messageDescriptor.Messages().Len()
		if nestedMessagesLength != 0 {
			messages, err := GetMessages(messageDescriptor.Messages(), message.Ignorables)
			if err != nil {
				return nil, err
			}
			out = append(out, messages...)
		}
	}
	return out, nil
}

func GetMessage(name protoreflect.Name, fd protoreflect.FieldDescriptors) (*Message, error) {
	out := new(Message)
	l := fd.Len()
	if l == 0 {
		return out, nil
	}
	out.Name = string(name)
	out.Fields = make([]*Field, l)
	out.Ignorables = make([]string, 0)
	for i := range l {
		fieldDescriptor := fd.Get(i)
		field, err := GetField(fieldDescriptor)
		if err != nil {
			return nil, err
		}
		out.Fields[i] = field
		if ok, value := CanBeIgnored(fieldDescriptor); ok {
			out.Ignorables = append(out.Ignorables, value)
		}
	}
	return out, nil
}

func GetField(fieldDescriptor protoreflect.FieldDescriptor) (*Field, error) {
	out := new(Field)
	out.Field = string(fieldDescriptor.FullName().Name())
	out.Type = GetKind(fieldDescriptor)
	out.Tags = GetTags(fieldDescriptor)
	out.MarshalledTag = MarshallTags(fieldDescriptor)
	return out, nil
}

func GetEnums(md protoreflect.EnumDescriptors) ([]*Enum, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}
	out := make([]*Enum, 0)
	for i := range l {
		enumDescriptor := md.Get(i)
		message, err := GetEnum(enumDescriptor.Name(), enumDescriptor.Values())
		if err != nil {
			return nil, err
		}
		out = append(out, message)
	}
	return out, nil
}

func GetEnum(name protoreflect.Name, ed protoreflect.EnumValueDescriptors) (*Enum, error) {
	out := new(Enum)
	l := ed.Len()
	if l == 0 {
		return nil, nil
	}
	out.Name = string(name)
	enumValues := make([]*EnumValue, l)
	for i := range l {
		enumValueDescriptor := ed.Get(i)
		enumValue := new(EnumValue)
		enumValue.Name = string(enumValueDescriptor.Name())
		enumValue.Number = int(enumValueDescriptor.Number())
		enumValues = append(enumValues, enumValue)
	}
	out.Values = enumValues
	return out, nil
}

func CanBeIgnored(fieldDescriptor protoreflect.FieldDescriptor) (bool, string) {
	if fieldDescriptor.IsMap() {
		return true, string(fieldDescriptor.Message().Name())
	}
	return false, ""
}

func GetKind(fieldDescriptor protoreflect.FieldDescriptor) string {
	if fieldDescriptor.IsMap() {
		return fmt.Sprintf("map[%s]%s", GetKind(fieldDescriptor.MapKey()), GetKind(fieldDescriptor.MapValue()))
	}
	flags := ""
	if fieldDescriptor.HasOptionalKeyword() {
		flags = "*"
	}
	if fieldDescriptor.IsList() {
		flags = "[]"
	}
	switch fieldDescriptor.Kind() {
	case protoreflect.BoolKind:
		{
			return fmt.Sprintf("%s%s", flags, "bool")
		}
	case protoreflect.EnumKind:
		{
			return fmt.Sprintf("%s%s", flags, fieldDescriptor.Enum().Name())
		}
	case protoreflect.Int32Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int")
		}
	case protoreflect.Sint32Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int")
		}
	case protoreflect.Uint32Kind:
		{
			return fmt.Sprintf("%s%s", flags, "uint")
		}
	case protoreflect.Int64Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int64")
		}
	case protoreflect.Sint64Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int64")
		}
	case protoreflect.Uint64Kind:
		{
			return fmt.Sprintf("%s%s", flags, "uint64")
		}
	case protoreflect.Sfixed32Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int")
		}
	case protoreflect.Fixed32Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int")
		}
	case protoreflect.FloatKind:
		{
			return fmt.Sprintf("%s%s", flags, "float32")
		}
	case protoreflect.Sfixed64Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int64")
		}
	case protoreflect.Fixed64Kind:
		{
			return fmt.Sprintf("%s%s", flags, "int64")
		}
	case protoreflect.DoubleKind:
		{
			return fmt.Sprintf("%s%s", flags, "float64")
		}
	case protoreflect.StringKind:
		{
			return fmt.Sprintf("%s%s", flags, "string")
		}
	case protoreflect.BytesKind:
		{
			return "[]byte"
		}
	case protoreflect.MessageKind:
		{
			return fmt.Sprintf("%s%s", flags, fieldDescriptor.Message().Name())
		}
	case protoreflect.GroupKind:
		{
			return "interfacce {}"
		}
	default:
		{
			return ""
		}
	}
}

func GetTags(fieldDescriptor protoreflect.FieldDescriptor) []string {
	out := make([]string, 0)
	if fieldDescriptor.HasJSONName() {
		out = append(out, fmt.Sprintf(`json:"%s"`, fieldDescriptor.JSONName()))
	}
	opts, ok := fieldDescriptor.Options().(*descriptorpb.FieldOptions)
	if !ok {
		return out
	}
	proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
		out = append(out, fmt.Sprintf("%s.%s:\"%v\"", et.TypeDescriptor().Parent().FullName().Name(), et.TypeDescriptor().FullName().Name(), a))
		return true
	})
	return out
}

func (i Ignorables) Contains(value string) bool {
	for _, v := range i {
		if v == value {
			return true
		}
	}
	return false
}

func MarshallTags(fd protoreflect.FieldDescriptor) string {
	var buffer bytes.Buffer
	buffer.WriteString("protobuf:")
	buffer.WriteRune('"')
	buffer.WriteString(marshallTags(fd, false))
	buffer.WriteRune('"')
	if fd.HasJSONName() {
		buffer.WriteString(" ")
		buffer.WriteString("json:")
		buffer.WriteRune('"')
		buffer.WriteString(fd.JSONName())
		buffer.WriteRune('"')
	}
	if fd.IsMap() {
		buffer.WriteString(" ")
		buffer.WriteString("protobuf_key:")
		buffer.WriteRune('"')
		buffer.WriteString(marshallTags(fd.MapKey(), true))
		buffer.WriteRune('"')
		buffer.WriteString(" ")
		buffer.WriteString("protobuf_val:")
		buffer.WriteRune('"')
		buffer.WriteString(marshallTags(fd.MapValue(), true))
		buffer.WriteRune('"')
	}
	return buffer.String()
}

func marshallTags(fd protoreflect.FieldDescriptor, skipSyntax bool) string {
	var tag []string
	switch fd.Kind() {
	case protoreflect.BoolKind, protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Uint32Kind, protoreflect.Int64Kind, protoreflect.Uint64Kind:
		tag = append(tag, "varint")
	case protoreflect.Sint32Kind:
		tag = append(tag, "zigzag32")
	case protoreflect.Sint64Kind:
		tag = append(tag, "zigzag64")
	case protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind, protoreflect.FloatKind:
		tag = append(tag, "fixed32")
	case protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind, protoreflect.DoubleKind:
		tag = append(tag, "fixed64")
	case protoreflect.StringKind, protoreflect.BytesKind, protoreflect.MessageKind:
		tag = append(tag, "bytes")
	case protoreflect.GroupKind:
		tag = append(tag, "group")
	}
	tag = append(tag, strconv.Itoa(int(fd.Number())))
	switch fd.Cardinality() {
	case protoreflect.Optional:
		tag = append(tag, "opt")
	case protoreflect.Required:
		tag = append(tag, "req")
	case protoreflect.Repeated:
		tag = append(tag, "rep")
	}
	if fd.IsPacked() {
		tag = append(tag, "packed")
	}
	name := string(fd.Name())
	if fd.Kind() == protoreflect.GroupKind {
		// The name of the FieldDescriptor for a group field is
		// lowercased. To find the original capitalization, we
		// look in the field's MessageType.
		name = string(fd.Message().Name())
	}
	tag = append(tag, "name="+name)
	if jsonName := fd.JSONName(); jsonName != "" && jsonName != name && !fd.IsExtension() {
		// NOTE: The jsonName != name condition is suspect, but it preserve
		// the exact same semantics from the previous generator.
		tag = append(tag, "json="+jsonName)
	}
	// The previous implementation does not tag extension fields as proto3,
	// even when the field is defined in a proto3 file. Match that behavior
	// for consistency.
	if !skipSyntax && fd.Syntax() == protoreflect.Proto3 && !fd.IsExtension() {
		tag = append(tag, "proto3")
	}
	if fd.Kind() == protoreflect.EnumKind {
		tag = append(tag, "enum="+string(fd.Enum().FullName()))
	}
	if fd.ContainingOneof() != nil {
		tag = append(tag, "oneof")
	}
	// This must appear last in the tag, since commas in strings aren't escaped.
	if fd.HasDefault() {
		def, _ := marshallDefaultValue(fd.Default(), fd.DefaultEnumValue(), fd.Kind(), GoTag)
		tag = append(tag, "def="+def)
	}
	return strings.Join(tag, ",")
}

func marshallDefaultValue(v protoreflect.Value, ev protoreflect.EnumValueDescriptor, k protoreflect.Kind, f Format) (string, error) {
	switch k {
	case protoreflect.BoolKind:
		if f == GoTag {
			if v.Bool() {
				return "1", nil
			} else {
				return "0", nil
			}
		} else {
			if v.Bool() {
				return "true", nil
			} else {
				return "false", nil
			}
		}
	case protoreflect.EnumKind:
		if f == GoTag {
			return strconv.FormatInt(int64(v.Enum()), 10), nil
		} else {
			return string(ev.Name()), nil
		}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return strconv.FormatInt(v.Int(), 10), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
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
			if k == protoreflect.FloatKind {
				return strconv.FormatFloat(f, 'g', -1, 32), nil
			} else {
				return strconv.FormatFloat(f, 'g', -1, 64), nil
			}
		}
	case protoreflect.StringKind:
		// String values are serialized as is without any escaping.
		return v.String(), nil
	case protoreflect.BytesKind:
		if s, ok := marshalBytes(v.Bytes()); ok {
			return s, nil
		}
	}
	return "", fmt.Errorf("could not format value for %v: %v", k, v)
}

func marshalBytes(b []byte) (string, bool) {
	var s []byte
	for _, c := range b {
		switch c {
		case '\n':
			s = append(s, `\n`...)
		case '\r':
			s = append(s, `\r`...)
		case '\t':
			s = append(s, `\t`...)
		case '"':
			s = append(s, `\"`...)
		case '\'':
			s = append(s, `\'`...)
		case '\\':
			s = append(s, `\\`...)
		default:
			if printableASCII := c >= 0x20 && c <= 0x7e; printableASCII {
				s = append(s, c)
			} else {
				s = append(s, fmt.Sprintf(`\%03o`, c)...)
			}
		}
	}
	return string(s), true
}
