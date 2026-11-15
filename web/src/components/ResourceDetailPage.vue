<template>
  <div v-loading="loading" class="rd-page" :class="{ 'rd-wide': wide }">
    <aside class="rd-summary"><div class="summary-icon">{{ kindLabel.slice(0, 1) }}</div><span class="muted">{{ kindLabel }}</span><h1>{{ rName }}</h1><el-tag v-if="stateTag" :type="stateTagType">{{ stateTag }}</el-tag><dl><dt>项目</dt><dd>{{ rNs || '集群资源' }}</dd><dt>创建时间</dt><dd>{{ fmtTime(meta.creationTime) }}</dd><dt>存活时长</dt><dd>{{ age }}</dd><template v-for="attribute in summaryAttributes" :key="attribute.label"><dt>{{ attribute.label }}</dt><dd>{{ attribute.value }}</dd></template></dl><p class="muted">通过右侧页签查看资源状态、关联资源及事件。</p></aside>
    <!-- 头部 -->
    <div class="rd-header">
      <el-button :icon="'Back'" @click="$router.back()">返回</el-button>
      <div class="rd-title">
        <span class="rd-kind">{{ kindLabel }}</span>
        <span class="rd-name">{{ rName }}</span>
        <el-tag v-if="stateTag" :type="stateTagType" size="small" effect="dark">{{ stateTag }}</el-tag>
      </div>
      <div class="rd-ns" v-if="rNs">
        <span class="muted">项目</span>
        <el-link type="primary" @click="$router.push('/namespaces/' + rNs)">{{ rNs }}</el-link>
      </div>
      <div style="flex: 1"></div>
      <slot name="actions" :detail="detail" />
      <el-button v-if="!$slots['operation-bar']" @click="openEdit(rName, rNs)">编辑 YAML</el-button>
      <el-button v-if="!$slots['operation-bar']" type="danger" plain @click="remove(rName, rNs)">删除</el-button>
      <el-button :icon="'Refresh'" @click="load" />
    </div>

    <div v-if="$slots['operation-bar']" class="rd-operation-bar"><slot name="operation-bar" :detail="detail" :reload="load" /></div>
    <el-alert v-if="error" type="error" :closable="false" :title="error" style="margin-bottom: 16px" />

    <YamlDialog v-model="dialogVisible" :mode="dialogMode" :kind="rKind" :namespace="currentNamespace" :name="currentName" @saved="load" />
    <el-tabs v-model="activeTab" class="rd-tabs">
      <el-tab-pane v-for="t in extraTabs.filter(t => t.name === 'runtime')" :key="t.name" :label="t.label" :name="t.name"><slot :name="`tab-${t.name}`" :detail="detail" /></el-tab-pane>
      <!-- 概览 -->
      <el-tab-pane label="资源状态" name="overview">
        <ResourceContent :detail="detail" />
        <!-- 类型专属摘要 -->
        <slot name="overview-extra" :detail="detail" />

        <!-- 状态状况（conditions）：协调状态一览，含代数偏差告警 -->
        <div v-if="conditions.length" class="rd-card" style="margin-bottom: 16px">
          <div class="rd-card-title">
            状态状况
            <el-tag v-if="staleGeneration" type="danger" size="small" style="margin-left: 8px">
              代数偏差：已观测 {{ detail.status?.observation?.observedGeneration }} / 期望 {{ detail.status?.observation?.generation }}
            </el-tag>
          </div>
          <el-table :data="conditions" size="small">
            <el-table-column prop="type" label="类型" width="180" />
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="conditionType(row)" size="small">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" width="180" />
            <el-table-column prop="message" label="消息" min-width="240" />
            <el-table-column label="最近更新" width="160">
              <template #default="{ row }">{{ fmtTime(row.lastUpdate || row.lastTransitionTime) }}</template>
            </el-table-column>
          </el-table>
        </div>
      </el-tab-pane>

      <el-tab-pane label="元数据" name="metadata">
        <div class="rd-grid">
          <div class="rd-card">
            <div class="rd-card-title">基本信息</div>
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="名称">{{ rName }}</el-descriptions-item>
              <el-descriptions-item label="类型">{{ kindLabel }}</el-descriptions-item>
              <el-descriptions-item label="项目">{{ rNs || '-' }}</el-descriptions-item>
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
              <div v-if="Object.keys(meta.annotations || {}).length" class="anno-list">
                <div v-for="(v, k) in meta.annotations" :key="k" class="anno-row">
                  <span class="anno-key">{{ k }}</span>
                  <span class="anno-val">{{ v }}</span>
                </div>
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

      </el-tab-pane>
      <!-- 关联资源 -->
      <el-tab-pane label="关联资源" name="related">
        <div v-loading="loadingRelated"><el-alert v-if="relatedError" :title="relatedError" type="error" :closable="false"><el-button link @click="loadRelated">重试</el-button></el-alert>
          <el-alert v-if="!loadingRelated && !relatedError && !relatedGroups.length" type="info" :closable="false" title="未发现关联资源" />
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
        <el-alert v-if="eventsError" :title="eventsError" type="error" :closable="false"><el-button link @click="loadEvents">重试</el-button></el-alert>
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
      <el-tab-pane v-for="t in extraTabs.filter(t => t.name !== 'runtime')" :key="t.name" :label="t.label" :name="t.name">
        <slot :name="`tab-${t.name}`" :detail="detail" />
      </el-tab-pane>

      <el-tab-pane v-if="envContainers.length" label="环境变量" name="environment">
        <section v-for="ctr in envContainers" :key="ctr.name" class="rd-card" style="margin-bottom: 16px">
          <div class="rd-card-title">{{ ctr.name }}</div>
          <el-table :data="ctr.rows" empty-text="未配置环境变量"><el-table-column prop="name" label="变量名" min-width="180" /><el-table-column prop="value" label="值 / 引用（Secret 仅展示引用）" min-width="260" /></el-table>
        </section>
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
import http from '../api/http'
import ResourceContent from './ResourceContent.vue'
import YamlDialog from './YamlDialog.vue'
import { useResourceCRUD } from '../api/crud'
import { resourceObject, environmentContainers, conditionType } from '../utils/resourceContent'

const props = defineProps({
  // 若传 kind 则用 prop，否则从路由参数取
  wide: Boolean,
  defaultTab: { type: String, default: 'overview' },
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
const rNs = computed(() => props.namespace || (route.params.namespace === '_' ? '' : route.params.namespace) || '')

const detail = ref({})
const loading = ref(false)
const error = ref('')
const initialTabs = ['overview', 'related', 'metadata', 'events', 'environment', 'yaml', ...props.extraTabs.map(tab => tab.name)]
const activeTab = ref(initialTabs.includes(route.query.tab) ? route.query.tab : props.defaultTab)

const { dialogVisible, dialogMode, currentNamespace, currentName, openEdit, remove } = useResourceCRUD(rKind, () => router.back())
const envContainers = computed(() => environmentContainers(resourceObject(detail.value)))
const relatedItems = ref([])
const loadingRelated = ref(false)
const relatedError = ref('')
const eventsError = ref('')
const events = ref([])
const loadingEvents = ref(false)
const includeRelated = ref(false)

const meta = computed(() => detail.value.metadata || {})
const conditions = computed(() => detail.value.status?.conditions || resourceObject(detail.value).status?.conditions || [])
// 代数偏差：spec.generation > status.observedGeneration 说明控制器还没处理完最新变更
const staleGeneration = computed(() => {
  const o = detail.value.status?.observation
  return o && o.generation > o.observedGeneration
})
const kindLabel = computed(() => props.kindLabel || KIND_LABELS[rKind.value] || detail.value.kind || rKind.value)

const KIND_LABELS = {
  pod: '容器组', node: '节点', service: '服务', ingress: '应用路由',
  replicaset: '副本集', pvc: '持久卷声明', pv: '持久卷', deployment: '部署',
  statefulset: '有状态副本集', daemonset: '守护进程集', configmap: '配置字典',
  secret: '保密字典', job: '任务', cronjob: '定时任务', serviceaccount: '服务账号',
  storageclass: '存储类', networkpolicy: '网络策略', hpa: '弹性伸缩',
}
const KIND_COLORS = {
  pod: 'primary', node: 'success', service: 'warning', ingress: 'danger',
  replicaset: 'info', pvc: 'info', pv: 'info', configmap: 'info', secret: 'danger',
}

const summaryAttributes = computed(() => {
  const spec = detail.value.spec || {}
  const fields = {
    pvc: [['存储类', 'storageClass'], ['访问模式', 'accessModes'], ['绑定卷', 'volumeName']],
    pv: [['存储类', 'storageClass'], ['访问模式', 'accessModes']],
    pod: [['所在节点', 'nodeName'], ['服务账号', 'serviceAccount'], ['重启策略', 'restartPolicy']],
    service: [['服务类型', 'type'], ['集群 IP', 'clusterIP']],
    deployment: [['更新策略', 'strategy'], ['期望副本', 'replicas']],
    statefulset: [['期望副本', 'replicas']],
  }[rKind.value] || []
  return fields.map(([label, key]) => ({ label, value: Array.isArray(spec[key]) ? spec[key].join(', ') : spec[key] ?? '—' }))
})

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
  return ts ? new Date(typeof ts === 'number' ? ts * 1000 : ts).toLocaleString('zh-CN', { hour12: false }) : '-'
}

function gotoObject(kind, name, ns) {
  const k = String(kind || '').toLowerCase()
  const path = ['deployment', 'statefulset', 'daemonset'].includes(k)
    ? `/workloads/${k}/${encodeURIComponent(ns || '_')}/${encodeURIComponent(name)}`
    : `/detail/${k}/${encodeURIComponent(ns || '_')}/${encodeURIComponent(name)}`
  router.push(path)
}
function gotoOwner(owner) { gotoObject(owner.kind, owner.name, rNs.value) }
function gotoRelated(item) { gotoObject(item.kind, item.name, item.namespace) }

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
  relatedError.value = ''
  loadingRelated.value = true
  try {
    const params = { kind: rKind.value, name: rName.value }
    if (rNs.value) params.namespace = rNs.value
    const d = await http.get('/related', { params })
    relatedItems.value = d.items || []
  } catch (e) {
    relatedError.value = '关联资源加载失败：' + (e?.message || '请重试')
  } finally {
    loadingRelated.value = false
  }
}

async function loadEvents() {
  eventsError.value = ''
  loadingEvents.value = true
  try {
    const params = { kind: rKind.value, name: rName.value }
    if (rNs.value) params.namespace = rNs.value
    params.includeRelated = includeRelated.value ? '1' : '0'
    const d = await http.get('/events/for', { params })
    events.value = d.items || []
  } catch (e) {
    eventsError.value = '事件加载失败：' + (e?.message || '请重试')
  } finally {
    loadingEvents.value = false
  }
}

onMounted(load)
defineExpose({ load })
</script>

<style scoped>
.rd-operation-bar { grid-column: 2; padding: 16px; background: var(--ks-card-bg, #fff); border: 1px solid var(--ks-border); border-radius: 6px; }
.rd-page { padding: 0; display: grid; grid-template-columns: 260px minmax(0, 1fr); gap: 20px; align-items: start; }
.rd-summary { grid-row: 1 / span 4; padding: 24px; background: white; border: 1px solid var(--ks-border); border-radius: 8px; position: sticky; top: 0; overflow-wrap: anywhere; }
.summary-icon { width: 48px; height: 48px; background: #eaf6ef; color: #23865b; border-radius: 12px; display: grid; place-items: center; font-size: 22px; margin-bottom: 20px; }
.rd-summary h1 { font-size: 22px; margin: 10px 0 16px; }
.rd-summary dl { border-top: 1px solid var(--ks-border); margin-top: 24px; padding-top: 8px; }
.rd-summary dt { font-size: 12px; color: var(--ks-text-secondary); margin-top: 18px; }
.rd-summary dd { margin: 6px 0; font-size: 13px; }
.rd-tabs { min-width: 0; grid-column: 2; }
@media(max-width: 1100px) { .rd-page { grid-template-columns: 1fr; } .rd-summary { grid-row: auto; position: static; } .rd-summary dl, .rd-summary p { display: none; } .rd-header, .rd-tabs { grid-column: 1; } }
.rd-page > .el-alert { grid-column: 2; }
.rd-header { grid-column: 2; display: flex; align-items: center; gap: 14px; margin-bottom: 16px; }
.rd-title { display: flex; align-items: center; gap: 10px; }
.rd-kind { color: var(--ks-text-secondary); font-size: 14px; }
.rd-name { font-size: 18px; font-weight: 600; }
.rd-ns { display: flex; align-items: center; gap: 6px; font-size: 13px; }
.rd-tabs { background: var(--ks-card-bg, #fff); border-radius: 8px; padding: 16px; }
.rd-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 16px; }
@media (max-width: 1100px) { .rd-grid { grid-template-columns: 1fr; } .rd-header, .rd-tabs, .rd-page > .el-alert, .rd-operation-bar { grid-column: 1; } }
.rd-card { background: var(--ks-bg-light, #f9fbfd); border-radius: 8px; padding: 16px; }
.rd-card-title { font-weight: 600; font-size: 14px; margin-bottom: 12px; }
.label-section { margin-bottom: 14px; }
.anno-list { max-height: 220px; overflow-y: auto; }
.anno-row { display: flex; gap: 8px; font-size: 12px; padding: 3px 0; border-bottom: 1px dashed var(--el-border-color-lighter, #eee); }
.anno-key { color: var(--ks-primary); flex-shrink: 0; word-break: break-all; }
.anno-val { color: var(--ks-text-secondary); word-break: break-all; }
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
.rd-page.rd-wide { grid-template-columns: minmax(0, 1fr); }
.rd-wide > .rd-summary { display: none; }
.rd-wide > .rd-header, .rd-wide > .rd-tabs, .rd-wide > .rd-operation-bar, .rd-wide > .el-alert { grid-column: 1; }
</style>
