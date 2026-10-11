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
      <el-table :data="filtered" v-loading="loading" stripe class="ks-table">
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
        <el-table-column label="镜像" min-width="200">
          <template #default="{ row }">
            <div v-for="img in row.images" :key="img" class="image-text">{{ img }}</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openLogs(row)">日志</el-button>
            <el-button link type="primary" size="small" @click="openTerm(row)">终端</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row.name, row.namespace)">YAML</el-button>
            <el-button link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

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

const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)
const logVisible = ref(false)
const termVisible = ref(false)
const containers = ref([])

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
</style>
