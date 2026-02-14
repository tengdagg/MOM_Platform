<template>
  <div class="model-config-page">
    <div class="page-header">
      <h3>模型配置</h3>
      <el-button type="primary" :icon="Plus" @click="showAddDialog">添加模型</el-button>
    </div>

    <div class="model-cards">
      <el-card
        v-for="model in models"
        :key="model.id"
        class="model-card"
        :class="{ 'is-default': model.isDefault }"
        shadow="hover"
      >
        <div class="model-card-header">
          <div class="provider-badge" :class="model.provider">
            {{ getProviderIcon(model.provider) }}
          </div>
          <div class="model-info">
            <div class="model-name">
              {{ model.name }}
              <el-tag v-if="model.isDefault" type="success" size="small">默认</el-tag>
              <el-tag v-if="model.status === 0" type="danger" size="small">已禁用</el-tag>
            </div>
            <div class="model-detail">{{ model.modelName }}</div>
          </div>
          <el-dropdown trigger="click">
            <el-icon class="more-btn"><MoreFilled /></el-icon>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item @click="editModel(model)">
                  <el-icon><Edit /></el-icon> 编辑
                </el-dropdown-item>
                <el-dropdown-item @click="testModelConn(model)">
                  <el-icon><Connection /></el-icon> 测试连接
                </el-dropdown-item>
                <el-dropdown-item v-if="!model.isDefault" @click="setDefault(model)">
                  <el-icon><Star /></el-icon> 设为默认
                </el-dropdown-item>
                <el-dropdown-item divided @click="handleDelete(model)" style="color: #f56c6c">
                  <el-icon><Delete /></el-icon> 删除
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
        <div class="model-card-body">
          <div class="model-field">
            <span class="label">提供商</span>
            <span class="value">{{ getProviderLabel(model.provider) }}</span>
          </div>
          <div class="model-field">
            <span class="label">API 地址</span>
            <span class="value url">{{ model.baseUrl || '默认' }}</span>
          </div>
          <div class="model-field">
            <span class="label">API Key</span>
            <span class="value">{{ model.apiKeySet ? '••••••••' : '未设置' }}</span>
          </div>
          <div class="model-field">
            <span class="label">Max Tokens</span>
            <span class="value">{{ model.maxTokens }}</span>
          </div>
          <div class="model-field">
            <span class="label">Temperature</span>
            <span class="value">{{ model.temperature }}</span>
          </div>
        </div>
      </el-card>

      <el-card v-if="models.length === 0" class="empty-card" shadow="never">
        <el-empty description="暂未配置 AI 模型">
          <el-button type="primary" @click="showAddDialog">添加模型</el-button>
        </el-empty>
      </el-card>
    </div>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      :title="isEdit ? '编辑模型' : '添加模型'"
      width="520px"
      destroy-on-close
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="显示名称" prop="name">
          <el-input v-model="form.name" placeholder="例如: GPT-4o" />
        </el-form-item>
        <el-form-item label="提供商" prop="provider">
          <el-select v-model="form.provider" placeholder="选择提供商" style="width: 100%" @change="onProviderChange">
            <el-option-group label="国际厂商">
              <el-option label="OpenAI" value="openai" />
              <el-option label="Google Gemini" value="gemini" />
            </el-option-group>
            <el-option-group label="国内厂商">
              <el-option label="通义千问 (阿里)" value="qwen" />
              <el-option label="DeepSeek" value="deepseek" />
              <el-option label="豆包 (字节)" value="doubao" />
            </el-option-group>
            <el-option-group label="本地 / 其他">
              <el-option label="Ollama (本地)" value="ollama" />
              <el-option label="OpenAI 兼容" value="openai_compatible" />
              <el-option label="自定义" value="custom" />
            </el-option-group>
          </el-select>
        </el-form-item>
        <el-form-item label="模型名称" prop="modelName">
          <el-select
            v-if="providerModels[form.provider]"
            v-model="form.modelName"
            filterable
            allow-create
            placeholder="选择或输入模型名称"
            style="width: 100%"
          >
            <el-option
              v-for="m in providerModels[form.provider]"
              :key="m.value"
              :label="m.label"
              :value="m.value"
            />
          </el-select>
          <el-input v-else v-model="form.modelName" placeholder="输入模型名称">
            <template #append>
              <el-tooltip content="模型名称需与 API 提供商支持的模型名一致" placement="top">
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="API 地址" prop="baseUrl">
          <el-input v-model="form.baseUrl" :placeholder="getBaseUrlPlaceholder()">
            <template #append>
              <el-tooltip content="留空使用默认地址" placement="top">
                <el-icon><QuestionFilled /></el-icon>
              </el-tooltip>
            </template>
          </el-input>
        </el-form-item>
        <el-form-item label="API Key" prop="apiKey">
          <el-input
            v-model="form.apiKey"
            type="password"
            show-password
            :placeholder="isEdit ? '留空保持不变' : '输入 API Key'"
          />
        </el-form-item>
        <el-form-item label="Max Tokens">
          <el-input-number v-model="form.maxTokens" :min="256" :max="128000" :step="1024" />
        </el-form-item>
        <el-form-item label="Temperature">
          <el-slider v-model="form.temperature" :min="0" :max="2" :step="0.1" show-input :show-input-controls="false" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" :loading="submitting">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Plus, MoreFilled, Edit, Connection, Star, Delete, QuestionFilled
} from '@element-plus/icons-vue'
import {
  getModelList, createModel, updateModel, deleteModel, testModel, setDefaultModel
} from '@/api/ai'

const models = ref<any[]>([])
const dialogVisible = ref(false)
const isEdit = ref(false)
const editId = ref(0)
const submitting = ref(false)
const formRef = ref()

const form = ref({
  name: '',
  provider: 'openai',
  baseUrl: '',
  apiKey: '',
  modelName: '',
  maxTokens: 4096,
  temperature: 0.7,
})

const rules = {
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  provider: [{ required: true, message: '请选择提供商', trigger: 'change' }],
  modelName: [{ required: true, message: '请输入模型名称', trigger: 'blur' }],
}

onMounted(() => {
  loadModels()
})

async function loadModels() {
  try {
    const data = await getModelList()
    // request.ts 拦截器成功时直接返回 response.data.data
    models.value = Array.isArray(data) ? data : (data?.list || [])
  } catch (e) {
    ElMessage.error('获取模型列表失败')
  }
}

function showAddDialog() {
  isEdit.value = false
  editId.value = 0
  form.value = {
    name: '',
    provider: 'openai',
    baseUrl: '',
    apiKey: '',
    modelName: '',
    maxTokens: 4096,
    temperature: 0.7,
  }
  dialogVisible.value = true
}

function editModel(model: any) {
  isEdit.value = true
  editId.value = model.id
  form.value = {
    name: model.name,
    provider: model.provider,
    baseUrl: model.baseUrl,
    apiKey: '',
    modelName: model.modelName,
    maxTokens: model.maxTokens,
    temperature: model.temperature,
  }
  dialogVisible.value = true
}

async function submitForm() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    if (isEdit.value) {
      const data = { ...form.value }
      if (!data.apiKey) delete (data as any).apiKey
      await updateModel(editId.value, data)
      ElMessage.success('更新成功')
    } else {
      await createModel(form.value)
      ElMessage.success('添加成功')
    }
    dialogVisible.value = false
    loadModels()
  } catch (e) {
    ElMessage.error('操作失败')
  }
  submitting.value = false
}

async function testModelConn(model: any) {
  try {
    const data = await testModel(model.id)
    ElMessage.success(data?.message || '连接成功')
  } catch (e: any) {
    ElMessage.error(e?.message || '连接测试失败')
  }
}

async function setDefault(model: any) {
  try {
    await setDefaultModel(model.id)
    ElMessage.success('已设为默认模型')
    loadModels()
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

async function handleDelete(model: any) {
  try {
    await ElMessageBox.confirm(`确定删除模型 "${model.name}" ？`, '提示', { type: 'warning' })
    await deleteModel(model.id)
    ElMessage.success('删除成功')
    loadModels()
  } catch (e) {
    // cancelled
  }
}

// 各提供商的预设模型列表
const providerModels: Record<string, { label: string; value: string }[]> = {
  openai: [
    { label: 'GPT-4o', value: 'gpt-4o' },
    { label: 'GPT-4o Mini', value: 'gpt-4o-mini' },
    { label: 'GPT-4 Turbo', value: 'gpt-4-turbo' },
    { label: 'GPT-3.5 Turbo', value: 'gpt-3.5-turbo' },
    { label: 'o1', value: 'o1' },
    { label: 'o1-mini', value: 'o1-mini' },
  ],
  gemini: [
    { label: 'Gemini 2.0 Flash', value: 'gemini-2.0-flash' },
    { label: 'Gemini 2.0 Pro', value: 'gemini-2.0-pro-exp-02-05' },
    { label: 'Gemini 1.5 Pro', value: 'gemini-1.5-pro' },
    { label: 'Gemini 1.5 Flash', value: 'gemini-1.5-flash' },
  ],
  qwen: [
    { label: 'Qwen-Max', value: 'qwen-max' },
    { label: 'Qwen-Plus', value: 'qwen-plus' },
    { label: 'Qwen-Turbo', value: 'qwen-turbo' },
    { label: 'Qwen-Long', value: 'qwen-long' },
    { label: 'Qwen2.5-72B', value: 'qwen2.5-72b-instruct' },
    { label: 'Qwen2.5-32B', value: 'qwen2.5-32b-instruct' },
  ],
  deepseek: [
    { label: 'DeepSeek-V3', value: 'deepseek-chat' },
    { label: 'DeepSeek-R1', value: 'deepseek-reasoner' },
  ],
  doubao: [
    { label: '豆包-Pro-32K', value: 'doubao-pro-32k' },
    { label: '豆包-Pro-128K', value: 'doubao-pro-128k' },
    { label: '豆包-Lite-32K', value: 'doubao-lite-32k' },
    { label: '豆包-Lite-128K', value: 'doubao-lite-128k' },
  ],
  ollama: [
    { label: 'Llama 3.1 (8B)', value: 'llama3.1' },
    { label: 'Llama 3.1 (70B)', value: 'llama3.1:70b' },
    { label: 'Qwen2.5 (7B)', value: 'qwen2.5' },
    { label: 'DeepSeek-R1 (7B)', value: 'deepseek-r1' },
    { label: 'Mistral (7B)', value: 'mistral' },
    { label: 'Gemma 2 (9B)', value: 'gemma2' },
  ],
}

// 提供商默认 API 地址
const providerDefaultUrls: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  gemini: 'https://generativelanguage.googleapis.com/v1beta/openai',
  qwen: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  deepseek: 'https://api.deepseek.com/v1',
  doubao: 'https://ark.cn-beijing.volces.com/api/v3',
  ollama: 'http://localhost:11434/v1',
  openai_compatible: '',
  custom: '',
}

// 切换提供商时自动填充
function onProviderChange(provider: string) {
  form.value.modelName = ''
  form.value.baseUrl = ''
  // 自动填充名称
  const nameMap: Record<string, string> = {
    openai: 'OpenAI', gemini: 'Gemini', qwen: '通义千问',
    deepseek: 'DeepSeek', doubao: '豆包', ollama: 'Ollama',
  }
  if (nameMap[provider] && !form.value.name) {
    form.value.name = nameMap[provider]
  }
}

function getProviderLabel(provider: string): string {
  const map: Record<string, string> = {
    openai: 'OpenAI',
    gemini: 'Google Gemini',
    qwen: '通义千问',
    deepseek: 'DeepSeek',
    doubao: '豆包',
    ollama: 'Ollama (本地)',
    openai_compatible: 'OpenAI 兼容',
    custom: '自定义',
  }
  return map[provider] || provider
}

function getProviderIcon(provider: string): string {
  const map: Record<string, string> = {
    openai: 'AI',
    gemini: 'G',
    qwen: '千',
    deepseek: 'DS',
    doubao: '豆',
    ollama: '🦙',
    openai_compatible: 'AI',
    custom: '⚙',
  }
  return map[provider] || '🤖'
}

function getBaseUrlPlaceholder(): string {
  const url = providerDefaultUrls[form.value.provider]
  return url || '输入 API 基础地址'
}
</script>

<style scoped>
.model-config-page {
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

.model-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 16px;
}

.model-card {
  border-radius: 12px;
  transition: transform 0.2s;
}

.model-card:hover {
  transform: translateY(-2px);
}

.model-card.is-default {
  border-color: #67c23a;
}

.model-card-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 16px;
}

.provider-badge {
  width: 48px;
  height: 48px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: bold;
  color: #fff;
}

.provider-badge.openai { background: linear-gradient(135deg, #10a37f, #1a7f64); }
.provider-badge.gemini { background: linear-gradient(135deg, #4285f4, #1a73e8); }
.provider-badge.qwen { background: linear-gradient(135deg, #ff6a00, #ee5a24); }
.provider-badge.deepseek { background: linear-gradient(135deg, #536dfe, #304ffe); }
.provider-badge.doubao { background: linear-gradient(135deg, #00d4aa, #00b894); }
.provider-badge.ollama { background: linear-gradient(135deg, #6366f1, #4f46e5); }
.provider-badge.openai_compatible { background: linear-gradient(135deg, #10a37f, #1a7f64); }
.provider-badge.custom { background: linear-gradient(135deg, #f59e0b, #d97706); }

.model-info {
  flex: 1;
}

.model-name {
  font-size: 16px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 6px;
}

.model-detail {
  font-size: 13px;
  color: #909399;
  margin-top: 2px;
}

.more-btn {
  font-size: 20px;
  cursor: pointer;
  color: #909399;
}

.more-btn:hover {
  color: #409eff;
}

.model-card-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.model-field {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.model-field .label {
  color: #909399;
}

.model-field .value {
  color: #303133;
  font-weight: 500;
}

.model-field .value.url {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-card {
  grid-column: 1 / -1;
  border-radius: 12px;
}
</style>
