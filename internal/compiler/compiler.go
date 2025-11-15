package compiler

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"maps"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"
	"unicode"

	"github.com/bufbuild/protocompile"
	"github.com/bufbuild/protocompile/linker"
	"github.com/bufbuild/protocompile/protoutil"
	"github.com/google/uuid"
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

type (
	Field struct {
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

	EnumValue struct {
		Name   string
		Number int
	}

	Enum struct {
		Name    string
		Values  []*EnumValue
		Options map[string]any
		File    *File
	}

	Message struct {
		Name       string
		Fields     []*Field
		Ignorables Ignorables
		Options    map[string]any
		Descriptor string
		TypeName   string
		File       *File
	}

	Service struct {
		Name           string
		Options        map[string]any
		Descriptor     string
		Rpcs           []*Rpc
		RpcOptions     map[string]any
		CodeGeneration []string
		File           *File
	}

	Rpc struct {
		Name        string
		Options     map[string]any
		Descriptor  string
		Input       string
		Output      string
		ServiceName string
	}

	File struct {
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

	AST struct {
		Files []*File
	}
)

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

func GetFile(dir string, filePath string, file linker.File) (*File, error) {
	out := &File{
		Options: make(map[string]any),
	}
	out.Dir = dir
	_, out.Source = path.Split(filePath)
	out.FileName = strings.ReplaceAll(strings.ToLower(out.Source), ".proto", "")
	protodesc := protodesc.ToFileDescriptorProto(file)
	out.Comments = GetComments(protodesc, file)

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

		protoDescriptor := protoutil.ProtoFromMessageDescriptor(messageDescriptor)
		jsonData, err := protojson.Marshal(protoDescriptor)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal descriptor for %s: %w", name, err)
		}
		message.Descriptor = base64.StdEncoding.EncodeToString(jsonData)

		out = append(out, message)

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

func (file *File) GetService(n int, service protoreflect.ServiceDescriptor) (*Service, error) {
	methods := service.Methods()

	codeGeneration := make([]string, 0)
	if value, ok := file.Comments[fmt.Sprintf(".service[%d]", n)]; ok {
		comments := ExpandComments(value)
		for _, comment := range comments {
			switch comment[0] {
			case "@generate":
				{
					codeGeneration = append(codeGeneration, comment[1])
				}
			}
		}
	}

	l := methods.Len()

	out := &Service{
		Name:           string(service.Name()),
		Rpcs:           make([]*Rpc, 0, l),
		Options:        make(map[string]any),
		CodeGeneration: codeGeneration,
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
		comments := ExpandComments(value)
		for _, comment := range comments {
			switch comment[0] {
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

func ExpandComments(comment string) [][2]string {
	lines := strings.FieldsFunc(comment, func(r rune) bool {
		return r == '\r' || r == '\n'
	})

	out := make([][2]string, 0)

	for _, line := range lines {
		str := Trim(line)
		segments := strings.Split(str, " ")
		if len(segments) <= 1 {
			continue
		}
		if strings.HasPrefix(segments[0], "@") {
			out = append(out, [2]string{segments[0], strings.Join(segments[1:], " ")})
		}
	}
	return out
}

func Trim(str string) string {
	out := strings.TrimRightFunc(str, func(r rune) bool {
		return r == ' ' || r == '\r'
	})
	out = strings.TrimLeft(out, " ")

	return out
}

func GetComments(protodesc *descriptorpb.FileDescriptorProto, file linker.File) map[string]string {
	out := make(map[string]string)
	for _, i := range protodesc.SourceCodeInfo.Location {
		if i.LeadingComments != nil {
			path := file.SourceLocations().ByPath(i.Path).Path.String()
			value := strings.TrimRight(*i.LeadingComments, "\r\n")
			value = strings.TrimRight(value, " ")
			value = strings.TrimLeft(value, " ")
			out[path] = value
		}
	}
	return out
}
