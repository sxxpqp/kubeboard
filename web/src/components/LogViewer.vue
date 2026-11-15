<template>
  <component :is="embedded ? 'section' : 'el-dialog'"
    v-model="visible"
    :title="`日志: ${podName}`"
    width="min(1100px, 90vw)"
    :close-on-click-modal="false"
    @closed="$emit('update:modelValue', false)"
  >
    <p class="log-context">项目 {{ namespace }} / Pod {{ podName }} · {{ connectionStatus }}</p>
    <div class="log-toolbar">
      <el-select v-model="container" placeholder="容器" size="small" style="width: 180px" @change="reload">
        <el-option v-for="c in containers" :key="c" :label="c" :value="c" />
      </el-select>
      <el-checkbox v-model="follow" size="small" @change="reload">实时跟随</el-checkbox>
      <el-select v-model="tailLines" size="small" style="width: 120px" @change="reload">
        <el-option :value="100" label="最近 100 行" />
        <el-option :value="500" label="最近 500 行" />
        <el-option :value="1000" label="最近 1000 行" />
        <el-option :value="5000" label="最近 5000 行" />
      </el-select>
      <el-checkbox v-model="autoScroll" size="small">自动滚动</el-checkbox>
      <div style="flex: 1"></div>
      <el-button size="small" :icon="'Refresh'" @click="reload" />
    </div>
    <pre ref="logBox" class="log-box" @scroll="onLogScroll">{{ logText || '（无日志）' }}</pre>
  </component>
</template>

<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  embedded: Boolean,
  namespace: { type: String, required: true },
  podName: { type: String, required: true },
  containers: { type: Array, default: () => [] },
  initialContainer: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const container = ref('')
const follow = ref(true)
const tailLines = ref(500)
const logText = ref('')
const connectionStatus = ref('未连接')
const autoScroll = ref(true)
function onLogScroll() {
  const box = logBox.value
  if (box && box.scrollHeight - box.scrollTop - box.clientHeight > 40) autoScroll.value = false
}
onBeforeUnmount(stop)
const logBox = ref(null)
let abortCtrl = null

watch(() => props.modelValue, (v) => {
  visible.value = v
  if (v) {
    container.value = props.initialContainer || props.containers[0] || ''
    start()
  } else {
    stop()
  }
}, { immediate: true })
watch(visible, (v) => { if (!v) emit('update:modelValue', false) })

function stop() {
  if (abortCtrl) { abortCtrl.abort(); abortCtrl = null }
}

async function reload() {
  stop()
  logText.value = ''
  await nextTick()
  start()
}

async function start() {
  const controller = new AbortController()
  abortCtrl = controller
  connectionStatus.value = '连接中'
  const params = new URLSearchParams({
    namespace: props.namespace,
    name: props.podName,
    tailLines: String(tailLines.value),
    follow: String(follow.value),
  })
  if (container.value) params.set('container', container.value)

  try {
    const resp = await fetch('/api/pods/logs?' + params, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('kubeboard_token') },
      signal: controller.signal,
    })
    if (controller.signal.aborted) return
    if (!resp.ok) {
      connectionStatus.value = '连接失败'
      const err = await resp.json().catch(() => ({}))
      logText.value = '加载失败: ' + (err.error || resp.statusText)
      return
    }
    connectionStatus.value = '已连接'
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    const MAX = 200000
    for (;;) {
      const { done, value } = await reader.read()
      if (controller.signal.aborted) return
      if (done) { connectionStatus.value = '读取完成'; break }
      logText.value += decoder.decode(value, { stream: true })
      if (logText.value.length > MAX) logText.value = logText.value.slice(-MAX)
      await nextTick()
      if (autoScroll.value && logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
    }
  } catch (e) {
    if (!controller.signal.aborted) { connectionStatus.value = '连接中断'; logText.value += '\n[连接中断: ' + e.message + ']' }
  }
}
</script>

<style scoped>
.log-context { color: var(--ks-text-secondary); font-size: 12px; margin: 0 0 16px; overflow-wrap: anywhere; }
.log-toolbar { flex-wrap: wrap; display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.log-box {
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.5;
  padding: 12px;
  border-radius: 4px;
  height: 460px;
  overflow: auto;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
