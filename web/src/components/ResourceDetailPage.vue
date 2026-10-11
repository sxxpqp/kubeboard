<template>
  <div v-loading="loading" class="rd-page">
    <!-- 头部 -->
    <div class="rd-header">
      <el-button :icon="'Back'" @click="$router.back()">返回</el-button>
      <div class="rd-title">
        <span class="rd-kind">{{ kindLabel }}</span>
        <span class="rd-name">{{ name }}</span>
        <el-tag v-if="stateTag" :type="stateTagType" size="small" effect="dark">{{ stateTag }}</el-tag>
      </div>
      <div class="rd-ns" v-if="namespace">
        <span class="muted">项目</span>
        <el-link type="primary" @click="$router.push('/namespaces/' + namespace)">{{ namespace }}</el-link>
      </div>
      <div style="flex: 1"></div>
      <slot name="actions" :detail="detail" />
      <el-button :icon="'Refresh'" @click="load" />
    </div>

    <el-alert v-if="error" type="error" :closable="false" :title="error" style="margin-bottom: 16px" />

    <el-tabs v-model="activeTab" class="rd-tabs">
      <!-- 概览 -->
      <el-tab-pane label="概览" name="overview">
        <div class="rd-grid">
          <div class="rd-card">
            <div class="rd-card-title">基本信息</div>
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="名称">{{ name }}</el-descriptions-item>
              <el-descriptions-item label="类型">{{ kindLabel }}</el-descriptions-item>
              <el-descriptions-item label="项目">{{ namespace || '-' }}</el-descriptions-item>
              <el-descriptions-item label="创建时间">{{ fmtTime(meta.creationTime) }}</el-descriptions-item>
              <el-descriptions-item label="UID">
                <span class="mono">{{ meta.uid || '-' }}</span>
              </el-descriptions-item>
              <el-descriptions-item label="资源版本">{{ meta.resourceVersion || '-' }}</el-descriptions-item>
              <el-descriptions-item v-if="meta.generation" label="Generation">{{ meta.generation }}</el-descriptions-item>
              <el-descriptions-item label="存活时长">{{ age }}</el-descriptions-item>
            </el-descriptions>
            <slot name="overview-info" :detail="detail" />
          </div>

          <div class="rd-card">
            <div class="rd-card-title">标签 / 注解</div>
            <div class="label-section">
              <div class="sub-title">Labels</div>
              <div v-if="Object.keys(meta.labels || {}).length">
                <el-tag v-for="(v, k) in meta.labels" :key="k" size="small" style="margin: 0 6px 6px 0">{{ k }}={{ v }}</el-tag>
              </div>
              <span v-else class="muted">无</span>
            </div>
            <div class="label-section">
              <div class="sub-title">Annotations</div>
              <div v-if="Object.keys(meta.annotations || {}).length">
                <el-tag v-for="(v, k) in meta.annotations" :key="k" type="info" size="small" class="anno-tag">{{ k }}</el-tag>
              </div>
              <span v-else class="muted">无</span>
            </div>
            <div class="label-section" v-if="meta.ownerReferences && meta.ownerReferences.length">
              <div class="sub-title">所属（Owner）</div>
              <el-tag v-for="(o, i) in meta.ownerReferences" :key="i" type="warning" size="small" style="margin: 0 6px 6px 0"
                class="owner-tag" @click="gotoOwner(o)">
                {{ o.kind }}/{{ o.name }}
              </el-tag>
            </div>
          </div>
        </div>

        <!-- 类型专属摘要 -->
        <slot name="overview-extra" :detail="detail" />
      </el-tab-pane>

      <!-- 关联资源 -->
      <el-tab-pane label="关联资源" name="related">
        <div v-loading="loadingRelated">
          <el-alert v-if="!loadingRelated && !relatedGroups.length" type="info" :closable="false" title="未发现关联资源" />
          <div v-for="g in relatedGroups" :key="g.kind" class="rel-group">
            <div class="rel-title">{{ g.label }} <span class="muted">({{ g.items.length }})</span></div>
            <div class="rel-list">
              <el-tag v-for="it in g.items" :key="g.kind + it.namespace + it.name" class="rel-tag"
                :type="g.color" effect="plain" size="default" @click="gotoRelated(it)">
                <span class="rel-name">{{ it.name }}</span>
                <span v-if="it.namespace && g.kind !== 'pod' && g.kind !== 'node'" class="rel-ns">{{ it.namespace }}</span>
                <span v-if="it.extra" class="rel-extra">{{ it.extra }}</span>
              </el-tag>
            </div>
          </div>
        </div>
      </el-tab-pane>

      <!-- 事件（按 uid 精确绑定 + 可选聚合关联资源）-->
      <el-tab-pane name="events">
        <template #label>事件 <el-badge v-if="events.length" :value="events.length" type="primary" /></template>
        <div class="ev-bar">
          <el-switch v-model="includeRelated" size="small" active-text="包含关联资源事件" @change="loadEvents" />
          <span class="muted">共 {{ events.length }} 条</span>
        </div>
        <el-table :data="events" size="small" v-loading="loadingEvents" max-height="460">
          <el-table-column label="类型" width="80">
            <template #default="{ row }">
              <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'" size="small">{{ row.type }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="对象" width="200" v-if="includeRelated">
            <template #default="{ row }">
              <span class="mono-sm">{{ row.kind }}/{{ row.name }}</span>
            </template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" width="140" />
          <el-table-column prop="message" label="消息" min-width="240" />
          <el-table-column prop="count" label="次数" width="70" align="center" />
          <el-table-column label="最近发生" width="160">
            <template #default="{ row }">{{ fmtTime(row.lastSeen) }}</template>
          </el-table-column>
        </el-table>
      </el-tab-pane>

      <!-- 扩展 Tab -->
      <el-tab-pane v-for="t in extraTabs" :key="t.name" :label="t.label" :name="t.name">
        <slot :name="`tab-${t.name}`" :detail="detail" />
      </el-tab-pane>

      <!-- YAML -->
      <el-tab-pane label="YAML" name="yaml">
        <pre class="yaml-box">{{ detail.rawYaml || '加载中…' }}</pre>
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const props = defineProps({
  // 若传 kind 则用 prop，否则从路由参数取
  kind: { type: String, default: '' },
  name: { type: String, default: '' },
  namespace: { type: String, default: '' },
  kindLabel: { type: String, default: '' },
  extraTabs: { type: Array, default: () => [] },
})

const route = useRoute()
const router = useRouter()

const rKind = computed(() => props.kind || route.params.kind || route.meta.kind || '')
const rName = computed(() => props.name || route.params.name || '')
const rNs = computed(() => props.namespace || route.params.namespace || '')

const detail = ref({})
const loading = ref(false)
const error = ref('')
const activeTab = ref('overview')

const relatedItems = ref([])
const loadingRelated = ref(false)
const events = ref([])
const loadingEvents = ref(false)
const includeRelated = ref(false)

const meta = computed(() => detail.value.metadata || {})
const kindLabel = computed(() => props.kindLabel || detail.value.kind || rKind.value)

const KIND_LABELS = {
  pod: '容器组', node: '节点', service: '服务', ingress: '应用路由',
  replicaset: '副本集', pvc: '持久卷声明', pv: '持久卷', deployment: '部署',
  statefulset: '有状态副本集', daemonset: '守护进程集', configmap: '配置字典',
  secret: '***', job: '任务', cronjob: '定时任务', serviceaccount: '服务账号',
  storageclass: '存储类', networkpolicy: '网络策略', hpa: '弹性伸缩',
}
const KIND_COLORS = {
  pod: 'primary', node: 'success', service: 'warning', ingress: 'danger',
  replicaset: 'info', pvc: 'info', pv: 'info', configmap: 'info', secret: '***',
}

const stateTag = computed(() => {
  const s = detail.value.status || {}
  return s.phase || s.message || ''
})
const stateTagType = computed(() => {
  const p = stateTag.value
  if (['Running', 'Active', 'Bound', 'Healthy'].includes(p)) return 'success'
  if (['Pending', 'ContainerCreating'].includes(p)) return 'warning'
  if (['Failed', 'Error', 'Lost'].includes(p)) return 'danger'
  return 'info'
})

const age = computed(() => {
  const ts = meta.value.creationTime
  if (!ts) return '-'
  const sec = Math.floor(Date.now() / 1000) - ts
  if (sec < 60) return sec + ' 秒'
  if (sec < 3600) return Math.floor(sec / 60) + ' 分钟'
  if (sec < 86400) return Math.floor(sec / 3600) + ' 小时'
  return Math.floor(sec / 86400) + ' 天'
})

const relatedGroups = computed(() => {
  const map = {}
  for (const it of relatedItems.value) {
    const k = it.kind
    if (!map[k]) map[k] = { kind: k, label: KIND_LABELS[k] || k, color: KIND_COLORS[k] || 'info', items: [] }
    if (!map[k].items.some((x) => x.namespace === it.namespace && x.name === it.name)) map[k].items.push(it)
  }
  return Object.values(map)
})

function fmtTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false }) : '-'
}

function gotoOwner(o) {
  const k = (o.kind || '').toLowerCase()
  const routes = { pod: '/pods', deployment: '/workloads', replicaset: '/workloads', statefulset: '/workloads', daemonset: '/workloads', job: '/jobs', cronjob: '/cronjobs' }
  const path = routes[k]
  if (!path) return ElMessage.info(`暂不支持跳转到 ${o.kind}`)
  router.push({ path, query: { namespace: rNs.value, name: o.name } })
}

function gotoRelated(it) {
  const routes = {
    pod: '/pods', service: '/services', ingress: '/ingresses', node: '/nodes',
    pvc: '/pvcs', pv: '/pvs', configmap: '/configmaps', secret: '***',
    replicaset: '/workloads',
  }
  const path = routes[it.kind]
  if (!path) return ElMessage.info(`暂不支持跳转到 ${it.kind}`)
  const query = {}
  if (it.namespace) query.namespace = it.namespace
  router.push({ path, query })
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const params = { kind: rKind.value, name: rName.value }
    if (rNs.value) params.namespace = rNs.value
    detail.value = await http.get('/resources/detail', { params })
    loadRelated()
    loadEvents()
  } catch (e) {
    error.value = e.response?.data?.error || e?.message || '加载详情失败'
  } finally {
    loading.value = false
  }
}

async function loadRelated() {
  loadingRelated.value = true
  try {
    const params = { kind: rKind.value, name: rName.value }
    if (rNs.value) params.namespace = rNs.value
    const d = await http.get('/related', { params })
    relatedItems.value = d.items || []
  } catch {
    relatedItems.value = []
  } finally {
    loadingRelated.value = false
  }
}

async function loadEvents() {
  loadingEvents.value = true
  try {
    const params = { kind: rKind.value, name: rName.value }
    if (rNs.value) params.namespace = rNs.value
    params.includeRelated = includeRelated.value ? '1' : '0'
    const d = await http.get('/events/for', { params })
    events.value = d.items || []
  } catch {
    events.value = []
  } finally {
    loadingEvents.value = false
  }
}

onMounted(load)
defineExpose({ load })
</script>

<style scoped>
.rd-page { padding: 0; }
.rd-header { display: flex; align-items: center; gap: 14px; margin-bottom: 16px; }
.rd-title { display: flex; align-items: center; gap: 10px; }
.rd-kind { color: var(--ks-text-secondary); font-size: 14px; }
.rd-name { font-size: 18px; font-weight: 600; }
.rd-ns { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.rd-tabs { background: var(--ks-card-bg, #fff); border-radius: 8px; padding: 16px; }
.rd-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
@media (max-width: 1100px) { .rd-grid { grid-template-columns: 1fr; } }
.rd-card { background: var(--ks-bg-light, #f9fbfd); border-radius: 8px; padding: 16px; }
.rd-card-title { font-weight: 600; font-size: 14px; margin-bottom: 12px; }
.label-section { margin-bottom: 14px; }
.sub-title { font-size: 12px; color: var(--ks-text-secondary); margin-bottom: 6px; }
.mono { font-family: monospace; font-size: 12px; }
.mono-sm { font-family: monospace; font-size: 12px; }
.anno-tag { max-width: 100%; overflow: hidden; text-overflow: ellipsis; margin: 0 6px 6px 0; }
.owner-tag { cursor: pointer; }
.owner-tag:hover { opacity: 0.8; }
.muted { color: var(--ks-text-secondary); font-size: 12px; }
.rel-group { margin-bottom: 18px; }
.rel-title { font-weight: 600; font-size: 13px; margin-bottom: 8px; }
.rel-list { display: flex; flex-wrap: wrap; gap: 8px; }
.rel-tag { cursor: pointer; padding: 4px 10px; }
.rel-tag:hover { opacity: 0.8; }
.rel-name { font-weight: 500; }
.rel-ns { color: var(--ks-text-secondary); font-size: 11px; margin-left: 6px; }
.rel-extra { color: var(--ks-text-secondary); font-size: 11px; margin-left: 6px; font-style: italic; }
.ev-bar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; }
.yaml-box {
  background: #1e1e1e; color: #d4d4d4;
  font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px; line-height: 1.6; padding: 14px; border-radius: 6px;
  max-height: 620px; overflow: auto; margin: 0; white-space: pre-wrap; word-break: break-all;
}
</style>
