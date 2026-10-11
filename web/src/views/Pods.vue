<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-select v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 170px">
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <el-button :icon="'Refresh'" @click="load" />
      </div>
      <el-table
        :data="filtered" v-loading="loading" stripe class="ks-table"
        highlight-current-row :row-style="{ cursor: 'pointer' }" @row-click="openDetail"
      >
        <el-table-column prop="namespace" label="项目" width="130" sortable />
        <el-table-column prop="name" label="名称" min-width="200" sortable>
          <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <span class="ks-status"><span class="dot" :class="row.status"></span>{{ row.status }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="restarts" label="重启" width="80" align="center" sortable />
        <el-table-column prop="node" label="节点" width="150" />
        <el-table-column prop="ip" label="Pod IP" width="130" />
        <el-table-column label="CPU" width="100">
          <template #default="{ row }">
            <span class="metric-text">{{ podMetric(row)?.cpuCores || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="内存" width="110">
          <template #default="{ row }">
            <span class="metric-text">{{ podMetric(row)?.memBytes || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="200">
          <template #default="{ row }">
            <div v-for="img in row.images" :key="img" class="image-text">{{ img }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click.stop="openDetail(row)">详情</el-button>
            <el-button link type="primary" size="small" @click.stop="openLogs(row)">日志</el-button>
            <el-button link type="primary" size="small" @click="openTerm(row)">终端</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row.name, row.namespace)">YAML</el-button>
            <el-button link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <ResourceDetail
      v-model="detailVisible"
      :title="`容器组: ${detailRow.namespace}/${detailRow.name}`"
      kind="pod" :namespace="detailRow.namespace || ''" :name="detailRow.name || ''"
      :fields="detailFields" :load-detail="loadPodDetail"
      :extra-tabs="[{ name: 'monitor', label: '监控' }]"
    >
      <template #tab-monitor>
        <div v-if="podMetric(detailRow)">
          <div class="metric-head"><span>CPU 用量</span><b>{{ podMetric(detailRow).cpuCores }}</b></div>
          <div class="metric-head" style="margin-top: 12px"><span>内存用量</span><b>{{ podMetric(detailRow).memBytes }}</b></div>
          <el-divider />
          <div class="metric-head"><span>容器明细</span></div>
          <el-table :data="podMetric(detailRow).containers || []" size="small">
            <el-table-column prop="name" label="容器" min-width="160" />
            <el-table-column prop="cpuCores" label="CPU" width="120" />
            <el-table-column prop="memBytes" label="内存" width="140" />
          </el-table>
        </div>
        <el-empty v-else description="该 Pod 暂无指标数据" />
      </template>
    </ResourceDetail>

    <YamlDialog v-model="dialogVisible" :mode="dialogMode" kind="pod"
      :namespace="currentNamespace" :name="currentName" :template="currentTemplate" @saved="load" />
    <LogViewer v-model="logVisible" :namespace="currentNamespace" :pod-name="currentName" :containers="containers" />
    <TerminalView v-model="termVisible" :namespace="currentNamespace" :pod-name="currentName" :containers="containers" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from '../components/YamlDialog.vue'
import LogViewer from '../components/LogViewer.vue'
import TerminalView from '../components/TerminalView.vue'
import ResourceDetail from '../components/ResourceDetail.vue'
import { usePodMetrics } from '../composables/useMetrics'

const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)
const logVisible = ref(false)
const termVisible = ref(false)
const containers = ref([])
const detailVisible = ref(false)
const detailRow = ref({})
const { items: podMetricsList, load: loadPodMetrics } = usePodMetrics()

const podMetricsMap = computed(() => {
  const m = {}
  for (const p of podMetricsList.value) m[p.namespace + '/' + p.name] = p
  return m
})
function podMetric(row) {
  return podMetricsMap.value[(row.namespace || '') + '/' + (row.name || '')] || null
}

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'status', label: '状态', type: 'status' },
  { key: 'node', label: '所在节点' },
  { key: 'ip', label: 'Pod IP' },
  { key: 'restarts', label: '重启次数' },
  { key: 'containerNames', label: '容器', type: 'tags' },
  { key: 'images', label: '镜像', type: 'list' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

function openDetail(row) {
  detailRow.value = row
  detailVisible.value = true
}
async function loadPodDetail() {
  return detailRow.value
}

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.toLowerCase().includes(keyword.value.toLowerCase()))
)

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openEdit, remove,
} = useResourceCRUD('pod', load)

function openLogs(row) {
  currentNamespace.value = row.namespace
  currentName.value = row.name
  containers.value = row.containerNames || []
  logVisible.value = true
}

function openTerm(row) {
  currentNamespace.value = row.namespace
  currentName.value = row.name
  containers.value = row.containerNames || []
  termVisible.value = true
}

async function load() {
  loading.value = true
  try {
    const params = {}
    if (namespace.value) params.namespace = namespace.value
    const data = await http.get('/pods', { params })
    items.value = data.items
    await loadPodMetrics(namespace.value)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  try {
    const nsData = await http.get('/namespaces')
    namespaces.value = nsData.items.map((i) => i.name)
  } catch { /* ignore */ }
  await load()
})
</script>

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.image-text { font-size: 12px; color: var(--ks-text-secondary); line-height: 1.8; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
.metric-text { font-size: 12px; color: var(--ks-text-secondary); }
.metric-head { display: flex; justify-content: space-between; align-items: center; font-size: 13px; font-weight: 600; }
</style>
