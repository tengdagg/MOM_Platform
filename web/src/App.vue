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

/* ================================
   主题色按钮样式 - #0a466a
   ================================ */

/* 黑色按钮类覆盖 - 改为主题色 */
.black-button,
.black-button.el-button,
button.black-button,
.el-button.black-button,
[class*="black-button"] {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
  color: #ffffff !important;
  background-image: none !important;
  background: #0a466a !important;
}

.black-button:hover,
.black-button:focus,
.black-button.el-button:hover,
.black-button.el-button:focus,
.el-button.black-button:hover,
.el-button.black-button:focus {
  background-color: #0d5a87 !important;
  border-color: #0d5a87 !important;
  color: #ffffff !important;
  background-image: none !important;
  background: #0d5a87 !important;
}

.black-button:active,
.black-button.el-button:active,
.el-button.black-button:active {
  background-color: #083a56 !important;
  border-color: #083a56 !important;
  background-image: none !important;
  background: #083a56 !important;
}

/* 所有带黑色渐变背景的按钮 */
.el-button[style*="background"],
.el-button[style*="#000"] {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
  background-image: none !important;
  background: #0a466a !important;
}

/* 主要按钮 - 使用更高优先级选择器 */
.el-button--primary,
.el-button.el-button--primary,
button.el-button--primary,
.batch-actions-right .el-button--primary,
.header-actions .el-button--primary,
.filter-actions .el-button--primary,
.search-btn.el-button--primary {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
  background-image: none !important;
  background: #0a466a !important;
}

.el-button--primary:hover,
.el-button--primary:focus,
.el-button.el-button--primary:hover,
.el-button.el-button--primary:focus,
.batch-actions-right .el-button--primary:hover,
.header-actions .el-button--primary:hover,
.filter-actions .el-button--primary:hover {
  background-color: #0d5a87 !important;
  border-color: #0d5a87 !important;
  background-image: none !important;
  background: #0d5a87 !important;
}

.el-button--primary:active,
.el-button.el-button--primary:active {
  background-color: #083a56 !important;
  border-color: #083a56 !important;
  background-image: none !important;
  background: #083a56 !important;
}

.el-button--primary.is-disabled,
.el-button--primary.is-disabled:hover {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
  background-image: none !important;
  background: #0a466a !important;
  opacity: 0.6;
}

/* 朴素主要按钮 */
.el-button--primary.is-plain {
  background-color: rgba(10, 70, 106, 0.1) !important;
  border-color: #0a466a !important;
  color: #0a466a !important;
}

.el-button--primary.is-plain:hover,
.el-button--primary.is-plain:focus {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
  color: #ffffff !important;
}

/* 文字按钮 */
.el-button--text {
  color: #0a466a !important;
}

.el-button--text:hover,
.el-button--text:focus {
  color: #0d5a87 !important;
}

/* 链接按钮 */
.el-button--link {
  color: #0a466a !important;
}

.el-button--link:hover,
.el-button--link:focus {
  color: #0d5a87 !important;
}

/* 复选框选中 */
.el-checkbox__input.is-checked .el-checkbox__inner {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
}

.el-checkbox__input.is-checked + .el-checkbox__label {
  color: #0a466a !important;
}

.el-checkbox__input.is-indeterminate .el-checkbox__inner {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
}

/* 单选框选中 */
.el-radio__input.is-checked .el-radio__inner {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
}

.el-radio__input.is-checked + .el-radio__label {
  color: #0a466a !important;
}

/* 开关 */
.el-switch.is-checked .el-switch__core {
  background-color: #0a466a !important;
  border-color: #0a466a !important;
}

/* 标签页 */
.el-tabs__item.is-active {
  color: #0a466a !important;
}

.el-tabs__item:hover {
  color: #0a466a !important;
}

.el-tabs__active-bar {
  background-color: #0a466a !important;
}

/* 分页选中 */
.el-pagination.is-background .el-pager li:not(.is-disabled).is-active {
  background-color: #0a466a !important;
}

.el-pager li:not(.is-disabled).is-active {
  color: #0a466a !important;
}

/* 链接 */
.el-link--primary {
  color: #0a466a !important;
}

.el-link--primary:hover {
  color: #0d5a87 !important;
}

/* 下拉选中项 */
.el-select-dropdown__item.is-selected {
  color: #0a466a !important;
}

/* 输入框聚焦 */
.el-input__wrapper:focus-within,
.el-input__wrapper.is-focus {
  box-shadow: 0 0 0 1px #0a466a inset !important;
}

.el-textarea__inner:focus {
  box-shadow: 0 0 0 1px #0a466a inset !important;
}

/* 选择器聚焦 */
.el-select .el-input.is-focus .el-input__wrapper {
  box-shadow: 0 0 0 1px #0a466a inset !important;
}

/* 日期选择器 */
.el-date-table td.current:not(.disabled) span {
  background-color: #0a466a !important;
}

.el-date-table td.today span {
  color: #0a466a !important;
}

/* 树形控件 */
.el-tree-node.is-current > .el-tree-node__content {
  background-color: rgba(10, 70, 106, 0.1) !important;
}

/* 步骤条 */
.el-step__head.is-process {
  color: #0a466a !important;
  border-color: #0a466a !important;
}

.el-step__head.is-finish {
  color: #0a466a !important;
  border-color: #0a466a !important;
}

.el-step__title.is-process,
.el-step__title.is-finish {
  color: #0a466a !important;
}

/* 加载中 */
.el-loading-spinner .circular {
  stroke: #0a466a !important;
}

.el-loading-spinner .el-loading-text {
  color: #0a466a !important;
}
</style>
