<template>
  <div class="skills-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><MagicStick /></el-icon>
        </div>
        <div>
          <h2 class="page-title">Skill 管理</h2>
          <p class="page-subtitle">管理 AI 助手的技能模块，支持启用/禁用和上传自定义 Skill</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedCategory"
          placeholder="全部分类"
          clearable
          style="width: 140px"
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
          <el-button class="black-button" :icon="Upload">上传 Skill</el-button>
        </el-upload>
      </div>
    </div>

    <!-- Skills 统计卡片 -->
    <div class="stats-row" v-if="stats">
      <div class="stat-card total">
        <div class="stat-number">{{ stats.total }}</div>
        <div class="stat-label">Skills 总数</div>
      </div>
      <div class="stat-card builtin">
        <div class="stat-number">{{ stats.builtinEnabled }}<span class="stat-sub">/{{ stats.builtinTotal }}</span></div>
        <div class="stat-label">内置已启用</div>
      </div>
      <div class="stat-card custom">
        <div class="stat-number">{{ stats.customEnabled }}<span class="stat-sub">/{{ stats.customTotal }}</span></div>
        <div class="stat-label">自定义已启用</div>
      </div>
      <div class="stat-card categories" v-for="(count, cat) in stats.byCategory" :key="cat">
        <div class="stat-number">{{ count }}</div>
        <div class="stat-label">{{ getCategoryLabel(cat as string) }}</div>
      </div>
    </div>

    <!-- 上传格式说明 -->
    <el-alert
      type="info"
      :closable="true"
      show-icon
      style="margin-bottom: 16px"
    >
      <template #title>
        <span>Skill 包标准结构 &amp; 替换说明</span>
      </template>
      <div class="upload-hint">
        <code>skill-name.zip</code> 内目录结构:
        <pre>skill-name/
├── SKILL.md          # 必需 (YAML 前置元数据 + Markdown 指令)
└── scripts/          # 可选 (可执行代码)
    └── script.js 或 script.py</pre>
        <div class="hint-tip">
          <strong>替换内置 Skill:</strong> 上传与内置 Skill 同名的包，将自动覆盖内置版本。
          你也可以先禁用内置 Skill，再上传增强版。
        </div>
      </div>
    </el-alert>

    <div class="skills-grid">
      <el-card
        v-for="skill in skills"
        :key="skill.id || skill.name"
        class="skill-card"
        :class="{ 'disabled-card': !skill.isEnabled }"
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
              <el-tag v-if="skill.markdown" type="success" size="small" effect="plain">SKILL.md</el-tag>
              <el-tag v-if="!skill.isEnabled" type="danger" size="small" effect="dark">已禁用</el-tag>
            </div>
          </div>
          <el-tooltip :content="skill.isEnabled ? '点击禁用' : '点击启用'" placement="top">
            <el-switch
              v-model="skill.isEnabled"
              @change="toggleSkillEnabled(skill)"
            />
          </el-tooltip>
        </div>
        <div class="skill-desc">{{ skill.description }}</div>
        <div class="skill-footer">
          <span class="script-type">
            <el-icon v-if="skill.scriptType === 'builtin'"><Setting /></el-icon>
            <el-icon v-else-if="skill.scriptType === 'javascript'"><Document /></el-icon>
            <el-icon v-else><Document /></el-icon>
            {{ getScriptLabel(skill.scriptType) }}
          </span>
          <div class="skill-actions">
            <el-button
              v-if="skill.markdown"
              text
              type="primary"
              size="small"
              @click="showSkillDetail(skill)"
            >
              详情
            </el-button>
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
        </div>
      </el-card>

      <el-card v-if="skills.length === 0" class="empty-card" shadow="never">
        <el-empty description="暂无 Skills，请先添加 AI 模型并启用内置 Skills" />
      </el-card>
    </div>

    <!-- Skill 详情弹窗 -->
    <el-dialog
      v-model="detailVisible"
      :title="detailSkill?.displayName || detailSkill?.name || 'Skill 详情'"
      width="600px"
      class="skill-detail-dialog"
    >
      <div v-if="detailSkill" class="skill-detail">
        <div class="detail-meta">
          <el-descriptions :column="2" size="small" border>
            <el-descriptions-item label="名称">{{ detailSkill.name }}</el-descriptions-item>
            <el-descriptions-item label="分类">{{ getCategoryLabel(detailSkill.category) }}</el-descriptions-item>
            <el-descriptions-item label="风险等级">
              <el-tag :type="getRiskColor(detailSkill.riskLevel)" size="small">
                {{ getRiskLabel(detailSkill.riskLevel) }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="类型">
              {{ getScriptLabel(detailSkill.scriptType) }}
            </el-descriptions-item>
          </el-descriptions>
        </div>
        <div class="detail-desc">
          <h4>描述</h4>
          <p>{{ detailSkill.description }}</p>
        </div>
        <div v-if="detailSkill.markdown" class="detail-markdown">
          <h4>SKILL.md 指令</h4>
          <div class="markdown-content" v-html="renderMarkdown(detailSkill.markdown)" />
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Setting, Document, MagicStick } from '@element-plus/icons-vue'
import { getSkillList, toggleSkill, toggleBuiltinSkill, deleteSkill, uploadSkill, getSkillStats } from '@/api/ai'

const skills = ref<any[]>([])
const selectedCategory = ref('')
const detailVisible = ref(false)
const detailSkill = ref<any>(null)
const stats = ref<any>(null)

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
  loadStats()
})

async function loadSkills() {
  try {
    const data = await getSkillList(selectedCategory.value)
    // request.ts 拦截器成功时直接返回 response.data.data
    skills.value = Array.isArray(data) ? data : []
  } catch (e) {
    ElMessage.error('获取 Skill 列表失败')
  }
}

async function loadStats() {
  try {
    const data = await getSkillStats()
    stats.value = data || null
  } catch {
    // 静默
  }
}

async function toggleSkillEnabled(skill: any) {
  try {
    if (skill.isBuiltin && !skill.id) {
      // 内置 Skill 没有 DB 记录，通过名称 toggle
      const result: any = await toggleBuiltinSkill(skill.name)
      // 更新返回的 id，后续 toggle 就能用 id 了
      if (result?.id) {
        skill.id = result.id
      }
    } else if (skill.id) {
      await toggleSkill(skill.id)
    } else {
      // 既没有 id 也不是内置的，不应该出现
      skill.isEnabled = !skill.isEnabled
      ElMessage.error('操作失败：无法识别 Skill')
      return
    }
    ElMessage.success(skill.isEnabled ? '已启用' : '已禁用')
    loadStats() // 刷新统计
  } catch (e) {
    skill.isEnabled = !skill.isEnabled
    ElMessage.error('操作失败')
  }
}

async function handleDeleteSkill(skill: any) {
  try {
    await ElMessageBox.confirm(`确定删除 Skill "${skill.displayName || skill.name}" ？`, '提示', { type: 'warning' })
    await deleteSkill(skill.id)
    ElMessage.success('删除成功')
    loadSkills()
    loadStats()
  } catch (e) {
    // cancelled
  }
}

function handleUploadBefore(file: any) {
  if (!file.name.endsWith('.zip')) {
    ElMessage.error('请上传 .zip 格式的文件')
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
    await uploadSkill(options.file)
    ElMessage.success('Skill 上传成功')
    loadSkills()
    loadStats()
  } catch (e: any) {
    ElMessage.error(e?.message || '上传失败')
  }
}

function showSkillDetail(skill: any) {
  detailSkill.value = skill
  detailVisible.value = true
}

function renderMarkdown(md: string): string {
  // 简单的 Markdown 渲染（标题、列表、代码块、粗体、分隔线）
  if (!md) return ''
  return md
    .replace(/^### (.+)$/gm, '<h5>$1</h5>')
    .replace(/^## (.+)$/gm, '<h4>$1</h4>')
    .replace(/^# (.+)$/gm, '<h3>$1</h3>')
    .replace(/^\- (.+)$/gm, '<li>$1</li>')
    .replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>')
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    .replace(/^---$/gm, '<hr />')
    .replace(/\n\n/g, '<br /><br />')
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
  padding: 16px;
}

/* ===== Page Header（与主机管理页面一致）===== */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 10px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 0;
  box-shadow: 0 1px 8px rgba(0, 0, 0, 0.04);
  flex-shrink: 0;
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  background: #0a466a;
  border-radius: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 16px;
  flex-shrink: 0;
  border: none;
}

.page-title {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 2px 0 0 0;
  font-size: 12px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.black-button {
  background: #0a466a;
  color: #fff;
  border: none;
}
.black-button:hover,
.black-button:focus {
  background: #0d5a8a;
  color: #fff;
}

/* ===== 统计卡片 ===== */
.stats-row {
  display: flex;
  gap: 10px;
  margin-bottom: 12px;
  flex-wrap: nowrap;
  overflow-x: auto;
}

.stat-card {
  background: #fff;
  border-radius: 4px;
  padding: 10px 14px;
  flex: 1;
  min-width: 0;
  text-align: center;
  border: 1px solid #ebeef5;
  transition: all 0.2s;
}

.stat-card:hover {
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.stat-card.total {
  background: #0a466a;
  color: white;
  border: none;
}
.stat-card.total .stat-label { color: rgba(255,255,255,0.75); }

.stat-card.builtin {
  background: #f0f9eb;
  border-color: #c2e7b0;
}
.stat-card.builtin .stat-number { color: #67c23a; }

.stat-card.custom {
  background: #fdf6ec;
  border-color: #f5dab1;
}
.stat-card.custom .stat-number { color: #e6a23c; }

.stat-card.categories {
  background: #f4f4f5;
  border-color: #dedfe0;
}
.stat-card.categories .stat-number { color: #606266; }

.stat-number {
  font-size: 20px;
  font-weight: 700;
  line-height: 1.3;
}

.stat-sub {
  font-size: 12px;
  font-weight: 400;
  opacity: 0.65;
}

.stat-label {
  font-size: 11px;
  color: #909399;
  margin-top: 2px;
}

/* ===== 上传提示 ===== */
.upload-hint {
  font-size: 12px;
  line-height: 1.5;
}

.upload-hint pre {
  background: #f5f7fa;
  padding: 8px 12px;
  border-radius: 4px;
  margin: 6px 0 0 0;
  font-size: 12px;
  line-height: 1.6;
  color: #606266;
}

.upload-hint code {
  background: #ecf5ff;
  color: #409eff;
  padding: 1px 4px;
  border-radius: 3px;
}

.hint-tip {
  margin-top: 8px;
  padding: 6px 10px;
  background: #fdf6ec;
  border-radius: 4px;
  color: #e6a23c;
  font-size: 12px;
}

/* ===== Skill 卡片网格 ===== */
.skills-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
  gap: 12px;
}

.skill-card {
  border-radius: 6px;
  border: 1px solid #ebeef5;
  transition: all 0.2s;
}

.skill-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
  border-color: #d0d7de;
}

.skill-card.disabled-card {
  opacity: 0.55;
  background: #fafafa;
}

.skill-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.skill-icon {
  width: 38px;
  height: 38px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
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
  min-width: 0;
}

.skill-name {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.skill-meta {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.skill-desc {
  margin: 10px 0;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.skill-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 10px;
  border-top: 1px solid #f2f3f5;
}

.script-type {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #c0c4cc;
}

.skill-actions {
  display: flex;
  gap: 4px;
}

.empty-card {
  grid-column: 1 / -1;
  border-radius: 6px;
}

/* ===== Skill 详情弹窗 ===== */
.skill-detail .detail-meta {
  margin-bottom: 16px;
}

.skill-detail .detail-desc {
  margin-bottom: 16px;
}

.skill-detail .detail-desc h4,
.skill-detail .detail-markdown h4 {
  font-size: 14px;
  margin: 0 0 8px 0;
  color: #303133;
}

.skill-detail .detail-desc p {
  font-size: 13px;
  color: #606266;
  line-height: 1.6;
  margin: 0;
}

.detail-markdown .markdown-content {
  background: #f8f9fa;
  border-radius: 6px;
  padding: 16px;
  font-size: 13px;
  line-height: 1.7;
  color: #303133;
  max-height: 400px;
  overflow-y: auto;
}

.detail-markdown .markdown-content :deep(h3) {
  font-size: 16px;
  margin: 0 0 8px;
}

.detail-markdown .markdown-content :deep(h4) {
  font-size: 14px;
  margin: 12px 0 6px;
}

.detail-markdown .markdown-content :deep(h5) {
  font-size: 13px;
  margin: 10px 0 4px;
}

.detail-markdown .markdown-content :deep(ul) {
  margin: 4px 0;
  padding-left: 20px;
}

.detail-markdown .markdown-content :deep(li) {
  margin: 2px 0;
}

.detail-markdown .markdown-content :deep(code) {
  background: #ecf5ff;
  color: #409eff;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}

.detail-markdown .markdown-content :deep(strong) {
  color: #e6a23c;
}

.detail-markdown .markdown-content :deep(hr) {
  border: none;
  border-top: 1px solid #dcdfe6;
  margin: 12px 0;
}
</style>
