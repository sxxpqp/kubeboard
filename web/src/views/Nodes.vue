<template>
  <div class="ks-card">
    <el-alert v-if="error" :title="error" type="error" :closable="false"><el-button link @click="load">重试</el-button></el-alert>
    <div class="ks-toolbar">
      <el-input v-model="keyword" placeholder="按名称搜索" clearable style="width: 240px" :prefix-icon="'Search'" />
      <el-select v-model="status" placeholder="全部状态" clearable style="width:160px"><el-option v-for="value in statusOptions" :key="value" :label="value" :value="value" /></el-select>
      <div style="flex: 1"></div>
      <el-button :icon="'DataLine'" @click="monitorVisible = true">监控</el-button>
      <el-button :icon="'Refresh'" @click="load" />
    </div>
    <el-table
      :data="paged" v-loading="loading" stripe class="ks-table"
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
    <div class="table-footer"><span>共 {{ filtered.length }} 项</span><el-pagination v-model:current-page="page" :page-size="20" :total="filtered.length" layout="prev,pager,next" /></div>

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


    <!-- 节点上的 Pod -->
    <el-drawer v-model="podsVisible" :title="`节点容器组: ${podsNode}`" size="760px">
      <el-table :data="nodePods" v-loading="podsLoading" size="small" max-height="600">
        <el-table-column prop="namespace" label="项目" width="140" />
        <el-table-column label="名称" min-width="240"><template #default="{row}"><router-link :to="`/detail/pod/${row.namespace}/${row.name}`">{{row.name}}</router-link></template></el-table-column>
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
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useRouter, useRoute } from 'vue-router'
import EChart from '../components/EChart.vue'
import { useNodeMetrics } from '../composables/useMetrics'

const items = ref([])
const loading = ref(true)
const route=useRoute()
const keyword = ref(String(route.query.search||'')),status=ref(String(route.query.status||'')),error=ref(''),page=ref(1)
const statusOptions=computed(()=>[...new Set(items.value.map(i=>i.status).filter(Boolean))])
const paged=computed(()=>filtered.value.slice((page.value-1)*20,page.value*20))
watch([keyword,status],()=>{page.value=1;router.replace({query:{...route.query,search:keyword.value||undefined,status:status.value||undefined}})})

const router = useRouter()
const podsVisible = ref(false)
const podsLoading = ref(false)
const podsNode = ref('')
const nodePods = ref([])

const monitorVisible = ref(false)
const { items: nodeMetrics, available: metricsAvailable, loading: metricsLoading, load: loadNodeMetrics } = useNodeMetrics()

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



const filtered = computed(() =>
  items.value.filter(i=>(!keyword.value||i.name.toLowerCase().includes(keyword.value.toLowerCase()))&&(!status.value||i.status===status.value))
)

function formatMem(mem) {
  const n = parseInt(mem)
  if (isNaN(n)) return mem
  return (n / (1024 * 1024 * 1024)).toFixed(1) + ' Gi'
}

function openDetail(row) {
  router.push(`/detail/node/${row.namespace || '_'}/${encodeURIComponent(row.name)}`)
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
    error.value=''
    const data = await http.get('/nodes')
    items.value = data.items
    page.value=Math.min(page.value,Math.max(1,Math.ceil(filtered.value.length/20)))
    await loadNodeMetrics()
  } catch(e){error.value=e.response?.data?.error||e.message} finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.table-footer {display:flex;justify-content:space-between;align-items:center; padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
.metric-head {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; font-weight: 600; margin-bottom: 8px;
}
.muted { color: var(--ks-text-secondary); font-weight: 400; font-size: 12px; }
</style>
