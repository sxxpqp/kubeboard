<template>
  <el-drawer
    v-model="visible"
    :title="title"
    size="620px"
    :close-on-click-modal="true"
    @closed="$emit('update:modelValue', false)"
  >
    <div v-loading="loading" class="detail-body">
      <el-tabs v-model="activeTab">
        <!-- 概览：键值描述 -->
        <el-tab-pane label="概览" name="overview">
          <el-descriptions :column="1" border size="small" class="detail-desc">
            <el-descriptions-item
              v-for="f in fields" :key="f.key" :label="f.label"
            >
              <template v-if="f.type === 'status'">
                <span class="ks-status"><span class="dot" :class="rawValue(f.key)"></span>{{ rawValue(f.key) }}</span>
              </template>
              <template v-else-if="f.type === 'time'">{{ fmtTime(rawValue(f.key)) }}</template>
              <template v-else-if="f.type === 'tags'">
                <el-tag
                  v-for="(v, i) in asArray(rawValue(f.key))" :key="i"
                  size="small" type="info" style="margin: 0 4px 2px 0"
                >{{ v }}</el-tag>
                <span v-if="!asArray(rawValue(f.key)).length" class="muted">-</span>
              </template>
              <template v-else-if="f.type === 'list'">
                <div v-for="(v, i) in asArray(rawValue(f.key))" :key="i" class="mono-line">{{ v }}</div>
                <span v-if="!asArray(rawValue(f.key)).length" class="muted">-</span>
              </template>
              <template v-else-if="f.type === 'kv'">
                <div v-for="(v, k) in (rawValue(f.key) || {})" :key="k" class="mono-line">
                  <span class="kv-key">{{ k }}</span>={{ v }}
                </div>
                <span v-if="!Object.keys(rawValue(f.key) || {}).length" class="muted">-</span>
              </template>
              <template v-else>{{ rawValue(f.key) ?? '-' }}</template>
            </el-descriptions-item>
          </el-descriptions>
        </el-tab-pane>

        <!-- 标签 -->
        <el-tab-pane label="标签" name="labels">
          <div class="label-block">
            <div class="label-title">Labels</div>
            <div v-if="Object.keys(detail.labels || {}).length">
              <el-tag v-for="(v, k) in detail.labels" :key="k" size="small" style="margin: 0 6px 6px 0">
                {{ k }}={{ v }}
              </el-tag>
            </div>
            <div v-else class="muted">无标签</div>
          </div>
          <div class="label-block">
            <div class="label-title">Annotations</div>
            <div v-if="Object.keys(detail.annotations || {}).length">
              <el-tag v-for="(v, k) in detail.annotations" :key="k" size="small" type="info" style="margin: 0 6px 6px 0; max-width: 100%; overflow: hidden; text-overflow: ellipsis">
                {{ k }}={{ v }}
              </el-tag>
            </div>
            <div v-else class="muted">无注解</div>
          </div>
        </el-tab-pane>

        <!-- 事件 -->
        <el-tab-pane label="事件" name="events">
          <el-table :data="detail.events || []" size="small" v-loading="loadingEvents" max-height="480">
            <el-table-column label="类型" width="80">
              <template #default="{ row }">
                <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'" size="small">{{ row.type }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" width="120" />
            <el-table-column prop="message" label="消息" min-width="220" />
            <el-table-column label="时间" width="150">
              <template #default="{ row }">{{ fmtTime(row.lastSeen) }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <!-- YAML -->
        <el-tab-pane label="YAML" name="yaml">
          <el-button size="small" :icon="'Refresh'" @click="loadYaml" style="margin-bottom: 8px">刷新</el-button>
          <pre class="yaml-box">{{ yamlText || '加载中…' }}</pre>
        </el-tab-pane>

        <!-- 扩展 Tab（各页面自定义，如监控/日志） -->
        <el-tab-pane
          v-for="t in extraTabs" :key="t.name" :label="t.label" :name="t.name"
        >
          <slot :name="`tab-${t.name}`" :detail="detail" />
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <slot name="footer" :detail="detail" />
      <el-button @click="visible = false">关闭</el-button>
    </template>
  </el-drawer>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  title: { type: String, default: '详情' },
  kind: { type: String, required: true },       // k8s resource kind，用于拉 YAML
  namespace: { type: String, default: '' },
  name: { type: String, default: '' },
  fields: { type: Array, default: () => [] },   // [{key,label,type}]
  extraTabs: { type: Array, default: () => [] },// [{name,label}]
  loadDetail: { type: Function, default: null },// 自定义详情加载，返回对象
})
const emit = defineEmits(['update:modelValue'])

const visible = ref(false)
const activeTab = ref('overview')
const loading = ref(false)
const loadingEvents = ref(false)
const yamlText = ref('')
const detail = ref({})

watch(() => props.modelValue, async (v) => {
  visible.value = v
  if (!v) return
  activeTab.value = 'overview'
  yamlText.value = ''
  detail.value = {}
  await loadDetail()
})

watch(visible, (v) => { if (!v) emit('update:modelValue', false) })

function rawValue(key) {
  return detail.value[key]
}

function asArray(v) {
  if (Array.isArray(v)) return v
  if (v == null || v === '') return []
  return [v]
}

function fmtTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

async function loadDetail() {
  loading.value = true
  try {
    if (props.loadDetail) {
      detail.value = await props.loadDetail() || {}
    }
  } finally {
    loading.value = false
  }
  loadYaml()
  loadEvents()
}

async function loadYaml() {
  try {
    const params = new URLSearchParams({ kind: props.kind, name: props.name })
    if (props.namespace) params.set('namespace', props.namespace)
    const resp = await fetch('/api/resource?' + params, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('kubeboard_token') },
    })
    yamlText.value = resp.ok ? await resp.text() : '加载失败: ' + resp.status
  } catch (e) {
    yamlText.value = '加载失败: ' + e.message
  }
}

async function loadEvents() {
  loadingEvents.value = true
  try {
    const params = new URLSearchParams()
    if (props.namespace) params.set('namespace', props.namespace)
    const data = await fetch('/api/events?' + params, {
      headers: { Authorization: 'Bearer ' + localStorage.getItem('kubeboard_token') },
    }).then((r) => r.json())
    const nm = props.name
    detail.value.events = (data.items || []).filter(
      (e) => e.object && e.object.includes(nm)
    )
  } catch {
    detail.value.events = []
  } finally {
    loadingEvents.value = false
  }
}

defineExpose({ reload: loadDetail })
</script>

<style scoped>
.detail-body { min-height: 200px; }
.detail-desc :deep(.el-descriptions__label) { width: 130px; color: var(--ks-text-secondary); }
.mono-line { font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace; font-size: 12px; line-height: 1.7; }
.kv-key { color: var(--ks-text-secondary); }
.muted { color: var(--ks-text-secondary); font-size: 12px; }
.label-block { margin-bottom: 20px; }
.label-title { font-weight: 600; font-size: 13px; margin-bottom: 10px; }
.yaml-box {
  background: #1e1e1e; color: #d4d4d4;
  font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px; line-height: 1.6;
  padding: 12px; border-radius: 4px;
  max-height: 520px; overflow: auto; margin: 0;
  white-space: pre-wrap; word-break: break-all;
}
</style>
