package compiler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"maps"
	"math"
	"os"
	"path"
	"reflect"
	"strconv"
	"strings"
	"text/template"
	"unicode"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/google/uuid"
	"github.com/vedadiyan/protov/internal/system/protoc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

var (
	//go:embed templates/decode.go.tmpl
	_decodeTemplate string
	//go:embed templates/decodemap.go.tmpl
	_decodeMapTemplate string
	//go:embed templates/decoderepeated.go.tmpl
	_decodeRepeatedTemplate string
	//go:embed templates/encode.go.tmpl
	_encodeTemplate string
	//go:embed templates/encodemap.go.tmpl
	_encodeMapTemplate string
	//go:embed templates/encoderepeated.go.tmpl
	_encodeRepeatedTemplate string
	//go:embed templates/enum.go.tmpl
	_enumTemplate string
	//go:embed templates/iszero.go.tmpl
	_isZeroTemplate string
	//go:embed templates/main.go.tmpl
	_mainTemplate string
	//go:embed templates/message.go.tmpl
	_messageTemplate string
	//go:embed templates/service.go.tmpl
	_serviceTemplate string
)

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
		protoPath, err := protoc.ProtoPath()
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

// Field represents a protocol buffer field.
type Field struct {
	Name          string
	Type          string
	BaseType      string
	KeyBaseType   string
	IndexBaseType string
	Options       map[string]any
	Optional      bool
	MarshalledTag string
	Kind          reflect.Kind
	Index         reflect.Kind
	Key           reflect.Kind
	FieldNum      int
}

// EnumValue represents a single enum value.
type EnumValue struct {
	Name   string
	Number int
}

// Enum represents a protocol buffer enum.
type Enum struct {
	Name    string
	Values  []*EnumValue
	Options map[string]any
	File    *File
}

// Message represents a protocol buffer message.
type Message struct {
	Name       string
	Fields     []*Field
	Ignorables Ignorables
	Options    map[string]any
	Descriptor string
	TypeName   string
	File       *File
}

// Message represents a protocol buffer service.
type Service struct {
	Name           string
	Options        map[string]any
	Descriptor     string
	Rpcs           []*Rpc
	RpcOptions     map[string]any
	CodeGeneration []string
	File           *File
}

// Message represents a protocol buffer rpc.
type Rpc struct {
	Name        string
	Options     map[string]any
	Descriptor  string
	Input       string
	Output      string
	ServiceName string
}

// File represents a compiled protocol buffer file.
type File struct {
	Dir         string
	PackageName string
	FilePath    string
	Source      string
	Options     map[string]any
	Messages    []*Message
	Services    []*Service
	Enums       []*Enum
	Comments    map[string]string
	FileName    string
}

// AST represents the complete abstract syntax tree of compiled files.
type AST struct {
	Files []*File
}

func parseTemplates(t *template.Template, templates ...string) (*template.Template, error) {
	if len(templates) == 0 {
		return nil, fmt.Errorf("template: no files named in call to ParseFiles")
	}
	for _, i := range templates {
		name := uuid.New().String()
		var tmpl *template.Template
		if t == nil {
			t = template.New(name)
		}
		tmpl = t.New(name)

		_, err := tmpl.Parse(i)
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

func Compile(file *File) ([]byte, error) {
	allTemplates := []string{
		_decodeTemplate,
		_decodeMapTemplate,
		_decodeRepeatedTemplate,
		_encodeTemplate,
		_encodeMapTemplate,
		_encodeRepeatedTemplate,
		_enumTemplate,
		_isZeroTemplate,
		_mainTemplate,
		_messageTemplate,
		_serviceTemplate,
	}
	template := template.New("temp")
	templates, err := parseTemplates(template, allTemplates...)
	if err != nil {
		return nil, err
	}
	out := bytes.NewBuffer([]byte{})
	if err := templates.ExecuteTemplate(out, "Main", file); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Parse compiles a protocol buffer file and returns its AST.
func Parse(file string) (*AST, error) {
	normalizedFile := strings.ReplaceAll(file, "\\", "/")
	dir := path.Dir(normalizedFile) + "/"

	var report report
	var symbols linker.Symbols

	compiler := protocompile.Compiler{
		SourceInfoMode: protocompile.SourceInfoExtraOptionLocations | protocompile.SourceInfoExtraComments,
		Resolver:       NewResolver(dir),
		Symbols:        &symbols,
		Reporter:       &report,
	}

	linkedFiles, err := compiler.Compile(context.TODO(), file)
	if err != nil {
		return nil, fmt.Errorf("compilation failed: %w", err)
	}

	ast := &AST{
		Files: make([]*File, len(linkedFiles)),
	}

	for i, linkedFile := range linkedFiles {
		fileAST, err := GetFile(dir, normalizedFile, linkedFile)
		if err != nil {
			return nil, fmt.Errorf("failed to process file %d: %w", i, err)
		}
		ast.Files[i] = fileAST
	}

	return ast, nil
}

// GetFile extracts file information from a linked file.
func GetFile(dir string, filePath string, file linker.File) (*File, error) {
	out := &File{
		Options: make(map[string]any),
	}
	out.Dir = dir
	_, out.Source = path.Split(filePath)
	out.FileName = strings.ReplaceAll(strings.ToLower(out.Source), ".proto", "")
	out.Comments = make(map[string]string)
	protodesc := protodesc.ToFileDescriptorProto(file)
	for _, i := range protodesc.SourceCodeInfo.Location {
		if i.LeadingComments != nil {
			path := file.SourceLocations().ByPath(i.Path).Path.String()
			value := strings.TrimRight(*i.LeadingComments, "\r\n")
			value = strings.TrimRight(value, " ")
			value = strings.TrimLeft(value, " ")
			out.Comments[path] = value
		}
	}

	if opts, ok := file.Options().(*descriptorpb.FileOptions); ok {
		out.FilePath = opts.GetGoPackage()
		_, out.PackageName = path.Split(out.FilePath)
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			out.Options[key] = out.getInnerOptions("", a)
			return true
		})
	}

	messages, err := out.GetMessages(file.Messages(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	out.Messages = messages

	services, err := out.GetServices(file.Services())
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	out.Services = services

	enums, err := out.GetEnums(file.Enums())
	if err != nil {
		return nil, fmt.Errorf("failed to get enums: %w", err)
	}
	out.Enums = enums

	return out, nil
}

// GetMessages extracts message information from message descriptors.
func (file *File) GetMessages(md protoreflect.MessageDescriptors, ignoreList Ignorables) ([]*Message, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}

	out := make([]*Message, 0, l)

	for i := 0; i < l; i++ {
		messageDescriptor := md.Get(i)
		name := messageDescriptor.Name()

		if ignoreList.Contains(string(name)) {
			continue
		}

		message, err := file.GetMessage(messageDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to get message %s: %w", name, err)
		}

		// Encode descriptor as base64 JSON
		protoDescriptor := protoutil.ProtoFromMessageDescriptor(messageDescriptor)
		jsonData, err := protojson.Marshal(protoDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal descriptor for %s: %w", name, err)
		}
		message.Descriptor = base64.StdEncoding.EncodeToString(jsonData)

		out = append(out, message)

		// Process nested messages recursively
		if nestedMsgLen := messageDescriptor.Messages().Len(); nestedMsgLen != 0 {
			nestedMessages, err := file.GetMessages(messageDescriptor.Messages(), message.Ignorables)
			if err != nil {
				return nil, fmt.Errorf("failed to get nested messages in %s: %w", name, err)
			}
			out = append(out, nestedMessages...)
		}
	}

	return out, nil
}

// GetMessage creates a Message from field descriptors.
func (file *File) GetMessage(message protoreflect.MessageDescriptor) (*Message, error) {
	name := message.Name()
	fullName := message.FullName()

	fields := message.Fields()

	l := fields.Len()

	out := &Message{
		Name:       string(name),
		TypeName:   string(fullName),
		Fields:     make([]*Field, 0, l),
		Ignorables: NewIgnorables(),
		File:       file,
	}

	if opts, ok := message.Options().(*descriptorpb.MessageOptions); ok {
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			key = toGoName(key)
			out.Options[key] = file.getInnerOptions("", a)
			return true
		})
	}

	if l == 0 {
		return out, nil
	}

	for i := 0; i < l; i++ {
		fieldDescriptor := fields.Get(i)

		field, err := file.GetField(fieldDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to get field %s: %w", fieldDescriptor.Name(), err)
		}
		out.Fields = append(out.Fields, field)

		if ok, value := canBeIgnored(fieldDescriptor); ok {
			out.Ignorables.Add(value)
		}
	}

	return out, nil
}

// GetField creates a Field from a field descriptor.
func (file *File) GetField(fd protoreflect.FieldDescriptor) (*Field, error) {
	fieldType := getKind(fd)

	out := &Field{
		Name:          toGoName(string(fd.Name())),
		Type:          fieldType,
		BaseType:      cleanType(fieldType),
		FieldNum:      int(fd.Number()),
		Optional:      fd.HasOptionalKeyword(),
		MarshalledTag: marshalTags(fd),
	}

	if opts, ok := fd.Options().(*descriptorpb.FieldOptions); ok {
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			out.Options[key] = a
			return true
		})
	}

	// Determine field kind and related types
	switch {
	case fd.IsMap():
		out.Kind = reflect.Map
		out.Index = getReflectedKind(fd.MapKey().Kind())
		out.Key = getReflectedKind(fd.MapValue().Kind())
		out.KeyBaseType = cleanType(getKind(fd.MapKey()))
		out.IndexBaseType = cleanType(getKind(fd.MapValue()))

	case fd.IsList():
		out.Kind = reflect.Array
		out.Index = getReflectedKind(fd.Kind())
		out.IndexBaseType = cleanType(fieldType)

	default:
		out.Kind = getReflectedKind(fd.Kind())
	}

	return out, nil
}

// GetEnums extracts enum information from enum descriptors.
func (file *File) GetEnums(md protoreflect.EnumDescriptors) ([]*Enum, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}

	out := make([]*Enum, 0, l)

	for i := 0; i < l; i++ {
		enumDescriptor := md.Get(i)
		enum, err := file.getEnum(enumDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to get enum %s: %w", enumDescriptor.Name(), err)
		}
		out = append(out, enum)
	}

	return out, nil
}

// getEnum creates an Enum from enum value descriptors.
func (file *File) getEnum(enum protoreflect.EnumDescriptor) (*Enum, error) {
	name := enum.Name()
	ed := enum.Values()
	l := ed.Len()
	if l == 0 {
		return nil, nil
	}

	out := &Enum{
		Name:   string(name),
		Values: make([]*EnumValue, 0, l),
		File:   file,
	}

	if opts, ok := enum.Options().(*descriptorpb.EnumOptions); ok {
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			out.Options[key] = file.getInnerOptions("", a)
			return true
		})
	}

	for i := 0; i < l; i++ {
		evd := ed.Get(i)
		out.Values = append(out.Values, &EnumValue{
			Name:   string(evd.Name()),
			Number: int(evd.Number()),
		})
	}

	return out, nil
}

// GetServices extracts service information from service descriptors.
func (file *File) GetServices(md protoreflect.ServiceDescriptors) ([]*Service, error) {
	l := md.Len()
	if l == 0 {
		return nil, nil
	}

	out := make([]*Service, 0, l)

	for i := 0; i < l; i++ {
		serviceDescriptor := md.Get(i)
		name := serviceDescriptor.Name()

		service, err := file.GetService(i, serviceDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to get message %s: %w", name, err)
		}

		// Encode descriptor as base64 JSON
		protoDescriptor := protoutil.ProtoFromServiceDescriptor(serviceDescriptor)
		jsonData, err := protojson.Marshal(protoDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal descriptor for %s: %w", name, err)
		}
		service.Descriptor = base64.StdEncoding.EncodeToString(jsonData)

		out = append(out, service)
	}

	return out, nil
}

// GetService creates a Service from service descriptors.
func (file *File) GetService(n int, service protoreflect.ServiceDescriptor) (*Service, error) {
	methods := service.Methods()

	comments := make([]string, 0)
	if value, ok := file.Comments[fmt.Sprintf(".service[%d]", n)]; ok {
		values := strings.Split(value, "\r\n")
		for _, i := range values {
			str := strings.TrimLeft(i, " ")
			str = strings.TrimRight(str, " ")
			strs := strings.Split(str, " ")
			if len(str) > 1 {
				switch strs[0] {
				case "@generate":
					{
						comments = append(comments, strs[1])
					}
				}
			}
		}
	}

	l := methods.Len()

	out := &Service{
		Name:           string(service.Name()),
		Rpcs:           make([]*Rpc, 0, l),
		Options:        make(map[string]any),
		CodeGeneration: comments,
		File:           file,
	}

	if opts, ok := service.Options().(*descriptorpb.ServiceOptions); ok {
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			key = toGoName(key)
			if v, ok := a.(*dynamicpb.Message); ok {
				data := make(map[string]any)
				v.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
					data[toGoName(string(fd.Name()))] = file.getInnerOptions("", v.Interface())
					return true
				})
				out.Options[key] = data
				return true
			}
			out.Options[key] = a
			return true
		})
	}

	if l == 0 {
		return out, nil
	}

	for i := 0; i < l; i++ {
		methodDescriptor := methods.Get(i)

		rpc, err := file.GetRpc(fmt.Sprintf(".service[%d].method[%d].options", n, i), string(service.Name()), methodDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to get field %s: %w", methodDescriptor.Name(), err)
		}
		out.Rpcs = append(out.Rpcs, rpc)
	}

	out.RpcOptions = make(map[string]any)
	for _, rpc := range out.Rpcs {
		ConcatOptions(out.RpcOptions, rpc.Options)
		tmpMap := make(map[string]any)
		ConcatOptionValues(tmpMap, rpc.Options)
		rpc.Options = tmpMap
	}

	return out, nil
}

func ConcatOptions(dest map[string]any, src map[string]any) {
	for key, value := range src {
		if value, ok := value.(map[string]any); ok {
			v := make(map[string]any)
			dest[key] = v
			ConcatOptions(v, value)
			continue
		}
		if value, ok := value.([]any); ok {
			v := make(map[string]any)
			dest[key] = []any{v}
			for _, val := range value {
				if value, ok := val.(map[string]any); ok {
					ConcatOptions(v, value)
					continue
				}
			}
			continue
		}
		dest[key] = fmt.Sprintf("%T", value)
	}
}

func ConcatOptionValues(dest map[string]any, src map[string]any) {
	for key, value := range src {
		if value, ok := value.(map[string]any); ok {
			v := make(map[string]any)
			dest[key] = v
			ConcatOptionValues(v, value)
			continue
		}
		if value, ok := value.([]any); ok {
			allKeys := make(map[string]any)
			for _, val := range value {
				if value, ok := val.(map[string]any); ok {
					v := make(map[string]any)
					ConcatOptionValues(v, value)
					maps.Copy(allKeys, v)
					continue
				}
			}
			for _, val := range value {
				if value, ok := val.(map[string]any); ok {
					for k, val := range allKeys {
						if _, ok := value[k]; !ok {
							value[k] = reflect.Zero(reflect.TypeOf(val)).Interface()
						}
					}
				}
			}
		}
		dest[key] = value
	}
}

// GetRpc creates a Rpc from a method descriptor.
func (file *File) GetRpc(path string, serviceName string, fd protoreflect.MethodDescriptor) (*Rpc, error) {
	input := fd.Input().Name()
	output := fd.Output().Name()

	out := &Rpc{
		Name:        string(fd.Name()),
		Input:       string(input),
		Output:      string(output),
		Options:     make(map[string]any),
		ServiceName: serviceName,
	}

	if opts, ok := fd.Options().(*descriptorpb.MethodOptions); ok {
		proto.RangeExtensions(opts, func(et protoreflect.ExtensionType, a any) bool {
			fieldDescriptor := protodesc.ToFieldDescriptorProto(et.TypeDescriptor().Descriptor())
			n1 := -1
			if fieldDescriptor.Number != nil {
				n1 = int(*fieldDescriptor.Number)
			}
			key := fmt.Sprintf("%s.%s",
				et.TypeDescriptor().Parent().FullName().Name(),
				et.TypeDescriptor().FullName().Name())
			key = toGoName(key)
			out.Options[key] = file.getInnerOptions(fmt.Sprintf("%s.%d", path, n1), a)
			return true
		})
	}

	return out, nil
}

// Helper functions

func (file *File) getInnerOptions(optionPath string, v any) any {
	if v, ok := v.(*dynamicpb.Message); ok {
		out := make(map[string]any)
		v.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			fieldDescriptor := protodesc.ToFieldDescriptorProto(fd)
			n := -1
			if fieldDescriptor.Number != nil {
				n = int(*fieldDescriptor.Number)
			}
			out[toGoName(string(fd.Name()))] = file.getInnerOptions(fmt.Sprintf("%s.%d", optionPath, n), v.Interface())
			return true
		})
		return out
	}
	if list, ok := v.(protoreflect.List); ok {
		out := make([]any, list.Len())
		for i := 0; i < list.Len(); i++ {
			out[i] = file.getInnerOptions(fmt.Sprintf("%s.%d", optionPath, i), list.Get(i).Interface())
		}
		return out
	}
	if value, ok := file.Comments[optionPath]; ok {
		switch value {
		case "@embed":
			{
				data, err := os.ReadFile(path.Join(file.Dir, v.(string)))
				if err != nil {
					panic(err)
				}

				return ByteString(StringToGoByteArray(string(data)))
			}
		}
	}
	return v
}

func canBeIgnored(fd protoreflect.FieldDescriptor) (bool, string) {
	if fd.IsMap() {
		return true, string(fd.Message().Name())
	}
	return false, ""
}

func cleanType(typ string) string {
	typ = strings.ReplaceAll(typ, "*", "")
	typ = strings.ReplaceAll(typ, "[]", "")
	return typ
}

func getKind(fd protoreflect.FieldDescriptor) string {
	if fd.IsMap() {
		return fmt.Sprintf("map[%s]%s", getKind(fd.MapKey()), getKind(fd.MapValue()))
	}

	var prefix string
	if fd.HasOptionalKeyword() || fd.Kind() == protoreflect.MessageKind {
		prefix = "*"
	}
	if fd.IsList() {
		prefix = "[]"
	}

	var baseType string
	switch fd.Kind() {
	case protoreflect.BoolKind:
		baseType = "bool"
	case protoreflect.EnumKind:
		baseType = string(fd.Enum().Name())
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		baseType = "int"
	case protoreflect.Uint32Kind:
		baseType = "uint"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		baseType = "int64"
	case protoreflect.Uint64Kind:
		baseType = "uint64"
	case protoreflect.FloatKind:
		baseType = "float32"
	case protoreflect.DoubleKind:
		baseType = "float64"
	case protoreflect.StringKind:
		baseType = "string"
	case protoreflect.BytesKind:
		return "[]byte"
	case protoreflect.MessageKind:
		baseType = string(fd.Message().Name())
	case protoreflect.GroupKind:
		return "interface{}"
	default:
		return ""
	}

	return prefix + baseType
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

func getReflectedKind(k protoreflect.Kind) reflect.Kind {
	switch k {
	case protoreflect.BoolKind:
		return reflect.Bool
	case protoreflect.EnumKind, protoreflect.Int32Kind, protoreflect.Sint32Kind,
		protoreflect.Sfixed32Kind, protoreflect.Fixed32Kind:
		return reflect.Int
	case protoreflect.Uint32Kind:
		return reflect.Uint
	case protoreflect.Int64Kind, protoreflect.Sint64Kind,
		protoreflect.Sfixed64Kind, protoreflect.Fixed64Kind:
		return reflect.Int64
	case protoreflect.Uint64Kind:
		return reflect.Uint64
	case protoreflect.FloatKind:
		return reflect.Float32
	case protoreflect.DoubleKind:
		return reflect.Float64
	case protoreflect.StringKind:
		return reflect.String
	case protoreflect.BytesKind:
		return reflect.Array
	case protoreflect.MessageKind:
		return reflect.Struct
	default:
		return reflect.Invalid
	}
}

func toGoName(s string) string {
	if s == "" {
		return ""
	}

	segments := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '.'
	})
	for i, segment := range segments {
		if segment == "" {
			continue
		}
		runes := []rune(segment)
		runes[0] = unicode.ToUpper(runes[0])
		segments[i] = string(runes)
	}
	return strings.Join(segments, "")
}

func StringToGoByteArray(str string) string {
	buffer := bytes.NewBufferString("[]byte {")
	line := bytes.NewBufferString("")
	for i := 0; i < len(str); i++ {
		if i != 0 && i%16 == 0 {
			buffer.WriteString("\r\n")
			buffer.WriteString("\t")
			buffer.Write(line.Bytes())
			line.Reset()
		}
		line.WriteString(fmt.Sprintf("0x%02x, ", str[i]))
	}
	buffer.WriteString("\r\n")
	buffer.WriteString("\t")
	buffer.Write(line.Bytes())
	buffer.WriteString("\r\n")
	buffer.WriteString("}")
	return buffer.String()
}
