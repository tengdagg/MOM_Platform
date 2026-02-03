<template>
  <div class="info-panel volume-panel">
    <div class="panel-header">
      <span class="panel-icon">💾</span>
      <span class="panel-title">数据卷</span>
      <el-button type="primary" @click="emit('addVolume')" :icon="Plus" size="small">添加</el-button>
    </div>
    <div class="panel-content">
      <div class="volume-list">
        <div v-for="(volume, index) in volumes" :key="'volume-'+index" class="volume-item">
          <div class="volume-row" @click="toggleExpand(index)">
            <div class="volume-info">
              <el-icon class="expand-icon" :class="{ expanded: expandedIndex === index }">
                <ArrowRight />
              </el-icon>
              <span class="volume-name-type">{{ volume.name || '未命名' }} ({{ getTypeLabel(volume.type) }})</span>
            </div>
            <el-button
              link
              type="danger"
              @click.stop="emit('removeVolume', index)"
              :icon="Delete"
              size="small"
            />
          </div>
          <div v-if="expandedIndex === index" class="volume-detail">
            <div class="volume-detail-form">
              <div class="form-row">
                <label>名称</label>
                <el-input v-model="volume.name" placeholder="volume-name" size="small" @input="updateVolume(index)" />
              </div>
              <div class="form-row">
                <label>类型</label>
                <el-select v-model="volume.type" placeholder="选择类型" size="small" style="width: 100%;" @change="handleTypeChange(index)">
                  <el-option label="EmptyDir" value="emptyDir" />
                  <el-option label="HostPath" value="hostPath" />
                  <el-option label="NFS" value="nfs" />
                  <el-option label="ConfigMap" value="configMap" />
                  <el-option label="Secret" value="secret" />
                  <el-option label="PVC" value="persistentVolumeClaim" />
                </el-select>
              </div>

              <!-- EmptyDir 配置 -->
              <template v-if="volume.type === 'emptyDir'">
                <div class="form-row">
                  <label>存储介质</label>
                  <el-select v-model="volume.medium" placeholder="默认使用节点磁盘" size="small" style="width: 100%;" @change="updateVolume(index)" clearable>
                    <el-option label="Memory（内存）" value="Memory" />
                  </el-select>
                </div>
                <div class="form-row" v-if="volume.medium === 'Memory'">
                  <label>大小限制</label>
                  <el-input v-model="volume.sizeLimit" placeholder="如: 1Gi, 100Mi" size="small" @input="updateVolume(index)" />
                </div>
              </template>

              <!-- HostPath 配置 -->
              <template v-if="volume.type === 'hostPath'">
                <div class="form-row">
                  <label>主机路径</label>
                  <el-input v-model="volume.hostPath!.path" placeholder="/host/path" size="small" @input="updateVolume(index)" />
                </div>
                <div class="form-row">
                  <label>路径类型</label>
                  <el-select v-model="volume.hostPath!.type" placeholder="默认（自动创建）" size="small" style="width: 100%;" @change="updateVolume(index)" clearable>
                    <el-option label="DirectoryOrCreate" value="DirectoryOrCreate" />
                    <el-option label="Directory" value="Directory" />
                    <el-option label="FileOrCreate" value="FileOrCreate" />
                    <el-option label="File" value="File" />
                    <el-option label="Socket" value="Socket" />
                    <el-option label="CharDevice" value="CharDevice" />
                    <el-option label="BlockDevice" value="BlockDevice" />
                  </el-select>
                </div>
              </template>

              <!-- NFS 配置 -->
              <template v-if="volume.type === 'nfs'">
                <div class="form-row">
                  <label>服务器地址</label>
                  <el-input v-model="volume.nfs!.server" placeholder="192.168.1.100" size="small" @input="updateVolume(index)" />
                </div>
                <div class="form-row">
                  <label>共享路径</label>
                  <el-input v-model="volume.nfs!.path" placeholder="/exports/data" size="small" @input="updateVolume(index)" />
                </div>
                <div class="form-row">
                  <label>只读</label>
                  <el-switch v-model="volume.nfs!.readOnly" @change="updateVolume(index)" />
                </div>
              </template>

              <!-- ConfigMap 配置 -->
              <template v-if="volume.type === 'configMap'">
                <div class="form-row">
                  <label>
                    ConfigMap 名称
                    <el-button link type="primary" @click="emit('refreshConfigMaps')" :icon="Refresh" size="small" circle />
                  </label>
                  <div class="select-with-refresh">
                    <el-select
                      v-model="volume.configMap!.name"
                      filterable
                      placeholder="选择 ConfigMap"
                      size="small"
                      style="flex: 1;"
                      @change="updateVolume(index)"
                    >
                      <el-option
                        v-for="cm in configMaps"
                        :key="cm.name"
                        :label="cm.name"
                        :value="cm.name"
                      />
                    </el-select>
                  </div>
                </div>
                <div class="form-row">
                  <label>默认权限</label>
                  <el-input-number v-model="volume.configMap!.defaultMode" :min="0" :max="511" size="small" style="width: 100%;" @change="updateVolume(index)" />
                </div>
                <div class="form-row">
                  <label>
                    键值映射
                    <el-button link type="primary" @click="addConfigMapItem(volume)" :icon="Plus" size="small">添加</el-button>
                  </label>
                  <div class="items-list">
                    <div v-for="(item, idx) in volume.configMap!.items" :key="'cm-item-'+idx" class="item-row">
                      <el-input v-model="item.key" placeholder="key" size="small" style="flex: 1;" @input="updateVolume(index)" />
                      <span class="arrow">→</span>
                      <el-input v-model="item.path" placeholder="path" size="small" style="flex: 1;" @input="updateVolume(index)" />
                      <el-input-number v-model="item.mode" :min="0" :max="511" placeholder="mode" size="small" style="width: 100px;" @change="updateVolume(index)" />
                      <el-button link type="danger" @click="removeConfigMapItem(volume, idx)" :icon="Delete" size="small" />
                    </div>
                    <div v-if="!volume.configMap!.items || volume.configMap!.items.length === 0" class="empty-items">使用全部键值</div>
                  </div>
                </div>
              </template>

              <!-- Secret 配置 -->
              <template v-if="volume.type === 'secret'">
                <div class="form-row">
                  <label>
                    Secret 名称
                    <el-button link type="primary" @click="emit('refreshSecrets')" :icon="Refresh" size="small" circle />
                  </label>
                  <div class="select-with-refresh">
                    <el-select
                      v-model="volume.secret!.secretName"
                      filterable
                      placeholder="选择 Secret"
                      size="small"
                      style="flex: 1;"
                      @change="updateVolume(index)"
                    >
                      <el-option
                        v-for="sec in secrets"
                        :key="sec.name"
                        :label="sec.name"
                        :value="sec.name"
                      />
                    </el-select>
                  </div>
                </div>
                <div class="form-row">
                  <label>默认权限</label>
                  <el-input-number v-model="volume.secret!.defaultMode" :min="0" :max="511" size="small" style="width: 100%;" @change="updateVolume(index)" />
                </div>
                <div class="form-row">
                  <label>
                    键值映射
                    <el-button link type="primary" @click="addSecretItem(volume)" :icon="Plus" size="small">添加</el-button>
                  </label>
                  <div class="items-list">
                    <div v-for="(item, idx) in volume.secret!.items" :key="'sec-item-'+idx" class="item-row">
                      <el-input v-model="item.key" placeholder="key" size="small" style="flex: 1;" @input="updateVolume(index)" />
                      <span class="arrow">→</span>
                      <el-input v-model="item.path" placeholder="path" size="small" style="flex: 1;" @input="updateVolume(index)" />
                      <el-input-number v-model="item.mode" :min="0" :max="511" placeholder="mode" size="small" style="width: 100px;" @change="updateVolume(index)" />
                      <el-button link type="danger" @click="removeSecretItem(volume, idx)" :icon="Delete" size="small" />
                    </div>
                    <div v-if="!volume.secret!.items || volume.secret!.items.length === 0" class="empty-items">使用全部键值</div>
                  </div>
                </div>
              </template>

              <!-- PVC 配置 -->
              <template v-if="volume.type === 'persistentVolumeClaim'">
                <div class="form-row">
                  <label>
                    声明名称
                    <el-button link type="primary" @click="emit('refreshPVCs')" :icon="Refresh" size="small" circle />
                  </label>
                  <div class="select-with-refresh">
                    <el-select
                      v-model="volume.persistentVolumeClaim!.claimName"
                      filterable
                      placeholder="选择 PVC"
                      size="small"
                      style="flex: 1;"
                      @change="updateVolume(index)"
                    >
                      <el-option
                        v-for="pvc in pvcs"
                        :key="pvc.name"
                        :label="pvc.name"
                        :value="pvc.name"
                      />
                    </el-select>
                  </div>
                </div>
                <div class="form-row">
                  <label>只读</label>
                  <el-switch v-model="volume.persistentVolumeClaim!.readOnly" @change="updateVolume(index)" />
                </div>
              </template>
            </div>
          </div>
        </div>
        <div v-if="volumes.length === 0" class="empty-tip">
          <el-empty description="暂无数据卷" :image-size="60" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Delete, ArrowRight, Refresh } from '@element-plus/icons-vue'

interface ConfigMapItem {
  key: string
  path: string
  mode?: number
}

interface ConfigMapVolume {
  name: string
  defaultMode?: number
  items?: ConfigMapItem[]
}

interface SecretVolume {
  secretName: string
  defaultMode?: number
  items?: ConfigMapItem[]
}

interface HostPathVolume {
  path: string
  type?: string
}

interface NFSVolume {
  server: string
  path: string
  readOnly?: boolean
}

interface PVCVolume {
  claimName: string
  readOnly?: boolean
}

interface Volume {
  name: string
  type: string
  medium?: string
  sizeLimit?: string
  hostPath?: HostPathVolume
  nfs?: NFSVolume
  configMap?: ConfigMapVolume
  secret?: SecretVolume
  persistentVolumeClaim?: PVCVolume
}

const props = defineProps<{
  volumes: Volume[]
  configMaps?: { name: string }[]
  secrets?: { name: string }[]
  pvcs?: { name: string }[]
}>()

const emit = defineEmits<{
  addVolume: []
  removeVolume: [index: number]
  update: [volumes: Volume[]]
  refreshConfigMaps: []
  refreshSecrets: []
  refreshPVCs: []
}>()

const expandedIndex = ref<number>(-1)

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    emptyDir: 'EmptyDir',
    hostPath: 'HostPath',
    nfs: 'NFS',
    configMap: 'ConfigMap',
    secret: 'Secret',
    persistentVolumeClaim: 'PVC'
  }
  return labels[type] || type
}

const toggleExpand = (index: number) => {
  if (expandedIndex.value === index) {
    expandedIndex.value = -1
  } else {
    // 展开前初始化对应的对象结构
    const volume = props.volumes[index]
    if (volume.type === 'emptyDir') {
      volume.medium = volume.medium ?? ''
      volume.sizeLimit = volume.sizeLimit ?? ''
    } else if (volume.type === 'hostPath' && !volume.hostPath) {
      volume.hostPath = { path: '', type: '' }
    } else if (volume.type === 'nfs' && !volume.nfs) {
      volume.nfs = { server: '', path: '', readOnly: false }
    } else if (volume.type === 'configMap' && !volume.configMap) {
      volume.configMap = { name: '', defaultMode: 0o644, items: [] }
    } else if (volume.type === 'secret' && !volume.secret) {
      volume.secret = { secretName: '', defaultMode: 0o644, items: [] }
    } else if (volume.type === 'persistentVolumeClaim' && !volume.persistentVolumeClaim) {
      volume.persistentVolumeClaim = { claimName: '', readOnly: false }
    }
    expandedIndex.value = index
  }
}

const updateVolume = (index: number) => {
  emit('update', [...props.volumes])
}

const handleTypeChange = (index: number) => {
  const volume = props.volumes[index]
  const newType = volume.type

  // 清理所有类型的属性
  delete volume.hostPath
  delete volume.nfs
  delete volume.configMap
  delete volume.secret
  delete volume.persistentVolumeClaim
  delete volume.medium
  delete volume.sizeLimit

  // 根据新类型初始化对应的子对象
  if (newType === 'emptyDir') {
    volume.medium = ''
    volume.sizeLimit = ''
  } else if (newType === 'hostPath') {
    volume.hostPath = { path: '', type: '' }
  } else if (newType === 'nfs') {
    volume.nfs = { server: '', path: '', readOnly: false }
  } else if (newType === 'configMap') {
    volume.configMap = { name: '', defaultMode: 0o644, items: [] }
  } else if (newType === 'secret') {
    volume.secret = { secretName: '', defaultMode: 0o644, items: [] }
  } else if (newType === 'persistentVolumeClaim') {
    volume.persistentVolumeClaim = { claimName: '', readOnly: false }
  }

  emit('update', [...props.volumes])
}

const addConfigMapItem = (volume: Volume) => {
  if (!volume.configMap) {
    volume.configMap = { name: '', items: [] }
  }
  if (!volume.configMap.items) {
    volume.configMap.items = []
  }
  volume.configMap.items.push({ key: '', path: '', mode: 0o644 })
  updateVolume(props.volumes.indexOf(volume))
}

const removeConfigMapItem = (volume: Volume, idx: number) => {
  if (volume.configMap?.items) {
    volume.configMap.items.splice(idx, 1)
    updateVolume(props.volumes.indexOf(volume))
  }
}

const addSecretItem = (volume: Volume) => {
  if (!volume.secret) {
    volume.secret = { secretName: '', items: [] }
  }
  if (!volume.secret.items) {
    volume.secret.items = []
  }
  volume.secret.items.push({ key: '', path: '', mode: 0o644 })
  updateVolume(props.volumes.indexOf(volume))
}

const removeSecretItem = (volume: Volume, idx: number) => {
  if (volume.secret?.items) {
    volume.secret.items.splice(idx, 1)
    updateVolume(props.volumes.indexOf(volume))
  }
}
</script>

<style scoped>
.volume-panel {
  background: #ffffff;
  border-radius: 0;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  background: #0f69a6;
  border-bottom: 1px solid #0f69a6;
}

.panel-icon {
  font-size: 18px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  letter-spacing: 0.3px;
}

.panel-header .el-button {
  font-weight: 500;
  border-radius: 0;
  background: #0a466a !important;
  border: none;
  color: #ffffff !important;
  padding: 8px 15px;
  height: 24px;
}

.panel-header .el-button:hover {
  background: #0d5a87 !important;
  border-color: #0d5a87;
  box-shadow: 0 2px 8px rgba(10, 70, 106, 0.3);
  opacity: 1;
}

.panel-content {
  padding: 16px;
  background: #ffffff;
}

.volume-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.volume-item {
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 0;
  overflow: hidden;
  transition: all 0.3s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.volume-item:hover {
  border-color: #ffffff;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.15);
}

.volume-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fafafa;
  cursor: pointer;
  transition: all 0.3s ease;
}

.volume-row:hover {
  background: #f5f5f5;
}

.volume-info {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
}

.expand-icon {
  font-size: 16px;
  color: #666;
  transition: transform 0.3s ease;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.volume-name-type {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
}

.volume-detail {
  padding: 20px;
  background: #ffffff;
  border-top: 1px solid #e8e8e8;
}

.volume-detail-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.form-row label {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  letter-spacing: 0.3px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.form-row :deep(.el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.form-row :deep(.el-input__wrapper:hover) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.form-row :deep(.el-input__wrapper.is-focus) {
  border-color: #ffffff;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.form-row :deep(.el-select .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 0;
}

.select-with-refresh {
  display: flex;
  align-items: center;
  gap: 8px;
}

.items-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.item-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px;
  background: #fafafa;
  border-radius: 0;
  border: 1px solid #e8e8e8;
}

.arrow {
  color: #ffffff;
  font-weight: 600;
  font-size: 16px;
}

.empty-items {
  text-align: center;
  padding: 16px;
  color: #999;
  font-size: 13px;
  background: #fafafa;
  border-radius: 0;
  border: 1px dashed #e0e0e0;
}

.form-row :deep(.el-switch) {
  --el-switch-on-color: #ffffff;
}

.empty-tip {
  text-align: center;
  padding: 40px;
  color: #999;
  font-size: 14px;
  background: #fafafa;
  border-radius: 0;
  border: 1px dashed #e0e0e0;
}
</style>
