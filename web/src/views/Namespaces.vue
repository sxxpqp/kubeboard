<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <el-button type="primary" :icon="'Plus'" @click="openCreate(nsTemplate)">创建项目</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>
      <el-table :data="filtered" v-loading="loading" stripe class="ks-table">
        <el-table-column prop="name" label="名称" min-width="220" sortable>
          <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="120">
          <template #default="{ row }">
            <el-tag :type="row.status === 'Active' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="标签" min-width="260">
          <template #default="{ row }">
            <el-tag v-for="(v, k) in row.labels" :key="k" size="small" type="info" style="margin-right: 4px; margin-bottom: 2px">{{ k }}={{ v }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="160" sortable prop="creationTime">
          <template #default="{ row }">{{ fmtTime(row.creationTime) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row.name)">编辑</el-button>
            <el-button
              link type="danger" size="small"
              :disabled="['default','kube-system','kube-public','kube-node-lease'].includes(row.name)"
              @click="remove(row.name)"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <YamlDialog
      v-model="dialogVisible"
      :mode="dialogMode"
      kind="namespace"
      :name="currentName"
      :template="currentTemplate"
      @saved="load"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from '../components/YamlDialog.vue'

const keyword = ref('')
const items = ref([])
const loading = ref(true)

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.toLowerCase().includes(keyword.value.toLowerCase()))
)

const nsTemplate = `apiVersion: v1
kind: Namespace
metadata:
  name: my-project
  labels:
    kubesphere.io/workspace: default
`

const {
  dialogVisible, dialogMode, currentName, currentTemplate,
  openCreate, openEdit, remove,
} = useResourceCRUD('namespace', load)

function fmtTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

async function load() {
  loading.value = true
  try {
    const data = await http.get('/namespaces')
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
