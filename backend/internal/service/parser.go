package service

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"thrift-perf-tool/internal/model"
)

// 预编译正则表达式（包级别，只编译一次）
var (
	// 移除注释
	reLineComment  = regexp.MustCompile(`//.*`)
	reBlockComment = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	reShellComment = regexp.MustCompile(`#.*`)

	// 类型模式：基础类型或容器类型（支持一层嵌套，如 map<string, list<i32>>）
	reTypePattern = `(?:\w+\s*<[^<>]*(?:<[^<>]*>)?[^<>]*>|\w+)`

	// 修复无参数函数：给空参数列表补一个占位参数（Kitex 拒绝解析无参数的方法）
	reEmptyArgs = regexp.MustCompile(`(` + reTypePattern + `)\s+(\w+)\s*\(\s*\)`)

	// 解析 service/struct/method/param/field
	reServiceDef = regexp.MustCompile(`service\s+(\w+)\s*{([^}]*)}`)
	reStructDef  = regexp.MustCompile(`struct\s+(\w+)\s*\{([^}]*)\}`)
	reMethodDef  = regexp.MustCompile(`(` + reTypePattern + `)\s+(\w+)\s*\(([^)]*)\)`)
	reParamDef   = regexp.MustCompile(`\d+\s*:\s*((?:required|optional)\s+)?(` + reTypePattern + `)\s+(\w+)`)
	reFieldDef   = regexp.MustCompile(`(\d+)\s*:\s*((?:optional|required)\s+)?(` + reTypePattern + `)\s+(\w+)`)
)

// ThriftParser Thrift IDL 解析器
type ThriftParser struct{}

// NewThriftParser 创建新的解析器
func NewThriftParser() *ThriftParser {
	return &ThriftParser{}
}

// ParseIDL 解析 Thrift IDL 文件
func (p *ThriftParser) ParseIDL(idlPath string) (*model.IDLInfo, error) {
	if _, err := os.Stat(idlPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("IDL 文件不存在: %s", idlPath)
	}

	content, err := os.ReadFile(idlPath)
	if err != nil {
		return nil, fmt.Errorf("读取 IDL 文件失败: %w", err)
	}

	// 修复无参数函数（Kitex 解析器会拒绝空参数列表的方法）
	fixedContent := reEmptyArgs.ReplaceAllString(string(content), "$1 $2(1: optional string _placeholder)")

	// 保存修复后的文件（用于 Kitex）
	fixedPath := idlPath + ".fixed"
	if err := os.WriteFile(fixedPath, []byte(fixedContent), 0644); err != nil {
		return nil, fmt.Errorf("保存修复后的 IDL 失败: %w", err)
	}

	// 解析文件内容提取服务和方法名
	idlInfo := p.parseIDLContent(fixedContent)
	idlInfo.FixedIDLPath = fixedPath

	if len(idlInfo.Services) == 0 {
		return nil, fmt.Errorf("未找到任何 service 定义")
	}

	return idlInfo, nil
}

// parseIDLContent 解析 Thrift IDL 内容，提取服务名、方法名和 struct 定义
func (p *ThriftParser) parseIDLContent(content string) *model.IDLInfo {
	idlInfo := &model.IDLInfo{
		Services: make([]model.ServiceInfo, 0),
		Structs:  make(map[string]model.StructInfo),
	}

	content = p.removeComments(content)
	idlInfo.Structs = p.parseStructs(content)

	serviceMatches := reServiceDef.FindAllStringSubmatch(content, -1)
	for _, match := range serviceMatches {
		if len(match) < 3 {
			continue
		}
		service := model.ServiceInfo{
			Name:    match[1],
			Methods: p.parseMethods(match[2], idlInfo.Structs),
		}
		idlInfo.Services = append(idlInfo.Services, service)
	}

	return idlInfo
}

// parseMethods 解析方法定义
func (p *ThriftParser) parseMethods(serviceBody string, structs map[string]model.StructInfo) []model.MethodInfo {
	methods := make([]model.MethodInfo, 0)
	methodMatches := reMethodDef.FindAllStringSubmatch(serviceBody, -1)

	for _, match := range methodMatches {
		if len(match) < 4 {
			continue
		}

		returnType := match[1]
		methodName := match[2]
		paramsStr := match[3]

		method := model.MethodInfo{
			Name:       methodName,
			ReturnType: returnType,
			Params:     p.parseParams(paramsStr, structs),
		}

		if structInfo, exists := structs[returnType]; exists {
			method.ReturnFields = structInfo.Fields
		}

		methods = append(methods, method)
	}

	return methods
}

// parseParams 解析参数列表
func (p *ThriftParser) parseParams(paramsStr string, structs map[string]model.StructInfo) []model.ParamInfo {
	params := make([]model.ParamInfo, 0)
	if strings.TrimSpace(paramsStr) == "" {
		return params
	}

	paramMatches := reParamDef.FindAllStringSubmatch(paramsStr, -1)
	for _, match := range paramMatches {
		if len(match) < 4 {
			continue
		}

		modifier := strings.TrimSpace(match[1])
		paramType := match[2]
		paramName := match[3]

		param := model.ParamInfo{
			Name:     paramName,
			Type:     paramType,
			Required: modifier == "required",
		}

		if structInfo, exists := structs[paramType]; exists {
			param.StructFields = structInfo.Fields
		}

		params = append(params, param)
	}

	return params
}

// parseStructs 解析 struct 定义
func (p *ThriftParser) parseStructs(content string) map[string]model.StructInfo {
	structs := make(map[string]model.StructInfo)
	structMatches := reStructDef.FindAllStringSubmatch(content, -1)

	for _, match := range structMatches {
		if len(match) < 3 {
			continue
		}
		structInfo := model.StructInfo{
			Name:   match[1],
			Fields: p.parseStructFields(match[2]),
		}
		structs[structInfo.Name] = structInfo
	}

	return structs
}

// parseStructFields 解析 struct 字段
func (p *ThriftParser) parseStructFields(structBody string) []model.FieldInfo {
	fields := make([]model.FieldInfo, 0)
	fieldMatches := reFieldDef.FindAllStringSubmatch(structBody, -1)

	for _, match := range fieldMatches {
		if len(match) < 5 {
			continue
		}

		index, _ := strconv.Atoi(match[1])
		modifier := strings.TrimSpace(match[2])

		fields = append(fields, model.FieldInfo{
			Index:    index,
			Name:     match[4],
			Type:     match[3],
			Required: modifier == "required",
		})
	}

	return fields
}

// removeComments 移除注释
func (p *ThriftParser) removeComments(content string) string {
	content = reBlockComment.ReplaceAllString(content, "")
	content = reLineComment.ReplaceAllString(content, "")
	content = reShellComment.ReplaceAllString(content, "")
	return content
}
