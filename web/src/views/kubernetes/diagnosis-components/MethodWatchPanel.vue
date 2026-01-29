<template>
  <div class="method-watch-panel">
    <div v-if="!attached" class="not-attached">
      <el-empty description="请先选择Pod并连接到进程">
        <template #image>
          <el-icon :size="60" color="#909399"><Connection /></el-icon>
        </template>
      </el-empty>
    </div>

    <div v-else class="panel-content">
      <!-- 工具栏 -->
      <div class="toolbar">
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">类名</span>
            <el-input
              v-model="classPattern"
              placeholder="类名表达式 (如: com.example.service.*)"
              style="width: 300px"
              size="default"
              clearable
              :disabled="watching"
            >
              <template #prefix>
                <el-icon><Folder /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="input-group">
            <span class="input-label">方法名</span>
            <el-input
              v-model="methodPattern"
              placeholder="方法名 (如: doSomething)"
              style="width: 200px"
              size="default"
              clearable
              :disabled="watching"
            >
              <template #prefix>
                <el-icon><Promotion /></el-icon>
              </template>
            </el-input>
          </div>
        </div>
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">监测点</span>
            <el-checkbox-group v-model="watchPoints" :disabled="watching">
              <el-checkbox label="-b">方法入口 (before)</el-checkbox>
              <el-checkbox label="-e">方法异常 (exception)</el-checkbox>
              <el-checkbox label="-s">方法返回 (return)</el-checkbox>
              <el-checkbox label="-f">方法结束 (finish)</el-checkbox>
            </el-checkbox-group>
          </div>
        </div>
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">观察表达式</span>
            <el-input
              v-model="expression"
              placeholder="OGNL 表达式 (如: {params, returnObj, throwExp})"
              style="width: 350px"
              size="default"
              clearable
              :disabled="watching"
            >
              <template #prefix>
                <el-icon><View /></el-icon>
              </template>
            </el-input>
          </div>
          <div class="input-group">
            <span class="input-label">条件表达式</span>
            <el-input
              v-model="condition"
              placeholder="条件 (如: params[0]>100)"
              style="width: 200px"
              size="default"
              clearable
              :disabled="watching"
            >
              <template #prefix>
                <el-icon><Filter /></el-icon>
              </template>
            </el-input>
          </div>
        </div>
        <div class="toolbar-row">
          <div class="input-group">
            <span class="input-label">最大次数</span>
            <el-input-number
              v-model="maxCount"
              :min="1"
              :max="1000"
              size="default"
              :disabled="watching"
              style="width: 120px"
            />
          </div>
          <div class="input-group">
            <span class="input-label">展开层数</span>
            <el-input-number
              v-model="expandLevel"
              :min="0"
              :max="5"
              size="default"
              :disabled="watching"
              style="width: 100px"
            />
          </div>
          <el-checkbox v-model="sizeLimit" :disabled="watching">
            限制字符串长度
          </el-checkbox>
        </div>
        <div class="toolbar-row actions">
          <el-button
            type="primary"
            @click="startWatch"
            :loading="starting"
            :disabled="watching || !classPattern || !methodPattern"
          >
            <el-icon><VideoPlay /></el-icon>
            {{ starting ? '启动中...' : '开始监测' }}
          </el-button>
          <el-button
            @click="stopWatch"
            :disabled="!watching"
            type="danger"
          >
            <el-icon><VideoPause /></el-icon> 停止监测
          </el-button>
          <el-button @click="clearOutput">
            <el-icon><Delete /></el-icon> 清空输出
          </el-button>
          <el-divider direction="vertical" />
          <el-tag v-if="watching" type="success" effect="dark">
            <el-icon class="is-loading"><Loading /></el-icon>
            监测中...
          </el-tag>
          <el-tag v-else type="info">未监测</el-tag>
          <span class="watch-count" v-if="watchCount > 0">
            已捕获: {{ watchCount }} 次调用
          </span>
        </div>
      </div>

      <!-- 使用说明 -->
      <el-collapse v-model="showHelp" class="help-collapse">
        <el-collapse-item title="使用说明" name="help">
          <div class="help-content">
            <p><strong>watch 命令</strong> 可以观察方法执行时的入参、返回值和异常信息。</p>

            <div class="help-section">
              <h4>监测点说明</h4>
              <ul>
                <li><strong>方法入口 (-b)</strong>: 在方法调用之前观察，此时入参可见但返回值为空</li>
                <li><strong>方法异常 (-e)</strong>: 在方法抛出异常时观察</li>
                <li><strong>方法返回 (-s)</strong>: 在方法正常返回时观察</li>
                <li><strong>方法结束 (-f)</strong>: 在方法结束时观察（无论正常返回还是抛出异常）</li>
              </ul>
            </div>

            <div class="help-section">
              <h4>观察表达式</h4>
              <p>使用 OGNL 表达式指定要观察的内容，常用变量:</p>
              <ul>
                <li><code>params</code> - 入参数组</li>
                <li><code>returnObj</code> - 返回值</li>
                <li><code>throwExp</code> - 异常对象</li>
                <li><code>target</code> - 当前对象</li>
                <li><code>clazz</code> - 类对象</li>
                <li><code>method</code> - 方法对象</li>
              </ul>
              <p>示例: <code>{params, returnObj}</code>, <code>params[0].name</code>, <code>target.field</code></p>
            </div>

            <p class="tip">提示: 展开层数控制对象打印的深度，层数越大输出越详细但可能影响性能。</p>
          </div>
        </el-collapse-item>
      </el-collapse>

      <!-- 快捷模板 -->
      <div class="template-bar">
        <span class="template-label">快捷模板:</span>
        <el-button size="small" @click="applyTemplate('params')" :disabled="watching">
          查看入参
        </el-button>
        <el-button size="small" @click="applyTemplate('return')" :disabled="watching">
          查看返回值
        </el-button>
        <el-button size="small" @click="applyTemplate('all')" :disabled="watching">
          查看全部
        </el-button>
        <el-button size="small" @click="applyTemplate('exception')" :disabled="watching">
          捕获异常
        </el-button>
        <el-button size="small" @click="applyTemplate('cost')" :disabled="watching">
          耗时分析
        </el-button>
      </div>

      <!-- 监测输出 -->
      <div class="output-section">
        <div class="section-header">
          <span>监测输出</span>
          <span class="output-info">
            <el-tag size="small" type="info">{{ outputLineCount }} 行</el-tag>
          </span>
        </div>
        <div class="watch-output" ref="outputRef">
          <div v-if="!rawOutput && !watching" class="empty-output">
            <el-empty description="等待监测结果...">
              <template #image>
                <el-icon :size="40" color="#c0c4cc"><View /></el-icon>
              </template>
            </el-empty>
          </div>
          <pre v-else>{{ cleanOutput }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Connection, VideoPlay, VideoPause, Delete, Folder, Promotion, Filter, View, Loading } from '@element-plus/icons-vue'
import { createArthasWebSocket, type ArthasWSMessage } from '@/api/arthas'

const props = defineProps<{
  clusterId: number | null
  namespace: string
  pod: string
  container: string
  processId: string
  attached: boolean
}>()

// 表单数据
const classPattern = ref('')
const methodPattern = ref('')
const watchPoints = ref<string[]>(['-b', '-s']) // 默认监测入口和返回
const expression = ref('{params, returnObj, throwExp}')
const condition = ref('')
const maxCount = ref(100)
const expandLevel = ref(2)
const sizeLimit = ref(true)

// 状态
const watching = ref(false)
const starting = ref(false)
const watchCount = ref(0)
const showHelp = ref<string[]>([])
const rawOutput = ref('')  // 使用原始字符串
const outputRef = ref<HTMLElement | null>(null)

// WebSocket 连接
let ws: WebSocket | null = null

// 计算输出行数
const outputLineCount = computed(() => {
  return rawOutput.value.split('\n').length
})

// 清理输出中的 ANSI 转义码
const cleanOutput = computed(() => {
  return rawOutput.value
    .replace(/\x1b\[[0-9;]*m/g, '')
    .replace(/\033\[[0-9;]*m/g, '')
    .replace(/\[\d+;\d+m/g, '')
    .replace(/\[\d+m/g, '')
    .replace(/\[0m/g, '')
    .replace(/\[m/g, '')
})

// 构建 watch 命令
const buildWatchCommand = (): string => {
  let cmd = `watch ${classPattern.value} ${methodPattern.value}`

  // 添加观察表达式
  if (expression.value) {
    cmd += ` '${expression.value}'`
  }

  // 添加条件表达式
  if (condition.value) {
    cmd += ` '${condition.value}'`
  }

  // 添加监测点选项
  watchPoints.value.forEach(point => {
    cmd += ` ${point}`
  })

  // 添加其他选项
  cmd += ` -n ${maxCount.value}`
  cmd += ` -x ${expandLevel.value}`

  if (sizeLimit.value) {
    cmd += ' -M 256' // 限制字符串长度为256
  }

  return cmd
}

// 应用快捷模板
const applyTemplate = (template: string) => {
  switch (template) {
    case 'params':
      expression.value = 'params'
      watchPoints.value = ['-b']
      break
    case 'return':
      expression.value = 'returnObj'
      watchPoints.value = ['-s']
      break
    case 'all':
      expression.value = '{params, returnObj, throwExp}'
      watchPoints.value = ['-b', '-s', '-e']
      break
    case 'exception':
      expression.value = '{throwExp, throwExp.message, throwExp.stackTrace}'
      watchPoints.value = ['-e']
      break
    case 'cost':
      expression.value = '{params, returnObj, #cost}'
      watchPoints.value = ['-f']
      break
  }
}

// 开始监测
const startWatch = async () => {
  if (!classPattern.value || !methodPattern.value) {
    ElMessage.warning('请输入类名和方法名')
    return
  }

  if (watchPoints.value.length === 0) {
    ElMessage.warning('请至少选择一个监测点')
    return
  }

  if (!props.clusterId || !props.namespace || !props.pod || !props.container) {
    ElMessage.warning('请先选择 Pod 和容器')
    return
  }

  starting.value = true
  watchCount.value = 0
  rawOutput.value = ''

  try {
    // 创建 WebSocket 连接
    ws = createArthasWebSocket({
      clusterId: props.clusterId,
      namespace: props.namespace,
      pod: props.pod,
      container: props.container,
      processId: props.processId
    })

    ws.onopen = () => {
      // 发送 watch 命令
      const command = buildWatchCommand()
      rawOutput.value = `[INFO] 执行命令: ${command}\n\n`

      const msg: ArthasWSMessage = {
        type: 'command',
        command: command
      }
      ws?.send(JSON.stringify(msg))
      watching.value = true
      starting.value = false
      ElMessage.success('开始监测')
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.type === 'output') {
          // 监测输出 - 直接追加到原始输出
          const content = data.content
          rawOutput.value += content

          // 统计监测次数（通过检测特定模式）
          if (content.includes('method=') || content.includes('ts=') || content.includes('@')) {
            watchCount.value++
          }

          // 自动滚动到底部
          scrollToBottom()
        } else if (data.type === 'error') {
          rawOutput.value += `\n[ERROR] ${data.content}\n`
          ElMessage.error(data.content)
        } else if (data.type === 'info') {
          rawOutput.value += `[INFO] ${data.content}\n`
        }
      } catch (e) {
        // 如果不是 JSON，直接追加原始数据
        rawOutput.value += event.data
      }
    }

    ws.onerror = (error) => {
      rawOutput.value += '\n[ERROR] WebSocket 连接错误\n'
      watching.value = false
      starting.value = false
      ElMessage.error('WebSocket 连接失败')
    }

    ws.onclose = () => {
      watching.value = false
      starting.value = false
      rawOutput.value += '\n[INFO] 监测已停止\n'
    }

  } catch (error: any) {
    ElMessage.error('启动监测失败: ' + (error.message || '未知错误'))
    starting.value = false
  }
}

// 停止监测
const stopWatch = () => {
  if (ws) {
    const msg: ArthasWSMessage = {
      type: 'stop'
    }
    ws.send(JSON.stringify(msg))
    ws.close()
    ws = null
  }
  watching.value = false
  rawOutput.value += '\n[INFO] 用户停止监测\n'
  ElMessage.info('已停止监测')
}

// 清空输出
const clearOutput = () => {
  rawOutput.value = ''
  watchCount.value = 0
}

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick()
  if (outputRef.value) {
    outputRef.value.scrollTop = outputRef.value.scrollHeight
  }
}

// 组件卸载时清理
onUnmounted(() => {
  if (ws) {
    ws.close()
    ws = null
  }
})

// 监听 attached 状态变化
watch(() => props.attached, (newVal) => {
  if (!newVal && ws) {
    ws.close()
    ws = null
    watching.value = false
  }
})
</script>

<style scoped>
.method-watch-panel {
  min-height: 400px;
}

.not-attached {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 400px;
}

.panel-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* 工具栏 */
.toolbar {
  background: #f8f9fa;
  border-radius: 0;
  padding: 16px;
  border: 1px solid #e9ecef;
}

.toolbar-row {
  display: flex;
  gap: 16px;
  align-items: center;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.toolbar-row:last-child {
  margin-bottom: 0;
}

.toolbar-row.actions {
  padding-top: 12px;
  border-top: 1px solid #e9ecef;
}

.input-group {
  display: flex;
  align-items: center;
  gap: 8px;
}

.input-label {
  font-size: 13px;
  color: #606266;
  white-space: nowrap;
  font-weight: 500;
}

.watch-count {
  font-size: 13px;
  color: #67c23a;
  font-weight: 500;
  margin-left: 8px;
}

/* 帮助折叠面板 */
.help-collapse {
  border: 1px solid #e4e7ed;
  border-radius: 0;
}

.help-collapse :deep(.el-collapse-item__header) {
  padding: 0 16px;
  font-size: 13px;
  color: #606266;
}

.help-content {
  padding: 0 8px;
  font-size: 13px;
  color: #606266;
  line-height: 1.8;
}

.help-section {
  margin: 12px 0;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 0;
}

.help-section h4 {
  margin: 0 0 8px 0;
  font-size: 13px;
  color: #303133;
}

.help-content ul {
  margin: 8px 0;
  padding-left: 20px;
}

.help-content code {
  background: #f0f0f0;
  padding: 2px 6px;
  border-radius: 0;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: #e6a23c;
}

.help-content .tip {
  background: #fdf6ec;
  padding: 8px 12px;
  border-radius: 0;
  border-left: 3px solid #e6a23c;
  margin-top: 12px;
}

/* 模板栏 */
.template-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
}

.template-label {
  font-size: 13px;
  color: #909399;
  font-weight: 500;
}

/* 输出区域 */
.output-section {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 0;
  overflow: hidden;
  flex: 1;
  display: flex;
  flex-direction: column;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #ebeef5;
  font-size: 14px;
  font-weight: 600;
  color: #303133;
}

.output-info {
  display: flex;
  gap: 8px;
  align-items: center;
}

.watch-output {
  flex: 1;
  overflow: auto;
  min-height: 300px;
  max-height: 500px;
}

.watch-output pre {
  margin: 0;
  padding: 16px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  line-height: 1.6;
  min-height: 100%;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty-output {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  background: #fafafa;
}

/* 加载动画 */
.is-loading {
  animation: rotating 1s linear infinite;
}

@keyframes rotating {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 响应式 */
@media (max-width: 992px) {
  .toolbar-row {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }

  .input-group {
    width: 100%;
  }

  .input-group :deep(.el-input) {
    width: 100% !important;
  }

  .template-bar {
    flex-wrap: wrap;
  }
}
</style>
