<template>
  <div class="ks-card">
    <div class="ks-toolbar">
      <el-select v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 180px">
        <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
      </el-select>
      <el-select v-model="type" placeholder="全部类型" clearable @change="load" style="width: 140px">
        <el-option label="Normal" value="Normal" />
        <el-option label="Warning" value="Warning" />
      </el-select>
      <el-input v-model="keyword" placeholder="搜索" clearable style="width: 200px" />
    </div>
    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column label="最后出现时间" width="170">
        <template #default="{ row }">{{ row.lastSeen ? new Date(row.lastSeen * 1000).toLocaleString() : '-' }}</template>
      </el-table-column>
      <el-table-column label="级别" width="100">
        <template #default="{ row }">
          <el-tag :type="row.type === 'Warning' ? 'danger' : 'info'" size="small">{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="object" label="资源对象" width="220" />
      <el-table-column prop="reason" label="原因" width="140" />
      <el-table-column prop="message" label="消息" min-width="320" />
      <el-table-column prop="count" label="次数" width="80" />
    </el-table>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'

const namespace = ref('')
const type = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.message?.includes(keyword.value) || i.object?.includes(keyword.value))
)

async function load() {
  loading.value = true
  try {
    const params = {}
    if (namespace.value) params.namespace = namespace.value
    const data = await http.get('/events', { params })
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
