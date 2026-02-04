<template>
  <div ref="editorContainer" class="monaco-editor-container"></div>
</template>

<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, toRaw } from 'vue'
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'

self.MonacoEnvironment = {
  getWorker(_, label) {
    if (label === 'json') {
      return new jsonWorker()
    }
    if (label === 'css' || label === 'scss' || label === 'less') {
      return new cssWorker()
    }
    if (label === 'html' || label === 'handlebars' || label === 'razor') {
      return new htmlWorker()
    }
    if (label === 'typescript' || label === 'javascript') {
      return new tsWorker()
    }
    return new editorWorker()
  }
}

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  },
  readOnly: {
    type: Boolean,
    default: false
  },
  language: {
    type: String,
    default: 'yaml'
  },
  theme: {
    type: String,
    default: 'vs-dark'
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const editorContainer = ref<HTMLElement | null>(null)
let editor: monaco.editor.IStandaloneCodeEditor | null = null
let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  if (editorContainer.value) {
    editor = monaco.editor.create(editorContainer.value, {
      value: props.modelValue,
      language: props.language,
      theme: props.theme,
      readOnly: props.readOnly,
      automaticLayout: false, // We use ResizeObserver for better control
      minimap: {
        enabled: true,
        scale: 0.75,
        renderCharacters: false
      },
      scrollBeyondLastLine: false,
      fontSize: 13,
      fontFamily: "'Monaco', 'Menlo', 'Courier New', monospace",
      tabSize: 2,
      insertSpaces: true,
      lineNumbers: 'on',
      renderLineHighlight: 'all',
      scrollbar: {
        verticalScrollbarSize: 10,
        horizontalScrollbarSize: 10
      }
    })

    // 监听内容变化
    editor.onDidChangeModelContent(() => {
      const value = editor?.getValue() || ''
      if (value !== props.modelValue) {
        emit('update:modelValue', value)
        emit('change', value)
      }
    })

    // 监听容器大小变化
    resizeObserver = new ResizeObserver(() => {
      editor?.layout()
    })
    resizeObserver.observe(editorContainer.value)
  }
})

// 监听外部 modelValue 变化
watch(() => props.modelValue, (newValue) => {
  if (editor && newValue !== editor.getValue()) {
    const position = editor.getPosition()
    editor.setValue(newValue)
    if (position) {
      editor.setPosition(position)
    }
  }
})

// 监听 theme 变化
watch(() => props.theme, (newTheme) => {
  if (editor) {
    monaco.editor.setTheme(newTheme)
  }
})

// 监听 readOnly 变化
watch(() => props.readOnly, (newVal) => {
  if (editor) {
    editor.updateOptions({ readOnly: newVal })
  }
})

onBeforeUnmount(() => {
  if (resizeObserver) {
    resizeObserver.disconnect()
  }
  if (editor) {
    editor.dispose()
  }
})

// 暴露 editor 实例，以便父组件可以进行更高级的操作
defineExpose({
  getEditor: () => editor
})
</script>

<style scoped>
.monaco-editor-container {
  width: 100%;
  height: 100%;
  min-height: 400px;
  overflow: hidden;
}
</style>
