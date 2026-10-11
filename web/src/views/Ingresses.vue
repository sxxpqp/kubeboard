<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-select v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 170px">
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <el-button type="primary" :icon="'Plus'" @click="openCreate(ingressTemplate)">创建</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>
      <el-table :data="filtered" v-loading="loading" stripe class="ks-table">
        <el-table-column prop="namespace" label="项目" width="130" sortable />
        <el-table-column prop="name" label="名称" min-width="160" sortable>
          <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column label="域名" min-width="180">
          <template #default="{ row }">
            <el-tag v-for="h in row.hosts" :key="h" size="small" type="info" style="margin-right: 4px">{{ h }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="路径 → 后端服务" min-width="260">
          <template #default="{ row }">
            <div v-for="(r, i) in row.rules" :key="i" class="rule-text">
              {{ r.path }} → {{ r.service }}:{{ r.port }}
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="className" label="Ingress Class" width="140" />
        <el-table-column label="TLS" width="80" align="center">
          <template #default="{ row }">
            <el-tag size="small" :type="row.tls ? 'success' : 'info'">{{ row.tls ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openEdit(row.name, row.namespace)">编辑</el-button>
            <el-button link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <YamlDialog v-model="dialogVisible" :mode="dialogMode" kind="ingress"
      :namespace="currentNamespace" :name="currentName" :template="currentTemplate" @saved="load" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from '../components/YamlDialog.vue'

const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.toLowerCase().includes(keyword.value.toLowerCase()))
)

const ingressTemplate = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: default
spec:
  ingressClassName: nginx
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-service
                port:
                  number: 80
`

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openCreate, openEdit, remove,
} = useResourceCRUD('ingress', load)

async function load() {
  loading.value = true
  try {
    const params = {}
    if (namespace.value) params.namespace = namespace.value
    const data = await http.get('/ingresses', { params })
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
.rule-text { font-size: 12px; color: var(--ks-text-secondary); line-height: 1.8; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
</style>
