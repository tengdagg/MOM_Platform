<template>
  <div class="system-config-container">
    <div class="page-header">
      <h2 class="page-title">系统配置</h2>
      <el-button class="black-button" @click="handleSave" :loading="saving">保存配置</el-button>
    </div>

    <el-tabs v-model="activeTab" class="config-tabs">
      <!-- 基础配置 Tab -->
      <el-tab-pane label="基础配置" name="basic">
        <el-card class="config-card">
          <el-form :model="config" label-width="150px">
            <el-form-item label="系统名称">
              <el-input v-model="config.systemName" placeholder="请输入系统名称" />
            </el-form-item>
            <el-form-item label="系统logo">
              <el-input v-model="config.systemLogo" placeholder="请输入系统logo地址" />
            </el-form-item>
            <el-form-item label="系统描述">
              <el-input v-model="config.systemDescription" type="textarea" :rows="3" placeholder="请输入系统描述" />
            </el-form-item>
            <el-form-item label="版权信息">
              <el-input v-model="config.copyright" placeholder="请输入版权信息" />
            </el-form-item>
          </el-form>
        </el-card>

        <el-card class="config-card">
          <template #header><span>安全配置</span></template>
          <el-form :model="config" label-width="150px">
            <el-form-item label="密码最小长度">
              <el-input-number v-model="config.passwordMinLength" :min="6" :max="20" />
            </el-form-item>
            <el-form-item label="Session过期时间">
              <el-input-number v-model="config.sessionTimeout" :min="30" :step="30" />
              <span style="margin-left: 10px; color: #999">秒</span>
            </el-form-item>
            <el-form-item label="开启验证码">
              <el-switch v-model="config.enableCaptcha" />
            </el-form-item>
            <el-form-item label="最大登录失败次数">
              <el-input-number v-model="config.maxLoginAttempts" :min="3" :max="10" />
            </el-form-item>
          </el-form>
        </el-card>

        <el-card class="config-card">
          <template #header><span>通知配置</span></template>
          <el-form :model="config" label-width="150px">
            <el-form-item label="邮件通知">
              <el-switch v-model="config.enableEmailNotification" />
            </el-form-item>
            <el-form-item label="SMTP服务器">
              <el-input v-model="config.smtpHost" placeholder="请输入SMTP服务器地址" />
            </el-form-item>
            <el-form-item label="SMTP端口">
              <el-input-number v-model="config.smtpPort" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="发件人邮箱">
              <el-input v-model="config.smtpFrom" placeholder="请输入发件人邮箱" />
            </el-form-item>
          </el-form>
        </el-card>

        <el-card class="config-card">
          <template #header><span>其他配置</span></template>
          <el-form :model="config" label-width="150px">
            <el-form-item label="开启注册">
              <el-switch v-model="config.enableRegister" />
            </el-form-item>
            <el-form-item label="默认用户角色">
              <el-select v-model="config.defaultUserRole" placeholder="请选择默认角色">
                <el-option label="普通用户" value="user" />
                <el-option label="管理员" value="admin" />
              </el-select>
            </el-form-item>
            <el-form-item label="日志保留天数">
              <el-input-number v-model="config.logRetentionDays" :min="7" :max="365" />
              <span style="margin-left: 10px; color: #999; font-size: 12px;">
                适用于操作日志、登录日志、数据变更日志，超过天数自动清理
              </span>
            </el-form-item>
          </el-form>
        </el-card>
      </el-tab-pane>

      <!-- LDAP 认证配置 Tab -->
      <el-tab-pane label="LDAP 认证" name="ldap">
        <el-card class="config-card">
          <template #header>
            <div class="card-header-row">
              <span>LDAP / Active Directory 认证配置</span>
              <div class="card-header-actions">
                <el-tag v-if="ldapConfig.enabled" type="success" size="small">已启用</el-tag>
                <el-tag v-else type="info" size="small">未启用</el-tag>
              </div>
            </div>
          </template>

          <el-alert type="info" :closable="false" style="margin-bottom: 20px;">
            启用 LDAP 认证后，用户可以使用 LDAP 账号密码登录系统。首次登录时系统将自动创建本地用户并标记为 LDAP 用户，不影响已有本地用户的正常使用。
          </el-alert>

          <el-form :model="ldapConfig" label-width="160px">
            <el-divider content-position="left">基本设置</el-divider>

            <el-form-item label="启用 LDAP 认证">
              <el-switch v-model="ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="服务器地址">
              <el-input v-model="ldapConfig.host" placeholder="如 ldap.example.com 或 192.168.1.100" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="端口">
              <el-input-number v-model="ldapConfig.port" :min="1" :max="65535" :disabled="!ldapConfig.enabled" />
              <span style="margin-left: 10px; color: #999; font-size: 12px;">LDAP 默认 389，LDAPS 默认 636</span>
            </el-form-item>

            <el-form-item label="使用 SSL (LDAPS)">
              <el-switch v-model="ldapConfig.useSsl" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-divider content-position="left">绑定设置</el-divider>

            <el-form-item label="绑定 DN">
              <el-input v-model="ldapConfig.bindDn" placeholder="如 cn=admin,dc=example,dc=com" :disabled="!ldapConfig.enabled" />
              <div class="form-tip">管理员账号的完整 DN，用于搜索用户</div>
            </el-form-item>

            <el-form-item label="绑定密码">
              <el-input v-model="ldapConfig.bindPassword" type="password" show-password :placeholder="ldapConfig.passwordSet ? '已设置（留空则不修改）' : '请输入绑定密码'" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="搜索 Base DN">
              <el-input v-model="ldapConfig.baseDn" placeholder="如 dc=example,dc=com" :disabled="!ldapConfig.enabled" />
              <div class="form-tip">用户搜索的基础 DN，所有用户都应在此 DN 下</div>
            </el-form-item>

            <el-divider content-position="left">搜索过滤器</el-divider>

            <el-form-item label="用户搜索过滤器">
              <el-input v-model="ldapConfig.userFilter" placeholder="(&(objectClass=person)(sAMAccountName=%s))" :disabled="!ldapConfig.enabled" />
              <div class="form-tip">%s 将被替换为用户名。AD 常用 sAMAccountName，OpenLDAP 常用 uid</div>
            </el-form-item>

            <el-divider content-position="left">属性映射</el-divider>

            <el-form-item label="用户名属性">
              <el-input v-model="ldapConfig.attrUsername" placeholder="sAMAccountName" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="姓名属性">
              <el-input v-model="ldapConfig.attrRealName" placeholder="displayName" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="邮箱属性">
              <el-input v-model="ldapConfig.attrEmail" placeholder="mail" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-form-item label="手机号属性">
              <el-input v-model="ldapConfig.attrPhone" placeholder="telephoneNumber" :disabled="!ldapConfig.enabled" />
            </el-form-item>

            <el-divider content-position="left">权限设置</el-divider>

            <el-form-item label="默认角色">
              <el-select v-model="ldapConfig.defaultRoleId" placeholder="LDAP 用户首次登录时分配的角色" :disabled="!ldapConfig.enabled" clearable>
                <el-option v-for="role in roles" :key="role.id" :label="role.name" :value="role.id" />
              </el-select>
              <div class="form-tip">LDAP 用户首次登录自动创建时将分配此角色</div>
            </el-form-item>

            <el-divider />

            <el-form-item>
              <el-button class="black-button" @click="testLDAPConnection" :loading="testingLdap" :disabled="!ldapConfig.enabled">
                测试连接
              </el-button>
              <el-button class="black-button" @click="saveLDAPConfig" :loading="savingLdap">
                保存 LDAP 配置
              </el-button>
              <el-button @click="syncLDAPUsers" :loading="syncingLdap" :disabled="!ldapConfig.enabled">
                同步 LDAP 用户
              </el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <!-- LDAP 用户统计 -->
        <el-card class="config-card" v-if="ldapConfig.enabled">
          <template #header><span>LDAP 用户统计</span></template>
          <div class="ldap-stats">
            <div class="stat-item">
              <span class="stat-label">LDAP 用户总数</span>
              <span class="stat-value">{{ ldapUserCount }}</span>
            </div>
          </div>
        </el-card>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '@/utils/request'
import { getAllRoles } from '@/api/role'

const saving = ref(false)
const activeTab = ref('basic')

const config = reactive({
  systemName: 'mom',
  systemLogo: '',
  systemDescription: '运维管理平台',
  copyright: '© 2025 mom. All rights reserved.',
  passwordMinLength: 6,
  sessionTimeout: 3600,
  enableCaptcha: true,
  maxLoginAttempts: 5,
  enableEmailNotification: false,
  smtpHost: '',
  smtpPort: 587,
  smtpFrom: '',
  enableRegister: false,
  defaultUserRole: 'user',
  logRetentionDays: 30
})

// LDAP 配置
const ldapConfig = reactive({
  enabled: false,
  host: '',
  port: 389,
  useSsl: false,
  bindDn: '',
  bindPassword: '',
  passwordSet: false,
  baseDn: '',
  userFilter: '(&(objectClass=person)(sAMAccountName=%s))',
  attrUsername: 'sAMAccountName',
  attrRealName: 'displayName',
  attrEmail: 'mail',
  attrPhone: 'telephoneNumber',
  defaultRoleId: 0 as number | undefined,
})

const roles = ref<any[]>([])
const testingLdap = ref(false)
const savingLdap = ref(false)
const syncingLdap = ref(false)
const ldapUserCount = ref(0)

// 加载角色列表
async function loadRoles() {
  try {
    const data: any = await getAllRoles()
    roles.value = Array.isArray(data) ? data : (data?.list || [])
  } catch { /* ignore */ }
}

// 加载 LDAP 配置
async function loadLDAPConfig() {
  try {
    const data: any = await request.get('/api/v1/ldap/config')
    if (data) {
      Object.assign(ldapConfig, {
        enabled: data.enabled || false,
        host: data.host || '',
        port: data.port || 389,
        useSsl: data.useSsl || false,
        bindDn: data.bindDn || '',
        bindPassword: '',
        passwordSet: data.passwordSet || false,
        baseDn: data.baseDn || '',
        userFilter: data.userFilter || '(&(objectClass=person)(sAMAccountName=%s))',
        attrUsername: data.attrUsername || 'sAMAccountName',
        attrRealName: data.attrRealName || 'displayName',
        attrEmail: data.attrEmail || 'mail',
        attrPhone: data.attrPhone || 'telephoneNumber',
        defaultRoleId: data.defaultRoleId || undefined,
      })
    }
  } catch { /* ignore */ }
}

// 加载 LDAP 用户数
async function loadLDAPUserCount() {
  try {
    const data: any = await request.get('/api/v1/ldap/users', { params: { page: 1, pageSize: 1 } })
    ldapUserCount.value = data?.total || 0
  } catch { /* ignore */ }
}

// 保存 LDAP 配置
async function saveLDAPConfig() {
  savingLdap.value = true
  try {
    await request.put('/api/v1/ldap/config', ldapConfig)
    ElMessage.success('LDAP 配置保存成功')
    ldapConfig.passwordSet = true
  } catch {
    ElMessage.error('保存失败')
  }
  savingLdap.value = false
}

// 测试 LDAP 连接
async function testLDAPConnection() {
  testingLdap.value = true
  try {
    await request.post('/api/v1/ldap/test', ldapConfig)
    ElMessage.success('LDAP 连接测试成功')
  } catch (e: any) {
    ElMessage.error(e?.message || 'LDAP 连接测试失败')
  }
  testingLdap.value = false
}

// 同步 LDAP 用户
async function syncLDAPUsers() {
  syncingLdap.value = true
  try {
    const data: any = await request.post('/api/v1/ldap/sync')
    ElMessage.success(`同步完成：发现 ${data?.totalFound || 0} 个用户，新创建 ${data?.newCreated || 0} 个`)
    await loadLDAPUserCount()
  } catch (e: any) {
    ElMessage.error(e?.message || '同步失败')
  }
  syncingLdap.value = false
}

const loadConfig = async () => {
  try {
    const data: any = await request.get('/api/v1/system-config')
    if (data && typeof data === 'object') {
      // 将后端字符串值转为正确的类型赋值
      if (data.systemName !== undefined) config.systemName = data.systemName
      if (data.systemLogo !== undefined) config.systemLogo = data.systemLogo
      if (data.systemDescription !== undefined) config.systemDescription = data.systemDescription
      if (data.copyright !== undefined) config.copyright = data.copyright
      if (data.passwordMinLength !== undefined) config.passwordMinLength = parseInt(data.passwordMinLength) || 6
      if (data.sessionTimeout !== undefined) config.sessionTimeout = parseInt(data.sessionTimeout) || 3600
      if (data.enableCaptcha !== undefined) config.enableCaptcha = data.enableCaptcha === 'true' || data.enableCaptcha === true
      if (data.maxLoginAttempts !== undefined) config.maxLoginAttempts = parseInt(data.maxLoginAttempts) || 5
      if (data.enableEmailNotification !== undefined) config.enableEmailNotification = data.enableEmailNotification === 'true' || data.enableEmailNotification === true
      if (data.smtpHost !== undefined) config.smtpHost = data.smtpHost
      if (data.smtpPort !== undefined) config.smtpPort = parseInt(data.smtpPort) || 587
      if (data.smtpFrom !== undefined) config.smtpFrom = data.smtpFrom
      if (data.enableRegister !== undefined) config.enableRegister = data.enableRegister === 'true' || data.enableRegister === true
      if (data.defaultUserRole !== undefined) config.defaultUserRole = data.defaultUserRole
      if (data.logRetentionDays !== undefined) config.logRetentionDays = parseInt(data.logRetentionDays) || 30
    }
  } catch {
    // 后端未返回数据时使用默认值
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    await request.put('/api/v1/system-config', config)
    ElMessage.success('保存成功')
  } catch {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  loadConfig()
  await loadRoles()
  await loadLDAPConfig()
  await loadLDAPUserCount()
})
</script>

<style scoped>
.system-config-container {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-title {
  font-size: 20px;
  font-weight: 600;
  color: #333;
  margin: 0;
}

.config-card {
  margin-bottom: 20px;
}

.config-tabs :deep(.el-tabs__header) {
  margin-bottom: 16px;
}

.card-header-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header-actions {
  display: flex;
  gap: 8px;
}

.form-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}

.ldap-stats {
  display: flex;
  gap: 40px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  font-size: 13px;
  color: #909399;
}

.stat-value {
  font-size: 24px;
  font-weight: 600;
  color: #0a466a;
}
</style>
