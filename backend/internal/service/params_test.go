package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// userGetField 与 example.thrift 的 getUser(1: GetUserRequest req) 对应，
// GetUserRequest 字段: userId
func userGetField() requestField {
	return requestField{
		Name:     "req",
		IsStruct: true,
		Fields:   map[string]struct{}{"userId": {}},
	}
}

func TestNormalizeParams_SingleStructArg_Flat(t *testing.T) {
	out, err := NormalizeParams(`{"userId": 1}`, userGetField(), "getUser")
	assert.NoError(t, err)
	assert.JSONEq(t, `{"userId": 1}`, out)
}

func TestNormalizeParams_SingleStructArg_WrappedUnwraps(t *testing.T) {
	// 旧版按参数名包装的写法应被解包，而不是静默丢值
	out, err := NormalizeParams(`{"req": {"userId": 1}}`, userGetField(), "getUser")
	assert.NoError(t, err)
	assert.JSONEq(t, `{"userId": 1}`, out)
}

func TestNormalizeParams_SingleStructArg_EmptyObject(t *testing.T) {
	out, err := NormalizeParams(`{}`, userGetField(), "getUser")
	assert.NoError(t, err)
	assert.JSONEq(t, `{}`, out)
}

func TestNormalizeParams_SingleStructArg_NoMatchingKey(t *testing.T) {
	// 键与字段全部不匹配（如拼错字段名）必须报错，而不是零值发出
	_, err := NormalizeParams(`{"userid": 1}`, userGetField(), "getUser")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "全部不匹配")
}

func TestNormalizeParams_SingleStructArg_NotObject(t *testing.T) {
	_, err := NormalizeParams(`123`, userGetField(), "getUser")
	assert.Error(t, err)
}

func TestNormalizeParams_InvalidJSON(t *testing.T) {
	_, err := NormalizeParams(`not-json`, userGetField(), "getUser")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不是合法的 JSON")
}

func TestNormalizeParams_SingleScalarArg(t *testing.T) {
	field := requestField{Name: "msg"}
	out, err := NormalizeParams(`"hello"`, field, "ping")
	assert.NoError(t, err)
	assert.Equal(t, `"hello"`, out)
}

func TestNormalizeParams_SingleScalarArg_UnwrapsSingleKey(t *testing.T) {
	field := requestField{Name: "msg"}
	out, err := NormalizeParams(`{"msg": "hello"}`, field, "ping")
	assert.NoError(t, err)
	assert.Equal(t, `"hello"`, out)
}

func TestNormalizeParams_SingleScalarArg_EmptyObject(t *testing.T) {
	// 占位参数方法（ping 的 _placeholder）界面会传 {}
	field := requestField{Name: "_placeholder"}
	out, err := NormalizeParams(`{}`, field, "ping")
	assert.NoError(t, err)
	assert.Equal(t, `{}`, out)
}

func TestNormalizeParams_SingleScalarArg_WrongKeys(t *testing.T) {
	field := requestField{Name: "msg"}
	_, err := NormalizeParams(`{"other": "x"}`, field, "ping")
	assert.Error(t, err)
}

func TestNormalizeParams_EmptyInput(t *testing.T) {
	out, err := NormalizeParams("", userGetField(), "getUser")
	assert.NoError(t, err)
	assert.Equal(t, "{}", out)
}

// TestLoadRequestField 使用真实 IDL 验证描述符加载
func TestLoadRequestField(t *testing.T) {
	dir := t.TempDir()
	idlPath := filepath.Join(dir, "example.thrift")
	content := `namespace go example

struct GetUserRequest {
    1: i64 userId
}

struct GetUserResponse {
    1: GetUserRequest user
}

service UserService {
    GetUserResponse getUser(1: GetUserRequest req)
    string getLicenseInfo(1: required string token, 2: required string device)
    string ping()
}`
	assert.NoError(t, os.WriteFile(idlPath, []byte(content), 0644))

	// 方法不存在 → 报错
	_, err := loadRequestField(idlPath, "notExist")
	assert.Error(t, err)

	// 先走 parser 的修复流程（ping() 无参数需要补占位）
	p := NewThriftParser()
	info, err := p.ParseIDL(idlPath)
	assert.NoError(t, err)

	field, err := loadRequestField(info.FixedIDLPath, "getUser")
	assert.NoError(t, err)
	assert.Equal(t, "req", field.Name)
	assert.True(t, field.IsStruct)
	assert.Contains(t, field.Fields, "userId")

	// 多参数方法：描述符只保留第一个参数（Kitex 泛化的限制）
	licField, err := loadRequestField(info.FixedIDLPath, "getLicenseInfo")
	assert.NoError(t, err)
	assert.Equal(t, "token", licField.Name)
	assert.False(t, licField.IsStruct)

	pingField, err := loadRequestField(info.FixedIDLPath, "ping")
	assert.NoError(t, err)
	assert.Equal(t, "_placeholder", pingField.Name)
}
