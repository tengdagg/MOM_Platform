package biz

import (
	"encoding/json"
	"fmt"
	"log"

	"gorm.io/gorm"
)

// SkillEngine Skill 执行引擎
// 管理自定义 Skills 的加载、注册和执行
type SkillEngine struct {
	db       *gorm.DB
	registry *ToolRegistry
	sandbox  *ScriptSandbox
}

// NewSkillEngine 创建 Skill 引擎
func NewSkillEngine(db *gorm.DB, registry *ToolRegistry) *SkillEngine {
	return &SkillEngine{
		db:       db,
		registry: registry,
		sandbox:  NewScriptSandbox(db),
	}
}

// LoadCustomSkills 从数据库加载自定义 Skills 并注册到 ToolRegistry
// 同时反注册已被禁用的 Skills
func (e *SkillEngine) LoadCustomSkills() {
	var enabledSkills []SkillDefinition
	if err := e.db.Where("is_enabled = ? AND script_type != 'builtin'", true).Find(&enabledSkills).Error; err != nil {
		log.Printf("[skill-engine] 加载自定义 Skills 失败: %v", err)
		return
	}

	// 反注册已禁用的自定义 Skills
	var disabledSkills []SkillDefinition
	if err := e.db.Where("is_enabled = ? AND script_type != 'builtin'", false).Find(&disabledSkills).Error; err == nil {
		for _, skill := range disabledSkills {
			e.registry.Unregister(skill.Name)
			log.Printf("[skill-engine] 反注册禁用的 Skill: %s", skill.Name)
		}
	}

	for _, skill := range enabledSkills {
		customSkill := &CustomScriptSkill{
			definition: skill,
			sandbox:    e.sandbox,
		}
		e.registry.Register(customSkill)
		log.Printf("[skill-engine] 注册自定义 Skill: %s (%s)", skill.Name, skill.ScriptType)
	}
}

// CustomScriptSkill 自定义脚本 Skill
type CustomScriptSkill struct {
	definition SkillDefinition
	sandbox    *ScriptSandbox
}

func (s *CustomScriptSkill) Name() string        { return s.definition.Name }
func (s *CustomScriptSkill) Description() string { return s.definition.Description }
func (s *CustomScriptSkill) RiskLevel() string   { return s.definition.RiskLevel }

// SetDefinition 设置 Skill 定义和沙箱（用于热加载注册）
func (s *CustomScriptSkill) SetDefinition(def SkillDefinition, sandbox *ScriptSandbox) {
	s.definition = def
	s.sandbox = sandbox
}

func (s *CustomScriptSkill) Parameters() json.RawMessage {
	if s.definition.Parameters != "" {
		return json.RawMessage(s.definition.Parameters)
	}
	return json.RawMessage(`{"type": "object", "properties": {}}`)
}

func (s *CustomScriptSkill) Execute(ctx SkillContext) (any, error) {
	if s.definition.ScriptBody == "" {
		return nil, fmt.Errorf("Skill %s 没有脚本内容", s.definition.Name)
	}

	result, err := s.sandbox.ExecuteScript(
		s.definition.ScriptType,
		s.definition.ScriptBody,
		ctx.Params,
	)
	if err != nil {
		return nil, fmt.Errorf("Skill %s 执行失败: %w", s.definition.Name, err)
	}

	return result, nil
}
