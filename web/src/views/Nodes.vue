<template>
  <div class="ks-card">
    <div class="ks-toolbar">
      <el-input v-model="keyword" placeholder="按名称搜索" clearable style="width: 240px" :prefix-icon="'Search'" />
      <div style="flex: 1"></div>
      <el-button :icon="'DataLine'" @click="monitorVisible = true">监控</el-button>
      <el-button :icon="'Refresh'" @click="load" />
    </div>
    <el-table
      :data="filtered" v-loading="loading" stripe class="ks-table"
      highlight-current-row :row-style="{ cursor: 'pointer' }" @row-click="openDetail"
    >
      <el-table-column prop="name" label="名称" min-width="160">
        <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
      </el-table-column>
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <span class="ks-status"><span class="dot" :class="row.status"></span>{{ row.status }}</span>
        </template>
      </el-table-column>
      <el-table-column label="调度" width="90">
        <template #default="{ row }">
          <el-tag size="small" :type="row.unschedulable ? 'danger' : 'success'">
            {{ row.unschedulable ? '已封锁' : '可调度' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="角色" width="140">
        <template #default="{ row }">
          <el-tag v-for="r in row.roles" :key="r" size="small" style="margin-right: 4px">{{ r }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="internalIP" label="内部 IP" width="130" />
      <el-table-column prop="version" label="Kubelet 版本" width="150" />
      <el-table-column prop="os" label="操作系统" min-width="180" />
      <el-table-column prop="cpu" label="CPU" width="90" />
      <el-table-column label="内存" width="110">
        <template #default="{ row }">{{ formatMem(row.memory) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" size="small" @click.stop="openDetail(row)">详情</el-button>
          <el-button link type="primary" size="small" @click.stop="viewPods(row)">容器组</el-button>
          <el-button
            link :type="row.unschedulable ? 'success' : 'warning'" size="small"
            @click.stop="toggleCordon(row)"
          >{{ row.unschedulable ? '解除封锁' : '封锁' }}</el-button>
        </template>
      </el-table-column>
    </el-table>
    <div class="table-footer">共 {{ filtered.length }} 项</div>

    <!-- 全节点监控 -->
    <el-drawer v-model="monitorVisible" title="节点资源监控" size="820px">
      <div v-loading="metricsLoading">
        <el-alert
          v-if="metricsAvailable === false"
          type="warning" :closable="false"
          title="集群未安装 metrics-server，无法获取资源指标"
          style="margin-bottom: 16px"
        />
        <template v-else>
          <div class="metric-head"><span>CPU 使用率</span><span class="muted">实时快照</span></div>
          <EChart :option="clusterCpuOption" :height="240" />
          <div class="metric-head" style="margin-top: 20px"><span>内存使用率</span><span class="muted">实时快照</span></div>
          <EChart :option="clusterMemOption" :height="240" />
        </template>
      </div>
    </el-drawer>

    <!-- 节点详情抽屉 -->
    <ResourceDetail
      v-model="detailVisible"
      :title="`节点: ${detailRow.name}`"
      kind="node" :name="detailRow.name || ''"
      :fields="detailFields" :load-detail="loadNodeDetail"
      :extra-tabs="[{ name: 'monitor', label: '监控' }]"
    >
      <template #tab-monitor>
        <div v-if="nodeMetric">
          <div class="metric-head">
            <span>CPU 用量</span>
            <b>{{ nodeMetric.cpuCores }}（{{ nodeMetric.cpuPercent }}%）</b>
          </div>
          <EChart :option="cpuOption" :height="140" />
          <div class="metric-head" style="margin-top: 16px">
            <span>内存用量</span>
            <b>{{ nodeMetric.memBytes }}（{{ nodeMetric.memPercent }}%）</b>
          </div>
          <EChart :option="memOption" :height="140" />
        </div>
        <el-empty v-else description="该节点暂无指标数据" />
      </template>
      <template #footer>
        <el-button
          :type="detailRow.unschedulable ? 'success' : 'warning'"
          @click="toggleCordon(detailRow); detailVisible = false"
        >{{ detailRow.unschedulable ? '解除封锁' : '封锁节点' }}</el-button>
      </template>
    </ResourceDetail>

    <!-- 节点上的 Pod -->
    <el-drawer v-model="podsVisible" :title="`节点容器组: ${podsNode}`" size="760px">
      <el-table :data="nodePods" v-loading="podsLoading" size="small" max-height="600">
        <el-table-column prop="namespace" label="项目" width="140" />
        <el-table-column prop="name" label="名称" min-width="240" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <span class="ks-status"><span class="dot" :class="row.status"></span>{{ row.status }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip" label="Pod IP" width="130" />
      </el-table>
    </el-drawer>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import ResourceDetail from '../components/ResourceDetail.vue'
import EChart from '../components/EChart.vue'
import { useNodeMetrics } from '../composables/useMetrics'

const items = ref([])
const loading = ref(true)
const keyword = ref('')

const detailVisible = ref(false)
const detailRow = ref({})
const podsVisible = ref(false)
const podsLoading = ref(false)
const podsNode = ref('')
const nodePods = ref([])

const monitorVisible = ref(false)
const { items: nodeMetrics, available: metricsAvailable, loading: metricsLoading, load: loadNodeMetrics } = useNodeMetrics()

// 当前详情节点的指标
const nodeMetric = computed(() =>
  nodeMetrics.value.find((n) => n.name === detailRow.value.name) || null
)

// 单节点 CPU/内存 环形图
function gaugeOption(value, label, color) {
  return {
    series: [{
      type: 'gauge', radius: '90%', startAngle: 210, endAngle: -30,
      min: 0, max: 100, splitNumber: 5,
      axisLine: { lineStyle: { width: 12, color: [[value / 100, color], [1, '#eef1f6']] } },
      pointer: { width: 4, length: '60%' },
      axisTick: { show: false }, splitLine: { show: false }, axisLabel: { show: false },
      detail: { valueAnimation: true, formatter: '{value}%', fontSize: 20, offsetCenter: [0, '70%'] },
      data: [{ value, name: label }],
      title: { fontSize: 12, offsetCenter: [0, '100%'], color: '#79879c' },
    }],
  }
}
const cpuOption = computed(() => gaugeOption(nodeMetric.value?.cpuPercent || 0, 'CPU', '#0084ff'))
const memOption = computed(() => gaugeOption(nodeMetric.value?.memPercent || 0, '内存', '#47c8bf'))

// 全集群柱状对比
function barOption(percentKey, cpuVal, label) {
  return {
    tooltip: { trigger: 'axis' },
    grid: { left: 50, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: nodeMetrics.value.map((n) => n.name), axisLabel: { fontSize: 11 } },
    yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%' } },
    series: [{
      type: 'bar', barWidth: '45%',
      data: nodeMetrics.value.map((n) => n[percentKey]),
      label: { show: true, position: 'top', formatter: '{c}%', fontSize: 11 },
      itemStyle: { borderRadius: [3, 3, 0, 0], color: '#0084ff' },
    }],
  }
}
const clusterCpuOption = computed(() => barOption('cpuPercent', 'cpuCores', 'CPU'))
const clusterMemOption = computed(() => barOption('memPercent', 'memBytes', '内存'))

const detailFields = [
  { key: 'name', label: '名称' },
  { key: 'status', label: '状态', type: 'status' },
  { key: 'roles', label: '角色', type: 'tags' },
  { key: 'internalIP', label: '内部 IP' },
  { key: 'version', label: 'Kubelet 版本' },
  { key: 'os', label: '操作系统' },
  { key: 'kernel', label: '内核版本' },
  { key: 'cpu', label: 'CPU 容量' },
  { key: 'memory', label: '内存容量' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.includes(keyword.value))
)

function formatMem(mem) {
  const n = parseInt(mem)
  if (isNaN(n)) return mem
  return (n / (1024 * 1024 * 1024)).toFixed(1) + ' Gi'
}

function openDetail(row) {
  detailRow.value = row
  detailVisible.value = true
}

async function loadNodeDetail() {
  // 列表已含核心字段，直接复用
  return detailRow.value
}

async function viewPods(row) {
  podsNode.value = row.name
  podsVisible.value = true
  podsLoading.value = true
  nodePods.value = []
  try {
    const data = await http.get('/pods', { params: { node: row.name } })
    nodePods.value = data.items || []
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '查询失败')
  } finally {
    podsLoading.value = false
  }
}

async function toggleCordon(row) {
  const action = row.unschedulable ? '解除封锁' : '封锁'
  try {
    await ElMessageBox.confirm(
      `确定对节点 ${row.name} 执行「${action}」吗？`,
      '节点调度', { type: 'warning' }
    )
  } catch { return }
  try {
    await http.post('/nodes/cordon', { name: row.name, unschedulable: !row.unschedulable })
    ElMessage.success(`${action}成功`)
    load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function load() {
  loading.value = true
  try {
    const data = await http.get('/nodes')
    items.value = data.items
    await loadNodeMetrics()
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
.metric-head {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; font-weight: 600; margin-bottom: 8px;
}
.muted { color: var(--ks-text-secondary); font-weight: 400; font-size: 12px; }
</style>
