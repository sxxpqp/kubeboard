<template>
  <div class="ks-card">
    <div class="ks-toolbar">
      <el-input v-model="keyword" placeholder="按名称搜索" clearable style="width: 240px" :prefix-icon="'Search'" />
    </div>
    <el-table :data="filtered" v-loading="loading" stripe>
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <span class="ks-status">
            <span class="dot" :class="row.status"></span>{{ row.status }}
          </span>
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
    </el-table>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'

const items = ref([])
const loading = ref(true)
const keyword = ref('')

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.includes(keyword.value))
)

function formatMem(mem) {
  const n = parseInt(mem)
  if (isNaN(n)) return mem
  return (n / (1024 * 1024 * 1024)).toFixed(1) + ' Gi'
}

onMounted(async () => {
  try {
    const data = await http.get('/nodes')
    items.value = data.items
  } finally {
    loading.value = false
  }
})
</script>
