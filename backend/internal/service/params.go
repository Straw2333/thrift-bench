package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/cloudwego/kitex/pkg/generic"
	"github.com/cloudwego/kitex/pkg/generic/descriptor"
)

// requestField 描述泛化调用请求中的一个字段
type requestField struct {
	Name     string
	Optional bool
	IsStruct bool
	// Fields 仅当 IsStruct 时有效：嵌套结构体的字段名集合（用于校验与解包判断）
	Fields map[string]struct{}
}

// loadRequestField 从 IDL 描述符中提取指定方法的请求字段。
//
// Kitex 泛化调用仅支持单参数方法：其内部解析器只保留方法的第一个参数
// （见 kitex generic/thrift/parse.go: "only support single argument"），
// 请求描述符即该参数（参数本身可以是包含多个字段的 struct）。
// 使用独立的 provider 实例读取描述符，与客户端使用的 provider 互不干扰。
func loadRequestField(idlPath, method string) (requestField, error) {
	provider, err := generic.NewThriftFileProvider(idlPath)
	if err != nil {
		return requestField{}, fmt.Errorf("解析 IDL 描述符失败: %w", err)
	}
	defer provider.Close()

	svcDesc := <-provider.Provide()
	fn, err := svcDesc.LookupFunctionByMethod(method)
	if err != nil {
		return requestField{}, fmt.Errorf("IDL 中不存在方法 %q（请检查方法名是否与 IDL 一致）", method)
	}
	if fn.Request == nil || fn.Request.Struct == nil || len(fn.Request.Struct.FieldsByName) == 0 {
		return requestField{}, fmt.Errorf("方法 %q 的参数描述为空", method)
	}

	// 按字段 ID 排序后取唯一字段（当前 Kitex 版本下恒为 1 个）
	fields := make([]requestField, 0, len(fn.Request.Struct.FieldsByName))
	for _, f := range fn.Request.Struct.FieldsByName {
		rf := requestField{Name: f.Name, Optional: f.Optional}
		if f.Type != nil && f.Type.Type == descriptor.STRUCT &&
			f.Type.Struct != nil && len(f.Type.Struct.FieldsByName) > 0 {
			rf.IsStruct = true
			rf.Fields = make(map[string]struct{}, len(f.Type.Struct.FieldsByName))
			for name := range f.Type.Struct.FieldsByName {
				rf.Fields[name] = struct{}{}
			}
		}
		fields = append(fields, rf)
	}
	sort.Slice(fields, func(i, j int) bool { return fields[i].Name < fields[j].Name })

	if len(fields) != 1 {
		return requestField{}, fmt.Errorf("方法 %q 的请求参数描述包含 %d 个字段，超出泛化调用支持范围", method, len(fields))
	}
	return fields[0], nil
}

// NormalizeParams 将用户输入的 JSON 参数规范化为 Kitex 泛化调用期望的格式。
//
// 请求 JSON 的键必须直接是请求字段名（"扁平"格式）：
//   - 单个 struct 参数（如 getUser(1: GetUserRequest req)）：直接写该 struct 的字段 {"userId": 1}
//   - 单个标量参数：直接填裸值（如 "hello"、42）
//
// 按参数名包装的旧写法（{"req": {"userId": 1}}）会导致字段值被静默丢弃
// （服务端收到零值但调用"成功"），因此这里对包装写法做一层解包兼容，
// 并对无法匹配任何字段的输入直接报错，避免静默失败。
func NormalizeParams(raw string, f requestField, method string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "{}", nil
	}

	var body interface{}
	if err := json.Unmarshal([]byte(raw), &body); err != nil {
		return "", fmt.Errorf("参数不是合法的 JSON: %v", err)
	}

	obj, isObj := body.(map[string]interface{})

	if f.IsStruct {
		if !isObj {
			return "", fmt.Errorf("方法 %q 的参数 %q 是 struct 类型，应填写 JSON 对象（字段: %s）",
				method, f.Name, joinNames(f.Fields))
		}
		// 兼容旧版按参数名包装的写法：{"req": {...}} → {...}
		if inner, wrapped := obj[f.Name]; wrapped {
			if _, isNestedField := f.Fields[f.Name]; !isNestedField {
				if innerObj, ok := inner.(map[string]interface{}); ok {
					obj = innerObj
				}
			}
		}
		if err := validateKeys(obj, f.Fields, method); err != nil {
			return "", err
		}
		return marshalCompact(obj)
	}

	// 标量参数：裸值直接使用
	if !isObj {
		return marshalCompact(body)
	}
	// {} 表示不传值（如占位参数方法）
	if len(obj) == 0 {
		return "{}", nil
	}
	// 兼容 {"paramName": value} 的写法，解包为裸值
	if len(obj) == 1 {
		if v, ok := obj[f.Name]; ok {
			return marshalCompact(v)
		}
	}
	return "", fmt.Errorf("方法 %q 只有单个标量参数 %q，请直接填写值本身（不要包 JSON 对象）",
		method, f.Name)
}

// validateKeys 校验对象的键至少有一个能匹配到已知字段名，
// 防止键名拼错/格式错误导致全部字段被静默丢弃（零值发出但调用"成功"）。
// 空对象（不传任何字段）与包含未知额外键的情况均放行。
func validateKeys(obj map[string]interface{}, known map[string]struct{}, method string) error {
	if len(obj) == 0 || len(known) == 0 {
		return nil
	}
	for k := range obj {
		if _, ok := known[k]; ok {
			return nil
		}
	}
	return fmt.Errorf("参数的键与方法 %q 的字段 %s 全部不匹配（注意：字段名需要与 IDL 一致，且不需要按参数名包装）",
		method, joinNames(known))
}

func marshalCompact(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("参数序列化失败: %w", err)
	}
	return string(b), nil
}

func joinNames(names map[string]struct{}) string {
	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}
