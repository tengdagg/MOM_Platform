<template>
  <div class="no-permission-container">
    <div class="content-wrapper">
      <!-- 图标 -->
      <div class="icon-wrapper">
        <div class="icon-bg">
          <el-icon class="lock-icon"><Lock /></el-icon>
        </div>
        <div class="icon-ring ring-1"></div>
        <div class="icon-ring ring-2"></div>
      </div>

      <!-- 标题和描述 -->
      <h1 class="title">暂无访问权限</h1>
      <p class="description">
        您的账号尚未被分配任何菜单权限，请联系系统管理员为您分配相应的角色和权限。
      </p>

      <!-- 用户信息卡片 -->
      <div class="user-card">
        <div class="user-avatar">
          <el-avatar :size="48" :src="avatarUrl">
            <el-icon><UserFilled /></el-icon>
          </el-avatar>
        </div>
        <div class="user-info">
          <div class="user-name">{{ userStore.userInfo?.realName || userStore.userInfo?.username || '未知用户' }}</div>
          <div class="user-role">{{ userRoleName }}</div>
        </div>
      </div>

      <!-- 提示信息 -->
      <div class="tips">
        <div class="tip-item">
          <el-icon><InfoFilled /></el-icon>
          <span>如需获取权限，请联系管理员</span>
        </div>
        <div class="tip-item">
          <el-icon><ChatDotRound /></el-icon>
          <span>您也可以发送邮件至管理员邮箱</span>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="actions">
        <el-button type="primary" size="large" @click="handleLogout" class="logout-btn">
          <el-icon><SwitchButton /></el-icon>
          <span>退出登录</span>
        </el-button>
        <el-button size="large" @click="handleRefresh" class="refresh-btn">
          <el-icon><Refresh /></el-icon>
          <span>刷新页面</span>
        </el-button>
      </div>
    </div>

    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="circle circle-1"></div>
      <div class="circle circle-2"></div>
      <div class="circle circle-3"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/stores/user'
import {
  Lock,
  UserFilled,
  InfoFilled,
  ChatDotRound,
  SwitchButton,
  Refresh
} from '@element-plus/icons-vue'

const router = useRouter()
const userStore = useUserStore()

// 头像URL
const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''
  if (avatar.startsWith('data:')) return avatar
  return avatar.startsWith('http') ? avatar : `/api${avatar}`
})

// 用户角色名称
const userRoleName = computed(() => {
  const roles = userStore.userInfo?.roles || []
  if (roles.length === 0) return '暂无角色'
  return roles.map((r: any) => r.name).join(', ')
})

// 退出登录
const handleLogout = () => {
  userStore.logout()
  router.push('/login')
}

// 刷新页面
const handleRefresh = () => {
  window.location.reload()
}
</script>

<style scoped>
.no-permission-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #f5f7fa 0%, #e4e8eb 100%);
  position: relative;
  overflow: hidden;
}

.content-wrapper {
  text-align: center;
  padding: 60px 40px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.1);
  max-width: 480px;
  width: 90%;
  position: relative;
  z-index: 10;
}

/* 图标样式 */
.icon-wrapper {
  position: relative;
  width: 120px;
  height: 120px;
  margin: 0 auto 32px;
}

.icon-bg {
  width: 120px;
  height: 120px;
  background: linear-gradient(135deg, #1a1a1a 0%, #333 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  z-index: 2;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.2);
}

.lock-icon {
  font-size: 48px;
  color: #ffffff;
}

.icon-ring {
  position: absolute;
  border-radius: 50%;
  border: 2px solid rgba(212, 175, 55, 0.3);
  animation: pulse 2s ease-in-out infinite;
}

.ring-1 {
  width: 140px;
  height: 140px;
  top: -10px;
  left: -10px;
  animation-delay: 0s;
}

.ring-2 {
  width: 160px;
  height: 160px;
  top: -20px;
  left: -20px;
  animation-delay: 0.5s;
}

@keyframes pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 0.5;
  }
  50% {
    transform: scale(1.05);
    opacity: 0.2;
  }
}

/* 标题和描述 */
.title {
  font-size: 28px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 16px 0;
}

.description {
  font-size: 15px;
  color: #666;
  line-height: 1.6;
  margin: 0 0 32px 0;
  padding: 0 20px;
}

/* 用户卡片 */
.user-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  background: #f8f9fa;
  border-radius: 0;
  margin-bottom: 24px;
}

.user-avatar {
  flex-shrink: 0;
}

.user-info {
  text-align: left;
}

.user-name {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  margin-bottom: 4px;
}

.user-role {
  font-size: 13px;
  color: #909399;
}

/* 提示信息 */
.tips {
  margin-bottom: 32px;
}

.tip-item {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  font-size: 14px;
  color: #909399;
  margin-bottom: 8px;
}

.tip-item:last-child {
  margin-bottom: 0;
}

.tip-item .el-icon {
  color: #ffffff;
}

/* 操作按钮 */
.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
}

.logout-btn {
  background: linear-gradient(135deg, #1a1a1a 0%, #333 100%) !important;
  border: none !important;
  padding: 12px 32px !important;
  border-radius: 0 !important;
  font-weight: 500;
}

.logout-btn:hover {
  background: linear-gradient(135deg, #333 0%, #444 100%) !important;
}

.refresh-btn {
  padding: 12px 32px !important;
  border-radius: 0 !important;
  border-color: #dcdfe6 !important;
}

.refresh-btn:hover {
  border-color: #0f69a6 !important;
  color: #0f69a6 !important;
}

/* 背景装饰 */
.bg-decoration {
  position: absolute;
  width: 100%;
  height: 100%;
  top: 0;
  left: 0;
  pointer-events: none;
  overflow: hidden;
}

.circle {
  position: absolute;
  border-radius: 50%;
  background: linear-gradient(135deg, rgba(212, 175, 55, 0.1) 0%, rgba(212, 175, 55, 0.05) 100%);
}

.circle-1 {
  width: 400px;
  height: 400px;
  top: -200px;
  right: -100px;
}

.circle-2 {
  width: 300px;
  height: 300px;
  bottom: -150px;
  left: -100px;
}

.circle-3 {
  width: 200px;
  height: 200px;
  top: 50%;
  left: 10%;
  transform: translateY(-50%);
}

/* 响应式 */
@media (max-width: 480px) {
  .content-wrapper {
    padding: 40px 24px;
  }

  .title {
    font-size: 24px;
  }

  .actions {
    flex-direction: column;
  }

  .logout-btn,
  .refresh-btn {
    width: 100%;
  }
}
</style>
