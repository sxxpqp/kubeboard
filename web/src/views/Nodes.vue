<template>
  <div class="ks-card">
    <div class="ks-toolbar">
      <el-input v-model="keyword" placeholder="按名称搜索" clearable style="width: 240px" :prefix-icon="'Search'" />
      <div style="flex: 1"></div>
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

    <!-- 节点详情抽屉 -->
    <ResourceDetail
      v-model="detailVisible"
      :title="`节点: ${detailRow.name}`"
      kind="node" :name="detailRow.name || ''"
      :fields="detailFields" :load-detail="loadNodeDetail"
    >
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

const items = ref([])
const loading = ref(true)
const keyword = ref('')

const detailVisible = ref(false)
const detailRow = ref({})
const podsVisible = ref(false)
const podsLoading = ref(false)
const podsNode = ref('')
const nodePods = ref([])

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
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
</style>
