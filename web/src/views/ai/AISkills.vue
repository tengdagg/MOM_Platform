<template>
  <div class="skills-page">
    <div class="page-header">
      <h3>Skill 管理</h3>
      <div class="header-actions">
        <el-select
          v-model="selectedCategory"
          placeholder="全部分类"
          clearable
          style="width: 150px; margin-right: 12px"
          @change="loadSkills"
        >
          <el-option
            v-for="cat in categories"
            :key="cat.value"
            :label="cat.label"
            :value="cat.value"
          />
        </el-select>
        <el-upload
          :show-file-list="false"
          accept=".zip"
          :before-upload="handleUploadBefore"
          :http-request="handleUploadRequest"
        >
          <el-button type="primary" :icon="Upload">上传 Skill</el-button>
        </el-upload>
      </div>
    </div>

    <div class="skills-grid">
      <el-card
        v-for="skill in skills"
        :key="skill.id || skill.name"
        class="skill-card"
        shadow="hover"
      >
        <div class="skill-header">
          <div class="skill-icon" :class="skill.category">
            {{ getCategoryIcon(skill.category) }}
          </div>
          <div class="skill-info">
            <div class="skill-name">{{ skill.displayName || skill.name }}</div>
            <div class="skill-meta">
              <el-tag :type="getRiskColor(skill.riskLevel)" size="small" effect="light">
                {{ getRiskLabel(skill.riskLevel) }}
              </el-tag>
              <el-tag size="small" effect="plain">{{ getCategoryLabel(skill.category) }}</el-tag>
              <el-tag v-if="skill.isBuiltin" type="info" size="small" effect="plain">内置</el-tag>
            </div>
          </div>
          <el-switch
            v-model="skill.isEnabled"
            @change="toggleSkillEnabled(skill)"
            :disabled="!skill.id"
          />
        </div>
        <div class="skill-desc">{{ skill.description }}</div>
        <div class="skill-footer">
          <span class="script-type">
            <el-icon v-if="skill.scriptType === 'builtin'"><Setting /></el-icon>
            <el-icon v-else-if="skill.scriptType === 'javascript'"><Document /></el-icon>
            <el-icon v-else><Document /></el-icon>
            {{ getScriptLabel(skill.scriptType) }}
          </span>
          <el-button
            v-if="!skill.isBuiltin"
            text
            type="danger"
            size="small"
            @click="handleDeleteSkill(skill)"
          >
            删除
          </el-button>
        </div>
      </el-card>

      <el-card v-if="skills.length === 0" class="empty-card" shadow="never">
        <el-empty description="暂无 Skills，请先添加 AI 模型并启用内置 Skills" />
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Setting, Document } from '@element-plus/icons-vue'
import { getSkillList, toggleSkill, deleteSkill } from '@/api/ai'

const skills = ref<any[]>([])
const selectedCategory = ref('')

const categories = [
  { value: 'host', label: '主机管理' },
  { value: 'k8s', label: 'Kubernetes' },
  { value: 'task', label: '任务中心' },
  { value: 'monitor', label: '监控告警' },
  { value: 'audit', label: '审计分析' },
  { value: 'cloud', label: '云账号' },
  { value: 'analysis', label: '综合分析' },
]

onMounted(() => {
  loadSkills()
})

async function loadSkills() {
  try {
    const res = await getSkillList(selectedCategory.value)
    if (res.data?.code === 0) {
      skills.value = res.data.data || []
    }
  } catch (e) {
    ElMessage.error('获取 Skill 列表失败')
  }
}

async function toggleSkillEnabled(skill: any) {
  if (!skill.id) return
  try {
    await toggleSkill(skill.id)
  } catch (e) {
    skill.isEnabled = !skill.isEnabled
    ElMessage.error('操作失败')
  }
}

async function handleDeleteSkill(skill: any) {
  try {
    await ElMessageBox.confirm(`确定删除 Skill "${skill.displayName}" ？`, '提示', { type: 'warning' })
    await deleteSkill(skill.id)
    ElMessage.success('删除成功')
    loadSkills()
  } catch (e) {
    // cancelled
  }
}

function handleUploadBefore(file: any) {
  if (!file.name.endsWith('.zip')) {
    ElMessage.error('请上传 .skill.zip 格式的文件')
    return false
  }
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.error('文件大小不能超过 5MB')
    return false
  }
  return true
}

async function handleUploadRequest(options: any) {
  try {
    const res = await uploadSkill(options.file)
    if (res.data?.code === 0) {
      ElMessage.success(res.data.message || 'Skill 上传成功')
      loadSkills()
    } else {
      ElMessage.error(res.data?.message || '上传失败')
    }
  } catch (e) {
    ElMessage.error('上传失败')
  }
}

function getCategoryIcon(cat: string): string {
  const map: Record<string, string> = {
    host: '🖥', k8s: '☸', task: '⚡', monitor: '📊',
    audit: '🔍', cloud: '☁', analysis: '📈',
  }
  return map[cat] || '🔧'
}

function getCategoryLabel(cat: string): string {
  return categories.find(c => c.value === cat)?.label || cat
}

function getRiskColor(level: string): string {
  const map: Record<string, string> = {
    low: 'success', medium: 'warning', high: 'danger', critical: 'danger',
  }
  return map[level] || 'info'
}

function getRiskLabel(level: string): string {
  const map: Record<string, string> = {
    low: '低风险', medium: '中风险', high: '高风险', critical: '危险',
  }
  return map[level] || level
}

function getScriptLabel(t: string): string {
  const map: Record<string, string> = {
    builtin: 'Go 内置', javascript: 'JavaScript', python: 'Python',
  }
  return map[t] || t
}
</script>

<style scoped>
.skills-page {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h3 {
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
}

.skills-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}

.skill-card {
  border-radius: 12px;
}

.skill-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.skill-icon {
  width: 44px;
  height: 44px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
}

.skill-icon.host { background: #e8f5e9; }
.skill-icon.k8s { background: #e3f2fd; }
.skill-icon.task { background: #fff3e0; }
.skill-icon.monitor { background: #f3e5f5; }
.skill-icon.audit { background: #e0f2f1; }
.skill-icon.cloud { background: #e8eaf6; }
.skill-icon.analysis { background: #fce4ec; }

.skill-info {
  flex: 1;
}

.skill-name {
  font-size: 15px;
  font-weight: 600;
  margin-bottom: 4px;
}

.skill-meta {
  display: flex;
  gap: 4px;
}

.skill-desc {
  margin: 12px 0;
  font-size: 13px;
  color: #606266;
  line-height: 1.5;
}

.skill-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.script-type {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #909399;
}

.empty-card {
  grid-column: 1 / -1;
  border-radius: 12px;
}
</style>
