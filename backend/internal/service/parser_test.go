package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"thrift-perf-tool/internal/model"
)

// ===================== removeComments =====================

func TestRemoveComments_LineComment(t *testing.T) {
	p := NewThriftParser()
	result := p.removeComments(`service Foo { void bar() } // this is a comment`)
	assert.Equal(t, `service Foo { void bar() } `, result)
}

func TestRemoveComments_BlockComment(t *testing.T) {
	p := NewThriftParser()
	result := p.removeComments(`service Foo { /* block comment */ void bar() }`)
	assert.Equal(t, `service Foo {  void bar() }`, result)
}

func TestRemoveComments_ShellComment(t *testing.T) {
	p := NewThriftParser()
	result := p.removeComments(`service Foo { void bar() } # shell comment`)
	assert.Equal(t, `service Foo { void bar() } `, result)
}

func TestRemoveComments_MixedComments(t *testing.T) {
	p := NewThriftParser()
	input := `/* block */ service Foo { // line
void bar() # shell
}`
	result := p.removeComments(input)
	assert.Contains(t, result, "service Foo")
	assert.NotContains(t, result, "/*")
	assert.NotContains(t, result, "//")
	assert.NotContains(t, result, "#")
}

func TestRemoveComments_NoComments(t *testing.T) {
	p := NewThriftParser()
	input := `service Foo { void bar() }`
	result := p.removeComments(input)
	assert.Equal(t, input, result)
}

// ===================== parseMethods =====================

func TestParseMethods_SingleMethod(t *testing.T) {
	p := NewThriftParser()
	methods := p.parseMethods("string getUser(1: required i32 userId)", nil)
	assert.Len(t, methods, 1)
	assert.Equal(t, "getUser", methods[0].Name)
	assert.Equal(t, "string", methods[0].ReturnType)
}

func TestParseMethods_MultipleMethods(t *testing.T) {
	p := NewThriftParser()
	body := `string getUser(1: required i32 userId)
	void addUser(1: required User user)`
	methods := p.parseMethods(body, nil)
	assert.Len(t, methods, 2)
	assert.Equal(t, "getUser", methods[0].Name)
	assert.Equal(t, "addUser", methods[1].Name)
}

func TestParseMethods_WithReturnFields(t *testing.T) {
	p := NewThriftParser()
	structs := map[string]model.StructInfo{
		"User": {Name: "User", Fields: []model.FieldInfo{{Index: 1, Name: "id", Type: "i32"}}},
	}
	methods := p.parseMethods("User getUser(1: required i32 userId)", structs)
	assert.Len(t, methods, 1)
	assert.Equal(t, "getUser", methods[0].Name)
	assert.Len(t, methods[0].ReturnFields, 1)
	assert.Equal(t, "id", methods[0].ReturnFields[0].Name)
}

// ===================== parseParams =====================

func TestParseParams_RequiredParam(t *testing.T) {
	p := NewThriftParser()
	params := p.parseParams("1: required i32 userId", nil)
	assert.Len(t, params, 1)
	assert.Equal(t, "userId", params[0].Name)
	assert.Equal(t, "i32", params[0].Type)
	assert.True(t, params[0].Required)
}

func TestParseParams_OptionalParam(t *testing.T) {
	p := NewThriftParser()
	params := p.parseParams("1: optional string name", nil)
	assert.Len(t, params, 1)
	assert.False(t, params[0].Required)
}

func TestParseParams_MultipleParams(t *testing.T) {
	p := NewThriftParser()
	params := p.parseParams("1: required i32 userId, 2: optional string name", nil)
	assert.Len(t, params, 2)
	assert.Equal(t, "userId", params[0].Name)
	assert.Equal(t, "name", params[1].Name)
}

func TestParseParams_EmptyString(t *testing.T) {
	p := NewThriftParser()
	params := p.parseParams("", nil)
	assert.Empty(t, params)
}

func TestParseParams_WithStructFields(t *testing.T) {
	p := NewThriftParser()
	structs := map[string]model.StructInfo{
		"User": {Name: "User", Fields: []model.FieldInfo{{Index: 1, Name: "id", Type: "i32"}}},
	}
	params := p.parseParams("1: required User user", structs)
	assert.Len(t, params, 1)
	assert.Equal(t, "User", params[0].Type)
	assert.Len(t, params[0].StructFields, 1)
}

// ===================== parseStructs + parseStructFields =====================

func TestParseStructs_SingleStruct(t *testing.T) {
	p := NewThriftParser()
	result := p.parseStructs(`struct User { 1: required i32 id, 2: optional string name }`)
	assert.Contains(t, result, "User")
	assert.Len(t, result["User"].Fields, 2)
}

func TestParseStructs_MultipleStructs(t *testing.T) {
	p := NewThriftParser()
	content := `struct User { 1: required i32 id }
struct Result { 1: required bool success }`
	result := p.parseStructs(content)
	assert.Contains(t, result, "User")
	assert.Contains(t, result, "Result")
}

func TestParseStructFields_FullFields(t *testing.T) {
	p := NewThriftParser()
	fields := p.parseStructFields("1: required i32 id, 2: optional string name")
	assert.Len(t, fields, 2)
	assert.Equal(t, 1, fields[0].Index)
	assert.Equal(t, "id", fields[0].Name)
	assert.Equal(t, "i32", fields[0].Type)
	assert.True(t, fields[0].Required)
	assert.Equal(t, 2, fields[1].Index)
	assert.Equal(t, "name", fields[1].Name)
	assert.False(t, fields[1].Required)
}

func TestParseStructFields_EmptyBody(t *testing.T) {
	p := NewThriftParser()
	fields := p.parseStructFields("")
	assert.Empty(t, fields)
}

// ===================== ParseIDL (integration) =====================

func TestParseIDL_ValidFile(t *testing.T) {
	dir := t.TempDir()
	idlContent := `
struct User {
    1: required i32 id
    2: optional string name
}

service UserService {
    User getUser(1: required i32 userId)
    void addUser(1: required User user)
}
`
	idlPath := filepath.Join(dir, "test.thrift")
	err := os.WriteFile(idlPath, []byte(idlContent), 0644)
	assert.NoError(t, err)

	p := NewThriftParser()
	info, err := p.ParseIDL(idlPath)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Len(t, info.Services, 1)
	assert.Equal(t, "UserService", info.Services[0].Name)
	assert.Len(t, info.Services[0].Methods, 2)
	assert.Contains(t, info.Structs, "User")
	assert.NotEmpty(t, info.FixedIDLPath)

	// Verify .fixed file was created
	_, err = os.Stat(info.FixedIDLPath)
	assert.NoError(t, err)
}

func TestParseIDL_FileNotExist(t *testing.T) {
	p := NewThriftParser()
	_, err := p.ParseIDL("/nonexistent/path/test.thrift")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "IDL 文件不存在")
}

func TestParseIDL_NoService(t *testing.T) {
	dir := t.TempDir()
	idlPath := filepath.Join(dir, "empty.thrift")
	err := os.WriteFile(idlPath, []byte("struct Foo { 1: required i32 id }"), 0644)
	assert.NoError(t, err)

	p := NewThriftParser()
	_, err = p.ParseIDL(idlPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "未找到任何 service 定义")
}

func TestParseIDL_ContainerTypes(t *testing.T) {
	dir := t.TempDir()
	idlPath := filepath.Join(dir, "container.thrift")
	content := `namespace go example

struct Tags {
    1: map<string, string> labels,
    2: list<i32> scores,
    3: map<string, list<i64>> index,
    4: optional set<string> names,
    5: i64 plain
}

struct QueryResult {
    1: map<string, Tags> groups
}

service TagService {
    QueryResult query(1: map<string, string> filters, 2: list<i32> ids)
    map<string, string> simple(1: Tags tags)
}`
	err := os.WriteFile(idlPath, []byte(content), 0644)
	assert.NoError(t, err)

	p := NewThriftParser()
	info, err := p.ParseIDL(idlPath)
	assert.NoError(t, err)
	assert.Len(t, info.Services, 1)

	methods := info.Services[0].Methods
	assert.Len(t, methods, 2)

	// query 方法：容器类型参数应完整解析
	q := methods[0]
	assert.Equal(t, "query", q.Name)
	assert.Len(t, q.Params, 2)
	assert.Equal(t, "map<string, string>", q.Params[0].Type)
	assert.Equal(t, "filters", q.Params[0].Name)
	assert.Equal(t, "list<i32>", q.Params[1].Type)
	// 返回类型与字段
	assert.Equal(t, "QueryResult", q.ReturnType)
	assert.NotEmpty(t, q.ReturnFields)

	// simple 方法：容器返回类型 + struct 参数
	s := methods[1]
	assert.Equal(t, "simple", s.Name)
	assert.Equal(t, "map<string, string>", s.ReturnType)
	assert.Equal(t, "Tags", s.Params[0].Type)

	// struct 字段中的容器类型
	tags := info.Structs["Tags"]
	assert.Len(t, tags.Fields, 5)
	assert.Equal(t, "map<string, string>", tags.Fields[0].Type)
	assert.Equal(t, "list<i32>", tags.Fields[1].Type)
	assert.Equal(t, "map<string, list<i64>>", tags.Fields[2].Type)
	assert.Equal(t, "set<string>", tags.Fields[3].Type)
	assert.False(t, tags.Fields[3].Required)
}
