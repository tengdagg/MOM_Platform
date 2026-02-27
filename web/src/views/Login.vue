<template>
  <div class="login-container">
    <!-- 左侧：品牌展示区 -->
    <div class="brand-section">
      <div class="curved-divider"></div>
      <div class="brand-content">
        <h1 class="brand-title">MOM Platform</h1>
        <div class="brand-slogan">
          <span>插件化</span>
          <span>多集群</span>
          <span>一站式</span>
          <span>AI集成</span>
        </div>
        <p class="brand-subtitle">大模型驱动现代化云原生运维管理专家</p>
        <div class="brand-illustration">
          <svg viewBox="0 0 400 300" class="illustration-svg">
            <defs>
              <linearGradient id="blueGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style="stop-color:#0f69a6;stop-opacity:1" />
                <stop offset="100%" style="stop-color:#00b4db;stop-opacity:1" />
              </linearGradient>
              <filter id="glow" x="-20%" y="-20%" width="140%" height="140%">
                <feGaussianBlur stdDeviation="3" result="blur" />
                <feComposite in="SourceGraphic" in2="blur" operator="over" />
              </filter>
            </defs>

            <!-- 底部基座 (Cloud Foundation) -->
            <ellipse cx="200" cy="230" rx="120" ry="25" fill="url(#blueGradient)" opacity="0.1" />
            <ellipse cx="200" cy="220" rx="100" ry="20" fill="url(#blueGradient)" opacity="0.2" />
            
            <!-- 中心控制塔 (MOM Core) -->
            <path d="M200,210 L240,190 L240,150 L200,130 L160,150 L160,190 Z" fill="url(#blueGradient)" opacity="0.8" filter="url(#glow)" />
            <path d="M200,130 L200,170 M200,170 L240,190 M200,170 L160,190" stroke="rgba(255,255,255,0.3)" stroke-width="1" />
            
            <!-- 顶部全息投影 (Dashboard) -->
            <path d="M170,110 L230,110 L240,90 L160,90 Z" fill="url(#blueGradient)" opacity="0.4" />
            <rect x="180" y="95" width="40" height="2" rx="1" fill="#fff" opacity="0.6" />
            
            <!-- 连接线 (Connectivity) -->
            <line x1="200" y1="130" x2="120" y2="80" stroke="url(#blueGradient)" stroke-width="2" opacity="0.4" stroke-dasharray="4,4" />
            <line x1="240" y1="150" x2="300" y2="100" stroke="url(#blueGradient)" stroke-width="2" opacity="0.4" stroke-dasharray="4,4" />
            <line x1="160" y1="150" x2="100" y2="180" stroke="url(#blueGradient)" stroke-width="2" opacity="0.4" stroke-dasharray="4,4" />

            <!-- 分布式节点 (Nodes/Clusters) -->
            <g transform="translate(120, 80)">
               <circle r="12" fill="url(#blueGradient)" opacity="0.7" />
               <path d="M-6,-2 L0,6 L6,-2" stroke="#fff" stroke-width="2" fill="none" transform="scale(0.6)" />
            </g>
            
            <g transform="translate(300, 100)">
               <rect x="-10" y="-10" width="20" height="20" rx="4" fill="url(#blueGradient)" opacity="0.7" />
               <circle r="4" fill="#fff" opacity="0.6" />
            </g>
            
            <g transform="translate(100, 180)">
               <polygon points="0,-12 10,6 -10,6" fill="url(#blueGradient)" opacity="0.7" />
            </g>

            <!-- 数据粒子 (Data Flow) -->
            <circle cx="160" cy="105" r="2" fill="#00b4db" opacity="0.9" />
            <circle cx="270" cy="125" r="2" fill="#00b4db" opacity="0.9" />
            <circle cx="130" cy="165" r="2" fill="#00b4db" opacity="0.9" />
          </svg>
        </div>
      </div>
    </div>

    <!-- 右侧：登录表单区 -->
    <div class="login-section">
      <div class="login-wrapper">
        <div class="login-header">
          <h2>用户登录</h2>
          <div class="header-line"></div>
          <p v-if="ldapEnabled" class="ldap-hint">支持 LDAP / AD 域账号登录</p>
        </div>

        <el-form :model="loginForm" :rules="rules" ref="formRef" class="login-form" size="default">
          <el-form-item prop="username">
            <el-input
              v-model="loginForm.username"
              placeholder="请输入用户名"
              :prefix-icon="User"
            />
          </el-form-item>

          <el-form-item prop="password">
            <el-input
              v-model="loginForm.password"
              type="password"
              placeholder="请输入密码"
              show-password
              :prefix-icon="Lock"
              @keyup.enter="handleLogin"
            />
          </el-form-item>

          <el-form-item prop="captchaCode">
            <div class="captcha-wrapper">
              <el-input
                v-model="loginForm.captchaCode"
                placeholder="请输入验证码"
                :prefix-icon="Key"
                @keyup.enter="handleLogin"
              />
              <div class="captcha-image" @click="refreshCaptcha">
                <img v-if="captchaImage" :src="captchaImage" alt="验证码" />
                <span v-else class="captcha-loading">加载中...</span>
              </div>
            </div>
          </el-form-item>

          <el-form-item>
            <div class="form-options">
              <el-checkbox v-model="loginForm.remember">记住登录名</el-checkbox>
            </div>
          </el-form-item>

          <el-form-item>
            <el-button
              type="primary"
              @click="handleLogin"
              :loading="loading"
              class="login-button"
            >
              登录
            </el-button>
          </el-form-item>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, FormInstance } from 'element-plus'
import { User, Lock, Key } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import request from '@/utils/request'

const router = useRouter()
const userStore = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const captchaImage = ref('')
const captchaId = ref('')

const ldapEnabled = ref(false)

const loginForm = reactive({
  username: '',
  password: '',
  captchaCode: '',
  captchaId: '',
  remember: false
})

const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  captchaCode: [{ required: true, message: '请输入验证码', trigger: 'blur' }]
}

// 获取验证码
const refreshCaptcha = async () => {
  try {
    captchaImage.value = ''
    const res: any = await request.get('/api/v1/captcha')
    captchaImage.value = res.image
    captchaId.value = res.captchaId
    loginForm.captchaId = res.captchaId
  } catch (error) {
    ElMessage.error('获取验证码失败')
  }
}

const handleLogin = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (valid) {
      loading.value = true
      try {
        await userStore.login({
          username: loginForm.username,
          password: loginForm.password,
          captchaId: loginForm.captchaId,
          captchaCode: loginForm.captchaCode
        })

        // 如果选择了记住登录名，保存到本地
        if (loginForm.remember) {
          localStorage.setItem('rememberedUsername', loginForm.username)
        } else {
          localStorage.removeItem('rememberedUsername')
        }

        ElMessage.success('登录成功')
        await router.push('/')
      } catch (error: any) {

        // 提取错误消息 - 支持多种错误对象格式
        let errorMessage = '登录失败'
        if (error) {
          // 优先使用message字段
          if (error.message && typeof error.message === 'string' && error.message !== '400') {
            errorMessage = error.message
          }
          // 其次尝试response.data.message
          else if (error.response && error.response.data && error.response.data.message) {
            errorMessage = error.response.data.message
          }
          // 如果message字段是"400"等状态码字符串，尝试其他字段
          else if (error.response && error.response.data && error.response.data.data) {
            errorMessage = error.response.data.data
          }
        }

        ElMessage.error(errorMessage)
        // 登录失败后刷新验证码
        refreshCaptcha()
        loginForm.captchaCode = ''
      } finally {
        loading.value = false
      }
    }
  })
}

// 检查 LDAP 状态
const checkLDAPStatus = async () => {
  try {
    const res: any = await request.get('/api/v1/public/ldap/status')
    ldapEnabled.value = res?.enabled || false
  } catch {
    ldapEnabled.value = false
  }
}

onMounted(() => {
  // 加载记住的用户名
  const rememberedUsername = localStorage.getItem('rememberedUsername')
  if (rememberedUsername) {
    loginForm.username = rememberedUsername
    loginForm.remember = true
  }

  // 加载验证码
  refreshCaptcha()
  // 检查 LDAP 状态
  checkLDAPStatus()
})
</script>

<style scoped>
.login-container {
  display: flex;
  min-height: 100vh;
  background: #ffffff;
  position: relative;
  overflow: hidden;
}

/* 左侧品牌区 - 黑白风格 */
.brand-section {
  flex: 0 0 62%;
  background: linear-gradient(135deg, #001529 0%, #003a5c 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

/* 微妙的灰色渐变背景 */
.brand-section::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background:
    radial-gradient(circle at 20% 30%, rgba(255, 255, 255, 0.03) 0%, transparent 50%),
    radial-gradient(circle at 80% 70%, rgba(255, 255, 255, 0.02) 0%, transparent 50%),
    radial-gradient(circle at 50% 50%, rgba(255, 255, 255, 0.015) 0%, transparent 60%);
  animation: shimmer 15s ease-in-out infinite;
}

@keyframes shimmer {
  0%, 100% { opacity: 0.8; }
  50% { opacity: 1; }
}

/* 曲线分割效果 - 金色保留 */
/* 曲线分割效果 - 蓝色主题 */
.curved-divider {
  position: absolute;
  right: -15%;
  top: 0;
  width: 30%;
  height: 100%;
  background: linear-gradient(135deg, rgba(15, 105, 166, 0.25) 0%, rgba(13, 90, 135, 0.15) 50%, rgba(10, 70, 106, 0.2) 100%);
  clip-path: polygon(
    30% 0%,
    70% 0%,
    100% 5%,
    100% 95%,
    70% 100%,
    30% 100%,
    0% 95%,
    0% 5%
  );
  box-shadow: -10px 0 30px rgba(15, 105, 166, 0.35);
}

.brand-section::after {
  content: '';
  position: absolute;
  top: -50%;
  right: -20%;
  width: 80%;
  height: 200%;
  background: radial-gradient(circle, rgba(255, 255, 255, 0.02) 0%, transparent 70%);
  animation: float 20s ease-in-out infinite;
}

@keyframes float {
  0%, 100% { transform: translateY(0) rotate(0deg); }
  50% { transform: translateY(-20px) rotate(5deg); }
}

.brand-content {
  text-align: center;
  color: #ffffff;
  z-index: 1;
  padding: 40px;
}

.brand-title {
  font-size: 36px;
  font-weight: 700;
  margin-bottom: 28px;
  letter-spacing: 2px;
  color: #ffffff;
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.5);
}

.brand-slogan {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-bottom: 20px;
  font-size: 20px;
  font-weight: 600;
}

.brand-slogan span {
  padding: 8px 18px;
  background: rgba(255, 255, 255, 0.08);
  border-radius: 0;
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #ffffff;
}

.brand-subtitle {
  font-size: 14px;
  opacity: 0.9;
  margin-bottom: 50px;
  letter-spacing: 1px;
  font-weight: 300;
  color: #cccccc;
}

.brand-illustration {
  max-width: 320px;
  margin: 0 auto;
}

.illustration-svg {
  width: 100%;
  height: auto;
  filter: drop-shadow(0 15px 30px rgba(0, 0, 0, 0.3));
}

/* 右侧登录区 - 白色背景 */
.login-section {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  background: #ffffff;
  padding: 40px 60px;
  position: relative;
}

/* 金色边框装饰保留 */
.login-section::before {
  content: '';
  position: absolute;
  left: 0;
  top: 10%;
  width: 2px;
  height: 80%;
  background: linear-gradient(180deg, transparent 0%, rgba(212, 175, 55, 0.6) 50%, transparent 100%);
}

.login-wrapper {
  width: 100%;
  max-width: 360px;
}

.login-header {
  margin-bottom: 32px;
}

.login-header h2 {
  font-size: 22px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 12px;
}

.header-line {
  width: 50px;
  height: 3px;
  background: linear-gradient(90deg, #0f69a6, #FFD700, #FFA500);
  border-radius: 0;
  box-shadow: 0 0 10px rgba(212, 175, 55, 0.4);
}

.ldap-hint {
  margin-top: 10px;
  font-size: 12px;
  color: #0f69a6;
  background: #f0f7ff;
  padding: 4px 10px;
  border-left: 2px solid #0f69a6;
}

.login-form {
  margin-top: 24px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

/* 黑白风格输入框 - 金色图标保留 */
.login-form :deep(.el-input__wrapper) {
  padding: 8px 12px;
  border-radius: 0;
  background: #ffffff;
  box-shadow: 0 0 0 1px #e0e0e0 inset;
  transition: all 0.3s;
}

.login-form :deep(.el-input__wrapper:hover) {
  box-shadow: 0 0 0 1px #cccccc inset;
}

.login-form :deep(.el-input__wrapper.is-focus) {
  box-shadow: 0 0 0 1px #0f69a6 inset, 0 0 10px rgba(212, 175, 55, 0.15);
  background: #fafafa;
}

.login-form :deep(.el-input__inner) {
  font-size: 14px;
  color: #1a1a1a;
}

.login-form :deep(.el-input__inner)::placeholder {
  color: #999999;
}

.login-form :deep(.el-input__prefix-inner) {
  color: #0f69a6;
}

/* 验证码样式 */
.captcha-wrapper {
  display: flex;
  gap: 10px;
  width: 100%;
}

.captcha-wrapper :deep(.el-input) {
  flex: 1;
}

.captcha-image {
  flex-shrink: 0;
  width: 120px;
  height: auto;
  align-self: stretch;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  overflow: hidden;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  transition: all 0.3s;
}

.captcha-image:hover {
  border-color: #0f69a6;
  background: #ffffff;
  transform: translateY(-1px);
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.2);
}

.captcha-image img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.captcha-loading {
  font-size: 12px;
  color: #999999;
}

/* 表单选项 */
.form-options {
  display: flex;
  justify-content: flex-start;
  align-items: center;
  width: 100%;
}

.form-options :deep(.el-checkbox__label) {
  font-size: 13px;
  color: #666666;
}

.form-options :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background-color: #0f69a6;
  border-color: #0f69a6;
}

.form-options :deep(.el-checkbox__inner) {
  border-color: #d0d0d0;
}

/* 登录按钮 - 黑白风格，金色装饰 */
.login-button {
  width: 100%;
  height: 40px;
  font-size: 14px;
  font-weight: 500;
  border-radius: 0;
  background: linear-gradient(135deg, #1a1a1a 0%, #2d2d2d 100%);
  border: 1px solid #0f69a6;
  color: #0f69a6;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: all 0.3s;
}

.login-button:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15), 0 0 15px rgba(212, 175, 55, 0.25);
  background: linear-gradient(135deg, #0f69a6 0%, #FFD700 100%);
  color: #1a1a1a;
  border-color: #FFD700;
}

.login-button:active {
  transform: translateY(0);
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .brand-section {
    flex: 0 0 55%;
  }

  .brand-title {
    font-size: 28px;
  }

  .brand-slogan {
    font-size: 16px;
    gap: 16px;
  }
}

@media (max-width: 768px) {
  .login-container {
    flex-direction: column;
  }

  .brand-section {
    flex: none;
    min-height: 35vh;
  }

  .curved-divider {
    display: none;
  }

  .brand-title {
    font-size: 24px;
  }

  .brand-slogan {
    font-size: 14px;
    gap: 12px;
  }

  .brand-slogan span {
    padding: 6px 12px;
  }

  .brand-illustration {
    max-width: 200px;
  }

  .login-section {
    padding: 24px 32px;
    align-items: center;
    justify-content: center;
  }

  .login-wrapper {
    padding: 16px;
  }
}
</style>
