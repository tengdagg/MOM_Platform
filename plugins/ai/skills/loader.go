package skills

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"gopkg.in/yaml.v3"
)

// SkillMeta SKILL.md 中的 YAML 前置元数据
type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`
	RiskLevel   string `yaml:"riskLevel"`
	ScriptType  string `yaml:"scriptType"`
	Parameters  any    `yaml:"parameters"`
}

// ExecuteFunc Skill 执行函数类型
type ExecuteFunc func(ctx biz.SkillContext) (any, error)

// BuiltinSkill 基于 SKILL.md 的内置技能
// 元数据从 SKILL.md YAML 前置元数据加载，执行逻辑由 Go 函数提供
type BuiltinSkill struct {
	meta      SkillMeta
	params    json.RawMessage
	markdown  string // SKILL.md 中 YAML 之后的 Markdown 指令部分
	executeFn ExecuteFunc
}

func (s *BuiltinSkill) Name() string                                   { return s.meta.Name }
func (s *BuiltinSkill) Description() string                            { return s.meta.Description }
func (s *BuiltinSkill) RiskLevel() string                              { return s.meta.RiskLevel }
func (s *BuiltinSkill) Category() string                               { return s.meta.Category }
func (s *BuiltinSkill) Parameters() json.RawMessage                    { return s.params }
func (s *BuiltinSkill) Markdown() string                               { return s.markdown }
func (s *BuiltinSkill) Execute(ctx biz.SkillContext) (any, error) {
	if s.executeFn == nil {
		return nil, fmt.Errorf("skill %s 未绑定执行函数", s.meta.Name)
	}
	return s.executeFn(ctx)
}

// ParseSKILLMD 解析 SKILL.md 文件内容
// 返回 YAML 前置元数据和 Markdown 指令部分
func ParseSKILLMD(content []byte) (*SkillMeta, string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	var inFrontmatter bool
	var frontmatterLines []string
	var markdownLines []string
	frontmatterDone := false

	for scanner.Scan() {
		line := scanner.Text()
		if !inFrontmatter && !frontmatterDone && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			inFrontmatter = false
			frontmatterDone = true
			continue
		}
		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterDone {
			markdownLines = append(markdownLines, line)
		}
	}

	if len(frontmatterLines) == 0 {
		return nil, "", fmt.Errorf("SKILL.md 缺少 YAML 前置元数据")
	}

	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(strings.Join(frontmatterLines, "\n")), &meta); err != nil {
		return nil, "", fmt.Errorf("解析 YAML 前置元数据失败: %w", err)
	}

	if meta.Name == "" {
		return nil, "", fmt.Errorf("SKILL.md 中 name 字段不能为空")
	}
	if meta.Description == "" {
		return nil, "", fmt.Errorf("SKILL.md 中 description 字段不能为空")
	}

	markdown := strings.TrimSpace(strings.Join(markdownLines, "\n"))
	return &meta, markdown, nil
}

// LoadBuiltinSkill 从嵌入的文件系统加载 SKILL.md 并创建技能实例
func LoadBuiltinSkill(name string, executeFn ExecuteFunc) (*BuiltinSkill, error) {
	path := name + "/SKILL.md"
	content, err := SkillFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取 %s 失败: %w", path, err)
	}

	meta, markdown, err := ParseSKILLMD(content)
	if err != nil {
		return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
	}

	// 将 parameters 转为 json.RawMessage
	var params json.RawMessage
	if meta.Parameters != nil {
		paramsJSON, err := json.Marshal(meta.Parameters)
		if err != nil {
			return nil, fmt.Errorf("序列化 %s 的 parameters 失败: %w", name, err)
		}
		params = paramsJSON
	} else {
		params = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	return &BuiltinSkill{
		meta:      *meta,
		params:    params,
		markdown:  markdown,
		executeFn: executeFn,
	}, nil
}

// MustLoadBuiltinSkill LoadBuiltinSkill 的 panic 版本，启动时使用
func MustLoadBuiltinSkill(name string, executeFn ExecuteFunc) *BuiltinSkill {
	skill, err := LoadBuiltinSkill(name, executeFn)
	if err != nil {
		panic(fmt.Sprintf("加载内置 Skill %s 失败: %v", name, err))
	}
	return skill
}

// RegisterAllBuiltinSkills 注册所有内置技能（统一入口）
func RegisterAllBuiltinSkills(registry *biz.ToolRegistry) {
	RegisterHostSkills(registry)
	RegisterK8sSkills(registry)
	RegisterAuditSkills(registry)
	RegisterTaskSkills(registry)
	RegisterMonitorSkills(registry)
	RegisterCloudSkills(registry)
	RegisterAnalysisSkills(registry)
}
