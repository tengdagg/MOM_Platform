<template>
  <router-view />
</template>

<script setup lang="ts">
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html,
body,
#app {
  height: 100%;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial,
    sans-serif;
}

/* 全局字体大小 - 紧凑模式 */
html {
  font-size: 14px;
}

body {
  font-size: 14px;
}

/* 调整Element Plus组件字体大小 - 紧凑模式 */
.el-button {
  font-size: 13px !important;
}

.el-input__inner,
.el-textarea__inner {
  font-size: 13px !important;
}

.el-form-item__label {
  font-size: 13px !important;
}

.el-table {
  font-size: 13px !important;
}

.el-table th,
.el-table td {
  font-size: 13px !important;
}

.el-menu-item {
  font-size: 13px !important;
}

.el-dropdown-menu__item {
  font-size: 13px !important;
}

.el-dialog__title {
  font-size: 15px !important;
}

.el-breadcrumb__inner {
  font-size: 13px !important;
}

.el-tag {
  font-size: 12px !important;
}

.el-pagination {
  font-size: 13px !important;
}

.el-card__header {
  font-size: 14px !important;
}

/* 侧边栏特殊字体 */
.el-aside .user-name {
  font-size: 12px !important;
}

.el-aside .user-role {
  font-size: 11px !important;
}

.logo h3 {
  font-size: 16px !important;
}

/* 表格行高紧凑 */
.el-table .el-table__cell {
  padding: 8px 0 !important;
}

/* 表单项间距紧凑 */
.el-form-item {
  margin-bottom: 14px !important;
}

/* 卡片内边距紧凑 */
.el-card__body {
  padding: 14px !important;
}

/* 对话框内边距紧凑 */
.el-dialog__body {
  padding: 14px 16px !important;
}

/* 分页紧凑 */
.el-pagination .el-pagination__sizes .el-input__inner {
  font-size: 13px !important;
}

/* 选择器选项紧凑 */
.el-select-dropdown__item {
  font-size: 13px !important;
  height: 30px !important;
  line-height: 30px !important;
}

/* 输入框高度紧凑 */
.el-input--default {
  --el-input-height: 32px;
}

/* 按钮高度紧凑 */
.el-button--default {
  --el-button-size: 32px;
}
</style>
