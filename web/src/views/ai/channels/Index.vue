<template>
  <div class="channel-index-page">
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><ChatDotRound /></el-icon>
        </div>
        <div>
          <h2 class="page-title">聊天渠道</h2>
          <p class="page-subtitle">为 AI 助手接入外部聊天入口，首期支持飞书应用私聊长连接。</p>
        </div>
      </div>
    </div>

    <el-card class="channel-shell" shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="飞书" name="feishu" />
        <el-tab-pane label="企业微信" name="wecom" />
        <el-tab-pane label="钉钉" name="dingtalk" />
      </el-tabs>

      <Feishu v-if="activeTab === 'feishu'" />

      <el-empty
        v-else-if="activeTab === 'wecom'"
        description="企业微信渠道页面已预留，后续可直接复用当前聊天渠道架构接入。"
      />

      <el-empty
        v-else
        description="钉钉渠道页面已预留，后续可直接复用当前聊天渠道架构接入。"
      />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ChatDotRound } from '@element-plus/icons-vue'
import Feishu from './Feishu.vue'

const activeTab = ref('feishu')
</script>

<style scoped>
.channel-index-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 16px;
}

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

.channel-shell {
  border-radius: 16px;
}
</style>
