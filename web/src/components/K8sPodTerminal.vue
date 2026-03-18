<template>
  <div class="k8s-pod-terminal">
    <div v-if="connecting" class="terminal-loading-overlay">
      <el-icon class="is-loading"><Loading /></el-icon>
      <span>正在连接终端...</span>
    </div>
    <div ref="terminalWrapper" class="terminal-wrapper"></div>
  </div>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Loading } from '@element-plus/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName: string
  container: string
}>()

const emit = defineEmits<{
  (e: 'status-change', connected: boolean): void
  (e: 'error', message: string): void
}>()

const terminalWrapper = ref<HTMLDivElement | null>(null)
const connecting = ref(true)

let terminal: Terminal | null = null
let terminalWebSocket: WebSocket | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let disposables: Array<{ dispose: () => void }> = []
let lastResizeSignature = ''

const buildWsUrl = () => {
  const token = localStorage.getItem('token')
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.hostname
  const isDev = import.meta.env.DEV
  const port = isDev ? '9876' : (window.location.port || (window.location.protocol === 'https:' ? '443' : '9876'))
  const cols = terminal?.cols || 120
  const rows = terminal?.rows || 30

  return `${protocol}//${host}:${port}/api/v1/plugins/kubernetes/shell/pods?` +
    `clusterId=${props.clusterId}&` +
    `namespace=${encodeURIComponent(props.namespace)}&` +
    `podName=${encodeURIComponent(props.podName)}&` +
    `container=${encodeURIComponent(props.container)}&` +
    `cols=${cols}&` +
    `rows=${rows}&` +
    `token=${encodeURIComponent(token || '')}`
}

const destroyTerminal = () => {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }

  disposables.forEach(disposable => disposable.dispose())
  disposables = []

  if (terminalWebSocket) {
    terminalWebSocket.close()
    terminalWebSocket = null
  }

  if (terminal) {
    terminal.dispose()
    terminal = null
  }

  fitAddon = null
  lastResizeSignature = ''
  connecting.value = false
  emit('status-change', false)
}

const sendResize = () => {
  if (!terminal || !terminalWebSocket || terminalWebSocket.readyState !== WebSocket.OPEN) {
    return
  }

  const signature = `${terminal.cols}x${terminal.rows}`
  if (signature === lastResizeSignature) {
    return
  }

  lastResizeSignature = signature
  terminalWebSocket.send(JSON.stringify({
    type: 'resize',
    cols: terminal.cols,
    rows: terminal.rows
  }))
}

const setupResizeObserver = () => {
  if (!terminalWrapper.value || !fitAddon) {
    return
  }

  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
    sendResize()
  })
  resizeObserver.observe(terminalWrapper.value)
}

const waitForContainerReady = async () => {
  let attempts = 0
  while (terminalWrapper.value && (terminalWrapper.value.clientWidth === 0 || terminalWrapper.value.clientHeight === 0) && attempts < 20) {
    await new Promise(resolve => setTimeout(resolve, 50))
    attempts++
  }
}

const initTerminal = async () => {
  destroyTerminal()
  connecting.value = true

  await nextTick()
  if (!terminalWrapper.value) {
    connecting.value = false
    return
  }

  terminalWrapper.value.innerHTML = ''
  await waitForContainerReady()

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#ffffff'
    }
  })

  fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(webLinksAddon)
  terminal.open(terminalWrapper.value)
  fitAddon.fit()
  await nextTick()
  await new Promise(resolve => setTimeout(resolve, 80))
  fitAddon.fit()
  terminal.focus()
  setupResizeObserver()

  terminal.writeln('\x1b[1;32m正在连接到容器...\x1b[0m')

  try {
    terminalWebSocket = new WebSocket(buildWsUrl())
  } catch (error: any) {
    const message = error?.message || '创建 WebSocket 连接失败'
    terminal.writeln(`\x1b[1;31m✗ ${message}\x1b[0m`)
    connecting.value = false
    emit('error', message)
    return
  }

  terminalWebSocket.onopen = () => {
    connecting.value = false
    emit('status-change', true)
    terminal?.clear()
    terminal?.writeln(`\x1b[1;32m✓ 已连接到容器 ${props.container}\x1b[0m`)
    terminal?.writeln('')
    lastResizeSignature = ''
    fitAddon?.fit()
    sendResize()
  }

  terminalWebSocket.onmessage = (event) => {
    terminal?.write(event.data)
  }

  terminalWebSocket.onerror = () => {
    connecting.value = false
    emit('status-change', false)
    terminal?.writeln('\x1b[1;31m✗ 连接错误\x1b[0m')
    terminal?.writeln('请检查:')
    terminal?.writeln('1. 集群连接是否正常')
    terminal?.writeln('2. Pod是否正在运行')
    terminal?.writeln('3. 当前容器是否支持 shell')
    emit('error', '终端连接失败')
  }

  terminalWebSocket.onclose = () => {
    connecting.value = false
    emit('status-change', false)
    try {
      terminal?.writeln('\x1b[1;33m连接已关闭\x1b[0m')
    } catch {
      // ignore terminal teardown race
    }
  }

  disposables.push(terminal.onData((data: string) => {
    if (terminalWebSocket?.readyState === WebSocket.OPEN) {
      terminalWebSocket.send(data)
    }
  }))

  disposables.push(terminal.onResize(() => {
    sendResize()
  }))
}

watch(
  () => [props.clusterId, props.namespace, props.podName, props.container],
  async () => {
    await initTerminal()
  }
)

onMounted(async () => {
  await initTerminal()
})

onBeforeUnmount(() => {
  destroyTerminal()
})
</script>

<style scoped>
.k8s-pod-terminal {
  position: relative;
  flex: 1;
  min-height: 0;
  background: #000;
  overflow: hidden;
}

.terminal-wrapper {
  width: 100%;
  height: 100%;
  min-height: 0;
}

.terminal-wrapper :deep(.xterm) {
  height: 100%;
}

.terminal-wrapper :deep(.xterm .xterm-viewport) {
  height: 100%;
  background-color: #1e1e1e !important;
}

.terminal-wrapper :deep(.xterm .xterm-screen) {
  padding: 0;
}

.terminal-loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: #d4d4d4;
  background: rgba(30, 30, 30, 0.78);
  backdrop-filter: blur(2px);
}
</style>
