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
	"gopkg.in/yaml.v3"
)

// ListSkills 获取 Skill 列表
func (h *Handler) ListSkills(c *gin.Context) {
	category := c.Query("category")

	var skills []biz.SkillDefinition
	query := h.db.Model(&biz.SkillDefinition{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if err := query.Order("category ASC, name ASC").Find(&skills).Error; err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取 Skill 列表失败")
		return
	}

	// 同步内置 Skills 到列表
	registeredSkills := h.registry.GetAll()
	registeredMap := make(map[string]bool)
	for _, s := range skills {
		registeredMap[s.Name] = true
	}

	for _, rs := range registeredSkills {
		if !registeredMap[rs.Name()] {
			skills = append(skills, biz.SkillDefinition{
				Name:        rs.Name(),
				DisplayName: rs.Name(),
				Description: rs.Description(),
				IsBuiltin:   true,
				ScriptType:  "builtin",
				IsEnabled:   true,
				RiskLevel:   rs.RiskLevel(),
			})
		}
	}

	response.Success(c, skills)
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

// SkillManifest manifest.yaml 结构
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

// UploadSkill 上传自定义 Skill (.skill.zip)
func (h *Handler) UploadSkill(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "请上传文件")
		return
	}

	// 验证文件扩展名
	if !strings.HasSuffix(file.Filename, ".skill.zip") && !strings.HasSuffix(file.Filename, ".zip") {
		response.ErrorCode(c, http.StatusBadRequest, "请上传 .skill.zip 格式的文件")
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

	var manifest *SkillManifest
	var scriptBody string

	for _, zf := range reader.File {
		name := zf.Name
		// 跳过目录
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

		switch baseName {
		case "manifest.yaml", "manifest.yml":
			var m SkillManifest
			if err := yaml.Unmarshal(content, &m); err != nil {
				response.ErrorCode(c, http.StatusBadRequest, "manifest.yaml 解析失败: "+err.Error())
				return
			}
			manifest = &m

		case "script.js":
			scriptBody = string(content)

		case "script.py":
			scriptBody = string(content)
		}
	}

	if manifest == nil {
		response.ErrorCode(c, http.StatusBadRequest, "缺少 manifest.yaml 文件")
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

	if manifest.ScriptType == "" {
		// 根据文件推断
		for _, zf := range reader.File {
			if strings.HasSuffix(zf.Name, ".js") {
				manifest.ScriptType = "javascript"
				break
			}
			if strings.HasSuffix(zf.Name, ".py") {
				manifest.ScriptType = "python"
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
			"script_type":  manifest.ScriptType,
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
		ScriptType:  manifest.ScriptType,
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
