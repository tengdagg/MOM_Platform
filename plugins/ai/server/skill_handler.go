package server

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ydcloud-dy/mom/pkg/response"
	"github.com/ydcloud-dy/mom/plugins/ai/biz"
	"github.com/ydcloud-dy/mom/plugins/ai/skills"
	"gopkg.in/yaml.v3"
)

// ListSkills 获取 Skill 列表
func (h *Handler) ListSkills(c *gin.Context) {
	category := c.Query("category")

	var dbSkills []biz.SkillDefinition
	query := h.db.Model(&biz.SkillDefinition{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Order("category ASC, name ASC").Find(&dbSkills).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取 Skill 列表失败")
		return
	}

	// 同步内置 Skills 到列表（也需要按分类过滤）
	registeredSkills := h.registry.GetAll()
	registeredMap := make(map[string]bool)
	for _, s := range dbSkills {
		registeredMap[s.Name] = true
	}

	for _, rs := range registeredSkills {
		if registeredMap[rs.Name()] {
			continue
		}

		skillDef := biz.SkillDefinition{
			Name:        rs.Name(),
			DisplayName: rs.Name(),
			Description: rs.Description(),
			IsBuiltin:   true,
			ScriptType:  "builtin",
			IsEnabled:   true,
			RiskLevel:   rs.RiskLevel(),
		}
		// 提取 BuiltinSkill 的额外信息
		if bs, ok := rs.(*skills.BuiltinSkill); ok {
			skillDef.Category = bs.Category()
			skillDef.Markdown = bs.Markdown()
		}

		// 如果指定了分类过滤，只添加匹配分类的内置 Skill
		if category != "" && skillDef.Category != category {
			continue
		}

		dbSkills = append(dbSkills, skillDef)
	}

	response.Success(c, dbSkills)
}

// ToggleSkill 启用/禁用 Skill
func (h *Handler) ToggleSkill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	var skill biz.SkillDefinition
	if err := h.db.First(&skill, id).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "Skill 不存在")
		return
	}

	skill.IsEnabled = !skill.IsEnabled
	h.db.Model(&skill).Update("is_enabled", skill.IsEnabled)

	response.Success(c, skill)
}

// SkillManifest 传统 manifest.yaml 结构（向后兼容）
type SkillManifest struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"displayName"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`
	Version     string `yaml:"version"`
	Author      string `yaml:"author"`
	RiskLevel   string `yaml:"riskLevel"`
	ScriptType  string `yaml:"scriptType"` // javascript / python
	Parameters  any    `yaml:"parameters"`
}

// UploadSkill 上传自定义 Skill
// 支持两种格式:
//  1. 标准 SKILL.md 格式 (.zip):
//     skill-name/
//     ├── SKILL.md          (必需) YAML 前置元数据 + Markdown 指令
//     └── scripts/
//     ├── script.js 或 script.py
//  2. 传统 manifest.yaml 格式 (.skill.zip):
//     ├── manifest.yaml
//     ├── script.js / script.py
func (h *Handler) UploadSkill(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请上传文件")
		return
	}

	// 验证文件扩展名
	if !strings.HasSuffix(file.Filename, ".zip") {
		response.ErrorCode(c, http.StatusBadRequest, "请上传 .zip 格式的文件")
		return
	}

	// 限制文件大小 (5MB)
	if file.Size > 5*1024*1024 {
		response.ErrorCode(c, http.StatusBadRequest, "文件大小不能超过 5MB")
		return
	}

	// 读取文件内容
	f, err := file.Open()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取文件失败")
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "读取文件失败")
		return
	}

	// 解压 ZIP
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ZIP 文件")
		return
	}

	// 收集文件内容
	var skillMDContent []byte
	var manifestContent []byte
	var scriptBody string
	var scriptType string

	for _, zf := range reader.File {
		name := zf.Name
		if zf.FileInfo().IsDir() {
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}

		baseName := name
		if idx := strings.LastIndex(name, "/"); idx >= 0 {
			baseName = name[idx+1:]
		}

		switch {
		case baseName == "SKILL.md":
			skillMDContent = content
		case baseName == "manifest.yaml" || baseName == "manifest.yml":
			manifestContent = content
		case baseName == "script.js" || strings.HasSuffix(name, "/scripts/script.js"):
			scriptBody = string(content)
			scriptType = "javascript"
		case baseName == "script.py" || strings.HasSuffix(name, "/scripts/script.py"):
			scriptBody = string(content)
			scriptType = "python"
		}
	}

	// 优先使用 SKILL.md 格式
	if skillMDContent != nil {
		h.uploadFromSKILLMD(c, skillMDContent, scriptBody, scriptType)
		return
	}

	// 回退到传统 manifest.yaml 格式
	if manifestContent != nil {
		h.uploadFromManifest(c, manifestContent, scriptBody, scriptType, reader)
		return
	}

	response.ErrorCode(c, http.StatusBadRequest, "缺少 SKILL.md 或 manifest.yaml 文件")
}

// uploadFromSKILLMD 使用 SKILL.md 格式上传
func (h *Handler) uploadFromSKILLMD(c *gin.Context, content []byte, scriptBody string, scriptType string) {
	meta, markdown, err := skills.ParseSKILLMD(content)
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "SKILL.md 解析失败: "+err.Error())
		return
	}

	if scriptBody == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 scripts/script.js 或 scripts/script.py 执行脚本")
		return
	}

	if scriptType == "" && meta.ScriptType != "" && meta.ScriptType != "builtin" {
		scriptType = meta.ScriptType
	}
	if scriptType == "" {
		response.ErrorCode(c, http.StatusBadRequest, "无法识别脚本类型，请在 SKILL.md 中指定 scriptType 或提供 script.js/script.py")
		return
	}

	if meta.RiskLevel == "" {
		meta.RiskLevel = "low"
	}

	// 序列化参数
	var paramsJSON string
	if meta.Parameters != nil {
		paramsBytes, _ := json.Marshal(meta.Parameters)
		paramsJSON = string(paramsBytes)
	} else {
		paramsJSON = `{"type":"object","properties":{}}`
	}

	// 检查是否已存在
	var existing biz.SkillDefinition
	if err := h.db.Where("name = ?", meta.Name).First(&existing).Error; err == nil {
		// 更新现有
		h.db.Model(&existing).Updates(map[string]interface{}{
			"display_name": meta.Name,
			"description":  meta.Description,
			"category":     meta.Category,
			"parameters":   paramsJSON,
			"script_type":  scriptType,
			"script_body":  scriptBody,
			"risk_level":   meta.RiskLevel,
			"markdown":     markdown,
		})
		response.SuccessWithMessage(c, fmt.Sprintf("Skill %s 已更新", meta.Name), existing)
		return
	}

	// 创建新 Skill
	skill := biz.SkillDefinition{
		Name:        meta.Name,
		DisplayName: meta.Name,
		Description: meta.Description,
		Category:    meta.Category,
		Parameters:  paramsJSON,
		IsBuiltin:   false,
		ScriptType:  scriptType,
		ScriptBody:  scriptBody,
		IsEnabled:   true,
		RiskLevel:   meta.RiskLevel,
		Markdown:    markdown,
	}

	if err := h.db.Create(&skill).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存 Skill 失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, fmt.Sprintf("Skill %s 上传成功", meta.Name), skill)
}

// uploadFromManifest 使用传统 manifest.yaml 格式上传（向后兼容）
func (h *Handler) uploadFromManifest(c *gin.Context, manifestContent []byte, scriptBody string, scriptType string, reader *zip.Reader) {
	var manifest SkillManifest
	if err := yaml.Unmarshal(manifestContent, &manifest); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "manifest.yaml 解析失败: "+err.Error())
		return
	}

	if manifest.Name == "" {
		response.ErrorCode(c, http.StatusBadRequest, "manifest.yaml 中缺少 name 字段")
		return
	}

	if scriptBody == "" {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 script.js 或 script.py 文件")
		return
	}

	if scriptType == "" && manifest.ScriptType != "" {
		scriptType = manifest.ScriptType
	}
	if scriptType == "" {
		// 根据文件推断
		for _, zf := range reader.File {
			if strings.HasSuffix(zf.Name, ".js") {
				scriptType = "javascript"
				break
			}
			if strings.HasSuffix(zf.Name, ".py") {
				scriptType = "python"
				break
			}
		}
	}

	if manifest.RiskLevel == "" {
		manifest.RiskLevel = "low"
	}

	// 序列化参数
	var paramsJSON string
	if manifest.Parameters != nil {
		paramsBytes, _ := json.Marshal(manifest.Parameters)
		paramsJSON = string(paramsBytes)
	} else {
		paramsJSON = `{"type":"object","properties":{}}`
	}

	// 检查是否已存在
	var existing biz.SkillDefinition
	if err := h.db.Where("name = ?", manifest.Name).First(&existing).Error; err == nil {
		// 更新现有
		h.db.Model(&existing).Updates(map[string]interface{}{
			"display_name": manifest.DisplayName,
			"description":  manifest.Description,
			"category":     manifest.Category,
			"parameters":   paramsJSON,
			"script_type":  scriptType,
			"script_body":  scriptBody,
			"risk_level":   manifest.RiskLevel,
		})
		response.SuccessWithMessage(c, fmt.Sprintf("Skill %s 已更新", manifest.Name), existing)
		return
	}

	// 创建新 Skill
	skill := biz.SkillDefinition{
		Name:        manifest.Name,
		DisplayName: manifest.DisplayName,
		Description: manifest.Description,
		Category:    manifest.Category,
		Parameters:  paramsJSON,
		IsBuiltin:   false,
		ScriptType:  scriptType,
		ScriptBody:  scriptBody,
		IsEnabled:   true,
		RiskLevel:   manifest.RiskLevel,
	}

	if err := h.db.Create(&skill).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存 Skill 失败: "+err.Error())
		return
	}

	response.SuccessWithMessage(c, fmt.Sprintf("Skill %s 上传成功", manifest.Name), skill)
}

// DeleteSkill 删除自定义 Skill
func (h *Handler) DeleteSkill(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if id == 0 {
		response.ErrorCode(c, http.StatusBadRequest, "无效的 ID")
		return
	}

	var skill biz.SkillDefinition
	if err := h.db.First(&skill, id).Error; err != nil {
		response.ErrorCode(c, http.StatusNotFound, "Skill 不存在")
		return
	}

	if skill.IsBuiltin {
		response.ErrorCode(c, http.StatusBadRequest, "内置 Skill 不可删除")
		return
	}

	h.db.Delete(&skill)
	response.Success(c, nil)
}
