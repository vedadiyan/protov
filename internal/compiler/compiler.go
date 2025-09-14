package compiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type (
	Resolver struct {
		protocompile.SourceResolver
		Dir string
	}
	Field struct {
		Field string
		Type  reflect.Kind
		Tags  []string
	}
	Message struct {
		Fields []*Field
	}
	File struct {
		Messages []*Message
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
	messages, err := GetMessages(file.Messages())
	if err != nil {
		return nil, err
	}
	out.Messages = messages
	return out, nil
}

func GetMessages(md protoreflect.MessageDescriptors) ([]*Message, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}
	out := make([]*Message, 0)
	for i := range l {
		messageDescriptor := md.Get(i)
		message, err := GetMessage(messageDescriptor.Fields())
		if err != nil {
			return nil, err
		}
		out = append(out, message)
		nestedMessagesLength := messageDescriptor.Messages().Len()
		if nestedMessagesLength != 0 {
			messages, err := GetMessages(messageDescriptor.Messages())
			if err != nil {
				return nil, err
			}
			out = append(out, messages...)
		}
	}
	return out, nil
}

func GetMessage(fd protoreflect.FieldDescriptors) (*Message, error) {
	out := new(Message)
	l := fd.Len()
	if l == 0 {
		return out, nil
	}
	out.Fields = make([]*Field, l)
	for i := range l {
		fieldDescriptor := fd.Get(i)
		field, err := GetField(fieldDescriptor)
		if err != nil {
			return nil, err
		}
		out.Fields[i] = field
	}
	return out, nil
}

func GetField(fieldDescriptor protoreflect.FieldDescriptor) (*Field, error) {
	out := new(Field)
	out.Field = string(fieldDescriptor.FullName().Name())
	out.Type = GetKind(fieldDescriptor)
	out.Tags = GetTags(fieldDescriptor)
	return out, nil
}

func GetKind(fieldDescriptor protoreflect.FieldDescriptor) reflect.Kind {
	if fieldDescriptor.IsMap() {
		return reflect.Map
	}
	flags := reflect.Kind(0)
	if fieldDescriptor.HasOptionalKeyword() {
		flags = reflect.Pointer
	}
	if fieldDescriptor.IsList() {
		flags = reflect.Array
	}
	switch fieldDescriptor.Kind() {
	case protoreflect.BoolKind:
		{
			return reflect.Bool | flags
		}
	case protoreflect.EnumKind:
		{
			panic("")
		}
	case protoreflect.Int32Kind:
		{
			return reflect.Int32 | flags
		}
	case protoreflect.Sint32Kind:
		{
			return reflect.Int32 | flags
		}
	case protoreflect.Uint32Kind:
		{
			return reflect.Uint32 | flags
		}
	case protoreflect.Int64Kind:
		{
			return reflect.Int64 | flags
		}
	case protoreflect.Sint64Kind:
		{
			return reflect.Int64 | flags
		}
	case protoreflect.Uint64Kind:
		{
			return reflect.Uint64 | flags
		}
	case protoreflect.Sfixed32Kind:
		{
			return reflect.Int32 | flags
		}
	case protoreflect.Fixed32Kind:
		{
			return reflect.Int32 | flags
		}
	case protoreflect.FloatKind:
		{
			return reflect.Float32 | flags
		}
	case protoreflect.Sfixed64Kind:
		{
			return reflect.Int64 | flags
		}
	case protoreflect.Fixed64Kind:
		{
			return reflect.Int64 | flags
		}
	case protoreflect.DoubleKind:
		{
			return reflect.Float64 | flags
		}
	case protoreflect.StringKind:
		{
			return reflect.String | flags
		}
	case protoreflect.BytesKind:
		{
			return reflect.Array | reflect.Uint8
		}
	case protoreflect.MessageKind:
		{
			return reflect.Interface | flags
		}
	case protoreflect.GroupKind:
		{
			return reflect.Invalid
		}
	default:
		{
			return reflect.Invalid
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
