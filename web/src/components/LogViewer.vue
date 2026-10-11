<template>
  <el-dialog
    v-model="visible"
    :title="`日志: ${podName}`"
    width="900px"
    :close-on-click-modal="false"
    @closed="$emit('update:modelValue', false)"
  >
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
      <div style="flex: 1"></div>
      <el-button size="small" :icon="'Refresh'" @click="reload" />
    </div>
    <pre ref="logBox" class="log-box">{{ logText || '（无日志）' }}</pre>
  </el-dialog>
</template>

<script setup>
import { nextTick, ref, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  namespace: { type: String, required: true },
  podName: { type: String, required: true },
  containers: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const container = ref('')
const follow = ref(true)
const tailLines = ref(500)
const logText = ref('')
const logBox = ref(null)
let abortCtrl = null

watch(() => props.modelValue, (v) => {
  visible.value = v
  if (v) {
    container.value = props.containers[0] || ''
    start()
  } else {
    stop()
  }
})
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
  abortCtrl = new AbortController()
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
      signal: abortCtrl.signal,
    })
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({}))
      logText.value = '加载失败: ' + (err.error || resp.statusText)
      return
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    const MAX = 200000
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      logText.value += decoder.decode(value, { stream: true })
      if (logText.value.length > MAX) logText.value = logText.value.slice(-MAX)
      await nextTick()
      if (logBox.value) logBox.value.scrollTop = logBox.value.scrollHeight
    }
  } catch (e) {
    if (e.name !== 'AbortError') logText.value += '\n[连接中断: ' + e.message + ']'
  }
}
</script>

<style scoped>
.log-toolbar { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
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
