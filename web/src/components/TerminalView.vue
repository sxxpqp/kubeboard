<template>
  <el-dialog
    v-model="visible"
    :title="`终端: ${podName}`"
    width="900px"
    :close-on-click-modal="false"
    destroy-on-close
    @closed="onClosed"
  >
    <div class="term-toolbar">
      <el-select v-model="container" size="small" style="width: 180px">
        <el-option v-for="c in containers" :key="c" :label="c" :value="c" />
      </el-select>
      <el-select v-model="shell" size="small" style="width: 140px">
        <el-option value="/bin/sh" label="/bin/sh" />
        <el-option value="/bin/bash" label="/bin/bash" />
        <el-option value="/bin/ash" label="/bin/ash" />
      </el-select>
      <el-button size="small" type="primary" @click="connect">连接</el-button>
      <el-button size="small" @click="disconnect">断开</el-button>
      <span class="term-status" :class="statusClass">{{ statusText }}</span>
    </div>
    <div ref="termEl" class="term-box"></div>
  </el-dialog>
</template>

<script setup>
import { ref, watch, nextTick } from 'vue'
import { Terminal } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import 'xterm/css/xterm.css'

const props = defineProps({
  modelValue: Boolean,
  namespace: { type: String, required: true },
  podName: { type: String, required: true },
  containers: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const container = ref('')
const shell = ref('/bin/sh')
const termEl = ref(null)
const statusText = ref('未连接')
const statusClass = ref('idle')

let term = null
let fitAddon = null
let ws = null
let resizeObserver = null

watch(() => props.modelValue, (v) => {
  visible.value = v
  if (v) {
    container.value = props.containers[0] || ''
    nextTick(initTerm)
  }
})
watch(visible, (v) => { if (!v) emit('update:modelValue', false) })

function initTerm() {
  term = new Terminal({
    cursorBlink: true,
    fontSize: 13,
    fontFamily: "'SF Mono', Menlo, Monaco, Consolas, monospace",
    theme: { background: '#1e1e1e', foreground: '#d4d4d4' },
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  term.open(termEl.value)
  fitAddon.fit()
  term.writeln('点击「连接」开始会话。')

  term.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) ws.send(data)
  })

  resizeObserver = new ResizeObserver(() => {
    try { fitAddon.fit() } catch { /* ignore */ }
  })
  resizeObserver.observe(termEl.value)
}

function connect() {
  disconnect()
  const proto = location.protocol === 'https:' ? 'wss' : 'ws'
  const params = new URLSearchParams({ namespace: props.namespace, name: props.podName, shell: shell.value })
  if (container.value) params.set('container', container.value)
  params.set('token', localStorage.getItem('kubeboard_token') || '')
  ws = new WebSocket(`${proto}://${location.host}/api/pods/terminal?${params}`)

  ws.onopen = () => {
    statusText.value = '已连接'
    statusClass.value = 'ok'
    term.clear()
  }
  ws.onmessage = (ev) => term.write(ev.data)
  ws.onclose = () => {
    statusText.value = '已断开'
    statusClass.value = 'idle'
  }
  ws.onerror = () => {
    statusText.value = '连接错误'
    statusClass.value = 'err'
  }
}

function disconnect() {
  if (ws) { ws.close(); ws = null }
}

function onClosed() {
  disconnect()
  if (resizeObserver) { resizeObserver.disconnect(); resizeObserver = null }
  if (term) { term.dispose(); term = null }
}
</script>

<style scoped>
.term-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.term-status { margin-left: 8px; font-size: 12px; }
.term-status.ok { color: var(--ks-success); }
.term-status.idle { color: var(--ks-text-secondary); }
.term-status.err { color: var(--ks-danger); }
.term-box { background: #1e1e1e; padding: 8px; border-radius: 4px; height: 460px; }
</style>
