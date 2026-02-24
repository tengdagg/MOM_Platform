package rbac

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-ldap/ldap/v3"
	"github.com/ydcloud-dy/mom/internal/biz/rbac"
	"github.com/ydcloud-dy/mom/pkg/response"
	"gorm.io/gorm"
)

// LDAPService LDAP 认证服务
type LDAPService struct {
	db *gorm.DB
}

// NewLDAPService 创建 LDAP 服务
func NewLDAPService(db *gorm.DB) *LDAPService {
	return &LDAPService{db: db}
}

// GetConfig 获取 LDAP 配置
func (s *LDAPService) GetConfig() (*rbac.SysLDAPConfig, error) {
	var config rbac.SysLDAPConfig
	err := s.db.First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 返回默认配置
			return &rbac.SysLDAPConfig{
				Port:         389,
				UserFilter:   "(&(objectClass=person)(sAMAccountName=%s))",
				AttrUsername: "sAMAccountName",
				AttrRealName: "displayName",
				AttrEmail:    "mail",
				AttrPhone:    "telephoneNumber",
			}, nil
		}
		return nil, err
	}
	config.PasswordSet = config.BindPassword != ""
	return &config, nil
}

// SaveConfig 保存 LDAP 配置
func (s *LDAPService) SaveConfig(config *rbac.SysLDAPConfig) error {
	var existing rbac.SysLDAPConfig
	err := s.db.First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 新建
		return s.db.Create(config).Error
	}
	if err != nil {
		return err
	}

	// 更新（保留原密码如果新密码为空）
	config.ID = existing.ID
	if config.BindPassword == "" {
		config.BindPassword = existing.BindPassword
	}
	return s.db.Save(config).Error
}

// Authenticate LDAP 认证 — 返回用户属性映射
func (s *LDAPService) Authenticate(username, password string) (map[string]string, error) {
	config, err := s.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("获取 LDAP 配置失败: %w", err)
	}
	if !config.Enabled {
		return nil, fmt.Errorf("LDAP 认证未启用")
	}

	// 连接 LDAP
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	var conn *ldap.Conn

	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", addr))
	}
	if err != nil {
		return nil, fmt.Errorf("连接 LDAP 服务器失败: %w", err)
	}
	defer conn.Close()

	// 管理员绑定
	if config.BindDN != "" {
		if err := conn.Bind(config.BindDN, config.BindPassword); err != nil {
			return nil, fmt.Errorf("LDAP 管理员绑定失败: %w", err)
		}
	}

	// 搜索用户
	filter := fmt.Sprintf(config.UserFilter, ldap.EscapeFilter(username))
	searchReq := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 30, false,
		filter,
		[]string{config.AttrUsername, config.AttrRealName, config.AttrEmail, config.AttrPhone, "dn"},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("LDAP 搜索用户失败: %w", err)
	}
	if len(result.Entries) == 0 {
		return nil, fmt.Errorf("LDAP 用户不存在")
	}

	entry := result.Entries[0]
	userDN := entry.DN

	// 用户绑定验证密码
	if err := conn.Bind(userDN, password); err != nil {
		return nil, fmt.Errorf("LDAP 密码验证失败")
	}

	// 提取属性
	attrs := map[string]string{
		"username": entry.GetAttributeValue(config.AttrUsername),
		"realName": entry.GetAttributeValue(config.AttrRealName),
		"email":    entry.GetAttributeValue(config.AttrEmail),
		"phone":    entry.GetAttributeValue(config.AttrPhone),
	}
	if attrs["username"] == "" {
		attrs["username"] = username
	}

	return attrs, nil
}

// TestConnection 测试 LDAP 连接
func (s *LDAPService) TestConnection(config *rbac.SysLDAPConfig) error {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	var conn *ldap.Conn
	var err error

	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", addr))
	}
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	defer conn.Close()

	// 如果没有提供密码，从数据库获取
	bindPassword := config.BindPassword
	if bindPassword == "" {
		var existing rbac.SysLDAPConfig
		if err := s.db.First(&existing).Error; err == nil {
			bindPassword = existing.BindPassword
		}
	}

	if config.BindDN != "" {
		if err := conn.Bind(config.BindDN, bindPassword); err != nil {
			return fmt.Errorf("绑定失败: %w", err)
		}
	}

	// 使用实际的用户过滤器进行搜索测试（用 * 替代用户名）
	filter := strings.Replace(config.UserFilter, "%s", "*", 1)
	if filter == "" {
		filter = "(objectClass=person)"
	}
	searchReq := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 1, 10, false,
		filter,
		[]string{"dn"},
		nil,
	)
	_, err = conn.Search(searchReq)
	if err != nil {
		// SizeLimitExceeded (code 4) 表示搜索有结果但超出限制，连接本身正常
		if ldapErr, ok := err.(*ldap.Error); ok && ldapErr.ResultCode == ldap.LDAPResultSizeLimitExceeded {
			return nil
		}
		return fmt.Errorf("搜索失败: %w", err)
	}

	return nil
}

// IsEnabled 检查 LDAP 是否启用
func (s *LDAPService) IsEnabled() bool {
	config, err := s.GetConfig()
	if err != nil {
		return false
	}
	return config.Enabled
}

// GetDefaultRoleID 获取默认角色 ID
func (s *LDAPService) GetDefaultRoleID() uint {
	config, err := s.GetConfig()
	if err != nil {
		return 0
	}
	return config.DefaultRoleID
}

// ---- HTTP Handlers ----

// GetLDAPConfig 获取 LDAP 配置
func (s *LDAPService) GetLDAPConfig(c *gin.Context) {
	config, err := s.GetConfig()
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "获取 LDAP 配置失败")
		return
	}
	// 不返回密码明文
	config.BindPassword = ""
	response.Success(c, config)
}

// SaveLDAPConfig 保存 LDAP 配置
func (s *LDAPService) SaveLDAPConfig(c *gin.Context) {
	var config rbac.SysLDAPConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}
	if err := s.SaveConfig(&config); err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "保存失败: "+err.Error())
		return
	}
	log.Printf("[LDAP] 配置已保存, enabled=%v, host=%s:%d", config.Enabled, config.Host, config.Port)
	response.Success(c, gin.H{"message": "保存成功"})
}

// TestLDAPConnection 测试 LDAP 连接
func (s *LDAPService) TestLDAPConnection(c *gin.Context) {
	var config rbac.SysLDAPConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.ErrorCode(c, http.StatusBadRequest, "参数错误")
		return
	}
	if err := s.TestConnection(&config); err != nil {
		response.ErrorCode(c, http.StatusOK, "连接测试失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "连接成功"})
}

// GetLDAPStatus 获取 LDAP 启用状态（公开接口，登录页使用）
func (s *LDAPService) GetLDAPStatus(c *gin.Context) {
	enabled := s.IsEnabled()
	response.Success(c, gin.H{"enabled": enabled})
}

// SyncLDAPUsers 同步 LDAP 用户列表（手动触发）
func (s *LDAPService) SyncLDAPUsers(c *gin.Context) {
	config, err := s.GetConfig()
	if err != nil || !config.Enabled {
		response.ErrorCode(c, http.StatusBadRequest, "LDAP 未启用")
		return
	}

	// 连接
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	var conn *ldap.Conn
	if config.UseSSL {
		conn, err = ldap.DialTLS("tcp", addr, &tls.Config{InsecureSkipVerify: true})
	} else {
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", addr))
	}
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "连接 LDAP 失败")
		return
	}
	defer conn.Close()

	if config.BindDN != "" {
		if err := conn.Bind(config.BindDN, config.BindPassword); err != nil {
			response.ErrorCode(c, http.StatusInternalServerError, "LDAP 绑定失败")
			return
		}
	}

	// 搜索所有用户
	filter := strings.Replace(config.UserFilter, "%s", "*", 1)
	searchReq := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 60, false,
		filter,
		[]string{config.AttrUsername, config.AttrRealName, config.AttrEmail, config.AttrPhone},
		nil,
	)

	result, err := conn.Search(searchReq)
	if err != nil {
		response.ErrorCode(c, http.StatusInternalServerError, "LDAP 搜索失败: "+err.Error())
		return
	}

	synced := 0
	for _, entry := range result.Entries {
		username := entry.GetAttributeValue(config.AttrUsername)
		if username == "" {
			continue
		}

		// 检查是否已存在
		var existingUser rbac.SysUser
		if s.db.Where("username = ?", username).First(&existingUser).Error == nil {
			// 已存在 → 更新 LDAP 信息
			if existingUser.Source == "ldap" {
				updates := map[string]interface{}{}
				if rn := entry.GetAttributeValue(config.AttrRealName); rn != "" {
					updates["real_name"] = rn
				}
				if email := entry.GetAttributeValue(config.AttrEmail); email != "" {
					updates["email"] = email
				}
				if phone := entry.GetAttributeValue(config.AttrPhone); phone != "" {
					updates["phone"] = phone
				}
				if len(updates) > 0 {
					s.db.Model(&existingUser).Updates(updates)
				}
			}
			continue
		}

		// 创建新用户
		newUser := &rbac.SysUser{
			Username: username,
			Password: "$ldap$", // 占位符，LDAP 用户不使用本地密码
			RealName: entry.GetAttributeValue(config.AttrRealName),
			Email:    entry.GetAttributeValue(config.AttrEmail),
			Phone:    entry.GetAttributeValue(config.AttrPhone),
			Source:   "ldap",
			Status:   1,
		}
		if err := s.db.Create(newUser).Error; err == nil {
			synced++
			// 分配默认角色
			if config.DefaultRoleID > 0 {
				s.db.Create(&rbac.SysUserRole{UserID: newUser.ID, RoleID: config.DefaultRoleID})
			}
		}
	}

	response.Success(c, gin.H{
		"message":    "同步完成",
		"totalFound": len(result.Entries),
		"newCreated": synced,
	})
}

// GetLDAPUsers 查询 LDAP 用户数
func (s *LDAPService) GetLDAPUsers(c *gin.Context) {
	var count int64
	s.db.Model(&rbac.SysUser{}).Where("source = 'ldap'").Count(&count)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	var users []rbac.SysUser
	s.db.Where("source = 'ldap'").Offset((page - 1) * pageSize).Limit(pageSize).Find(&users)

	response.Success(c, gin.H{
		"total": count,
		"list":  users,
	})
}
