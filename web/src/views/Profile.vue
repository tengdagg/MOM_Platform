<template>
  <div class="profile-container">
    <!-- 页面标题 -->
    <div class="page-header">
      <h2 class="page-title">个人信息</h2>
    </div>

    <el-tabs v-model="activeTab" class="profile-tabs">
      <!-- 基本信息标签页 -->
      <el-tab-pane label="基本信息" name="basic">
        <div class="tab-content">
          <el-form
            :model="profileForm"
            :rules="profileRules"
            ref="profileFormRef"
            label-width="100px"
            class="profile-form"
          >
            <el-form-item label="头像">
              <div class="avatar-section">
                <el-avatar :size="100" :src="avatarUrl" :key="avatarKey">
                  <el-icon><UserFilled /></el-icon>
                </el-avatar>
                <el-upload
                  class="avatar-uploader"
                  :show-file-list="false"
                  :before-upload="beforeAvatarUpload"
                  :http-request="handleAvatarUpload"
                  accept="image/*"
                >
                  <el-button class="black-button" :loading="uploadLoading" style="margin-left: 20px;">
                    {{ uploadLoading ? '上传中...' : '更换头像' }}
                  </el-button>
                </el-upload>
              </div>
              <div class="avatar-tip">支持 JPG、PNG 格式,文件大小不超过 2MB</div>
            </el-form-item>

            <el-form-item label="用户名">
              <el-input v-model="profileForm.username" disabled />
            </el-form-item>

            <el-form-item label="真实姓名" prop="realName">
              <el-input v-model="profileForm.realName" placeholder="请输入真实姓名" />
            </el-form-item>

            <el-form-item label="邮箱" prop="email">
              <el-input v-model="profileForm.email" placeholder="请输入邮箱" />
            </el-form-item>

            <el-form-item label="手机号" prop="phone">
              <el-input v-model="profileForm.phone" placeholder="请输入手机号" />
            </el-form-item>

            <el-form-item>
              <el-button class="black-button" @click="handleUpdateProfile" :loading="updateLoading">
                保存修改
              </el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>

      <!-- 修改密码标签页 -->
      <el-tab-pane label="修改密码" name="password">
        <div class="tab-content">
          <el-form
            :model="passwordForm"
            :rules="passwordRules"
            ref="passwordFormRef"
            label-width="100px"
            class="profile-form"
            style="max-width: 600px"
          >
            <el-form-item label="原密码" prop="oldPassword">
              <el-input
                v-model="passwordForm.oldPassword"
                type="password"
                show-password
                placeholder="请输入原密码"
              />
            </el-form-item>

            <el-form-item label="新密码" prop="newPassword">
              <el-input
                v-model="passwordForm.newPassword"
                type="password"
                show-password
                placeholder="请输入新密码（至少6位）"
              />
            </el-form-item>

            <el-form-item label="确认密码" prop="confirmPassword">
              <el-input
                v-model="passwordForm.confirmPassword"
                type="password"
                show-password
                placeholder="请再次输入新密码"
              />
            </el-form-item>

            <el-form-item>
              <el-button class="black-button" @click="handleUpdatePassword" :loading="passwordLoading">
                修改密码
              </el-button>
              <el-button @click="handleResetPassword">重置</el-button>
            </el-form-item>
          </el-form>
        </div>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { UserFilled } from '@element-plus/icons-vue'
import { useUserStore } from '@/stores/user'
import { updateUser, changePassword } from '@/api/user'
import { uploadAvatar, updateUserAvatar } from '@/api/upload'
import type { UploadProps } from 'element-plus'

const userStore = useUserStore()
const activeTab = ref('basic')
const updateLoading = ref(false)
const passwordLoading = ref(false)
const uploadLoading = ref(false)

// 用于强制刷新头像组件的 key
const avatarKey = ref(Date.now())

const profileFormRef = ref<FormInstance>()
const passwordFormRef = ref<FormInstance>()

const profileForm = reactive({
  id: 0,
  username: '',
  realName: '',
  email: '',
  phone: '',
  avatar: ''
})

const passwordForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: ''
})

const profileRules = {
  realName: [{ required: true, message: '请输入真实姓名', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入正确的邮箱格式', trigger: 'blur' }
  ],
  phone: [{ pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号', trigger: 'blur' }]
}

const validateConfirmPassword = (_rule: any, value: any, callback: any) => {
  if (value !== passwordForm.newPassword) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const passwordRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' }
  ]
}

// 加载用户信息
const loadUserInfo = async () => {
  try {
    // 如果 store 中已有用户信息，先显示出来
    if (userStore.userInfo) {
      profileForm.id = userStore.userInfo.ID || userStore.userInfo.id
      profileForm.username = userStore.userInfo.username
      profileForm.realName = userStore.userInfo.realName || ''
      profileForm.email = userStore.userInfo.email || ''
      profileForm.phone = userStore.userInfo.phone || ''
      profileForm.avatar = userStore.userInfo.avatar || ''
    }

    // 然后异步刷新最新数据
    await userStore.getProfile()

    // 刷新后再次更新表单，确保数据最新
    if (userStore.userInfo) {
      profileForm.id = userStore.userInfo.ID || userStore.userInfo.id
      profileForm.username = userStore.userInfo.username
      profileForm.realName = userStore.userInfo.realName || ''
      profileForm.email = userStore.userInfo.email || ''
      profileForm.phone = userStore.userInfo.phone || ''
      profileForm.avatar = userStore.userInfo.avatar || ''
    }
  } catch (error) {
    ElMessage.error('获取用户信息失败')
  }
}

// 头像URL - 添加时间戳破坏缓存
const avatarUrl = computed(() => {
  const avatar = userStore.userInfo?.avatar || ''
  if (!avatar) return ''

  // 如果是base64图片，直接返回
  if (avatar.startsWith('data:')) return avatar

  // 添加时间戳参数破坏浏览器缓存（使用 store 中的时间戳）
  const separator = avatar.includes('?') ? '&' : '?'
  return `${avatar}${separator}t=${userStore.avatarTimestamp}`
})

// 上传前校验
const beforeAvatarUpload: UploadProps['beforeUpload'] = (file) => {
  const isImage = file.type.startsWith('image/')
  const isLt2M = file.size / 1024 / 1024 < 2

  if (!isImage) {
    ElMessage.error('只能上传图片文件!')
    return false
  }
  if (!isLt2M) {
    ElMessage.error('图片大小不能超过 2MB!')
    return false
  }
  return true
}

// 处理头像上传
const handleAvatarUpload = async (options: any) => {
  const { file } = options
  uploadLoading.value = true

  try {
    // 上传图片到服务器
    const uploadRes: any = await uploadAvatar(file)

    if (uploadRes.code === 0 && uploadRes.data) {
      // 获取服务器返回的头像路径
      const serverPath = uploadRes.data.url || uploadRes.data

      // 更新用户头像到服务器（保存相对路径）
      await updateUserAvatar(serverPath)

      // 立即更新 store 中的头像，触发所有组件更新
      userStore.updateAvatar(serverPath)

      // 等待 DOM 更新
      await nextTick()

      // 强制刷新组件（通过改变 key）
      avatarKey.value = Date.now()

      ElMessage.success('头像上传成功')

      // 延迟刷新完整的用户信息（避免覆盖刚更新的头像）
      setTimeout(() => {
        userStore.getProfile().then(() => {
        })
      }, 500)
    } else {
      throw new Error(uploadRes.message || '上传失败')
    }
  } catch (error: any) {
    const errorMsg = error.response?.data?.message || error.message || '头像上传失败'
    ElMessage.error(errorMsg)
  } finally {
    uploadLoading.value = false
  }
}

// 更新基本信息
const handleUpdateProfile = async () => {
  if (!profileFormRef.value) return

  await profileFormRef.value.validate(async (valid) => {
    if (valid) {
      updateLoading.value = true
      try {
        await updateUser(profileForm.id, {
          realName: profileForm.realName,
          email: profileForm.email,
          phone: profileForm.phone
        })
        ElMessage.success('保存成功')
        // 重新获取用户信息
        await userStore.getProfile()
        // 更新表单数据
        if (userStore.userInfo) {
          profileForm.id = userStore.userInfo.ID || userStore.userInfo.id
          profileForm.username = userStore.userInfo.username
          profileForm.realName = userStore.userInfo.realName || ''
          profileForm.email = userStore.userInfo.email || ''
          profileForm.phone = userStore.userInfo.phone || ''
          profileForm.avatar = userStore.userInfo.avatar || ''
        }
      } catch (error) {
        ElMessage.error('保存失败')
      } finally {
        updateLoading.value = false
      }
    }
  })
}

// 修改密码
const handleUpdatePassword = async () => {
  if (!passwordFormRef.value) return

  await passwordFormRef.value.validate(async (valid) => {
    if (valid) {
      passwordLoading.value = true
      try {
        await changePassword(passwordForm.oldPassword, passwordForm.newPassword)
        ElMessage.success('密码修改成功，请重新登录')
        handleResetPassword()
        // 延迟后跳转到登录页
        setTimeout(() => {
          userStore.logout()
          window.location.href = '/login'
        }, 1500)
      } catch (error: any) {
        const errorMsg = error.response?.data?.message || error.message || '修改密码失败'
        ElMessage.error(errorMsg)
      } finally {
        passwordLoading.value = false
      }
    }
  })
}

// 重置密码表单
const handleResetPassword = () => {
  passwordFormRef.value?.resetFields()
}

onMounted(() => {
  loadUserInfo()
})
</script>

<style scoped>
.profile-container {
  padding: 20px;
  background-color: #fff;
  min-height: 100%;
}

.page-header {
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #e6e6e6;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 500;
  color: #303133;
}

.profile-tabs {
  background-color: transparent;
}

.tab-content {
  padding-top: 20px;
}

.profile-form {
  max-width: 800px;
}

.avatar-section {
  display: flex;
  align-items: center;
}

.avatar-section :deep(.el-avatar) {
  background-color: #FFAF35;
  border: 3px solid rgba(255, 255, 255, 0.2);
}

.avatar-section :deep(.el-icon) {
  font-size: 50px;
  color: #fff;
}

.avatar-uploader {
  display: inline-block;
}

.avatar-tip {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

/* 按钮样式 - 使用全局样式 .black-button */
</style>
