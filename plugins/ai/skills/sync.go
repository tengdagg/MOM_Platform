package skills

import (
	"fmt"

	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"gorm.io/gorm"
)

// SyncBuiltinSkillDefinitions 将运行时内置 Skill 元数据同步到数据库。
// 仅同步元数据字段，保留数据库中已有的 is_enabled 状态，避免覆盖用户启停配置。
func SyncBuiltinSkillDefinitions(db *gorm.DB, registry *biz.ToolRegistry) error {
	if db == nil || registry == nil {
		return nil
	}

	for _, skill := range registry.GetAll() {
		builtin, ok := skill.(*BuiltinSkill)
		if !ok {
			continue
		}

		params := string(builtin.Parameters())
		if params == "" {
			params = `{"type":"object","properties":{}}`
		}

		definition := biz.SkillDefinition{
			Name:        builtin.Name(),
			DisplayName: builtin.Name(),
			Description: builtin.Description(),
			Category:    builtin.Category(),
			Parameters:  params,
			IsBuiltin:   true,
			ScriptType:  "builtin",
			ScriptBody:  "",
			Markdown:    builtin.Markdown(),
			RiskLevel:   builtin.RiskLevel(),
		}

		var existing biz.SkillDefinition
		err := db.Where("name = ?", definition.Name).First(&existing).Error
		if err == nil {
			updates := map[string]any{
				"display_name": definition.DisplayName,
				"description":  definition.Description,
				"category":     definition.Category,
				"parameters":   definition.Parameters,
				"is_builtin":   true,
				"script_type":  definition.ScriptType,
				"script_body":  "",
				"markdown":     definition.Markdown,
				"risk_level":   definition.RiskLevel,
			}
			if updateErr := db.Model(&existing).Updates(updates).Error; updateErr != nil {
				return fmt.Errorf("更新内置 Skill %s 失败: %w", definition.Name, updateErr)
			}
			continue
		}
		if err != gorm.ErrRecordNotFound {
			return fmt.Errorf("查询内置 Skill %s 失败: %w", definition.Name, err)
		}

		definition.IsEnabled = true
		if createErr := db.Create(&definition).Error; createErr != nil {
			return fmt.Errorf("创建内置 Skill %s 失败: %w", definition.Name, createErr)
		}
	}

	return nil
}
