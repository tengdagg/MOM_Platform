<template>
  <div class="system-config-container">
    <div class="page-header">
      <h2 class="page-title">系统配置</h2>
      <el-button class="black-button" @click="handleSave" :loading="saving">保存配置</el-button>
    </div>

    <el-card class="config-card">
      <template #header>
        <span>基础配置</span>
      </template>

      <el-form :model="config" label-width="150px">
        <el-form-item label="系统名称">
          <el-input v-model="config.systemName" placeholder="请输入系统名称" />
        </el-form-item>
        <el-form-item label="系统logo">
          <el-input v-model="config.systemLogo" placeholder="请输入系统logo地址" />
        </el-form-item>
        <el-form-item label="系统描述">
          <el-input
            v-model="config.systemDescription"
            type="textarea"
            :rows="3"
            placeholder="请输入系统描述"
          />
        </el-form-item>
        <el-form-item label="版权信息">
          <el-input v-model="config.copyright" placeholder="请输入版权信息" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="config-card">
      <template #header>
        <span>安全配置</span>
      </template>

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
      <template #header>
        <span>通知配置</span>
      </template>

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
      <template #header>
        <span>其他配置</span>
      </template>

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
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'

const saving = ref(false)

const config = reactive({
  // 基础配置
  systemName: 'iom',
  systemLogo: '',
  systemDescription: '运维管理平台',
  copyright: '© 2025 iom. All rights reserved.',

  // 安全配置
  passwordMinLength: 6,
  sessionTimeout: 3600,
  enableCaptcha: true,
  maxLoginAttempts: 5,

  // 通知配置
  enableEmailNotification: false,
  smtpHost: '',
  smtpPort: 587,
  smtpFrom: '',

  // 其他配置
  enableRegister: false,
  defaultUserRole: 'user',
  logRetentionDays: 30
})

const loadConfig = async () => {
  try {
    // TODO: 从后端加载配置
    const savedConfig = localStorage.getItem('system_config')
    if (savedConfig) {
      Object.assign(config, JSON.parse(savedConfig))
    }
  } catch (error) {
  }
}

const handleSave = async () => {
  saving.value = true
  try {
    // TODO: 调用后端API保存配置
    localStorage.setItem('system_config', JSON.stringify(config))
    await new Promise(resolve => setTimeout(resolve, 500))
    ElMessage.success('保存成功')
  } catch (error) {
    ElMessage.error('保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
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

.black-button {
  background-color: #000;
  color: #fff;
  border: none;
}

.black-button:hover {
  background-color: #333;
  color: #fff;
}
</style>
