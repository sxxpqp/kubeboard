<template>
  <div v-loading="loading">
    <div class="ks-toolbar">
      <el-button :icon="'Back'" @click="$router.push('/namespaces')">返回</el-button>
      <div class="ns-title">项目: {{ ns }}</div>
    </div>

    <div class="ks-stat-row" style="margin-bottom: 16px">
      <div class="ks-stat">
        <div class="label">容器组</div>
        <div class="value">{{ sm.pods?.total ?? '-' }}</div>
        <div class="sub">
          <span style="color: var(--ks-success)">{{ sm.pods?.running ?? 0 }} 运行</span> /
          <span style="color: var(--ks-warning)">{{ sm.pods?.pending ?? 0 }} 等待</span> /
          <span style="color: var(--ks-danger)">{{ sm.pods?.failed ?? 0 }} 失败</span>
        </div>
      </div>
      <div class="ks-stat">
        <div class="label">工作负载</div>
        <div class="value">{{ totalWorkloads }}</div>
        <div class="sub">部署 {{ sm.workloads?.deployments ?? 0 }} · 有状态 {{ sm.workloads?.statefulsets ?? 0 }} · 守护 {{ sm.workloads?.daemonsets ?? 0 }}</div>
      </div>
      <div class="ks-stat">
        <div class="label">服务 / 路由</div>
        <div class="value">{{ sm.services ?? 0 }} / {{ sm.ingresses ?? 0 }}</div>
        <div class="sub">Service / Ingress</div>
      </div>
      <div class="ks-stat">
        <div class="label">配置 / 密钥</div>
        <div class="value">{{ sm.configmaps ?? 0 }} / {{ sm.secrets ?? 0 }}</div>
        <div class="sub">ConfigMap / Secret</div>
      </div>
    </div>

    <div class="ks-card">
      <div class="ks-card-title">快捷跳转</div>
      <div style="display: flex; gap: 10px; flex-wrap: wrap">
        <el-button @click="goto('/pods')">容器组 ({{ sm.pods?.total ?? 0 }})</el-button>
        <el-button @click="goto('/workloads')">工作负载 ({{ totalWorkloads }})</el-button>
        <el-button @click="goto('/services')">服务 ({{ sm.services ?? 0 }})</el-button>
        <el-button @click="goto('/ingresses')">应用路由 ({{ sm.ingresses ?? 0 }})</el-button>
        <el-button @click="goto('/pvcs')">持久卷声明 ({{ sm.pvcs ?? 0 }})</el-button>
        <el-button @click="goto('/configmaps')">配置字典 ({{ sm.configmaps ?? 0 }})</el-button>
        <el-button @click="goto('/secrets')">保密字典 ({{ sm.secrets ?? 0 }})</el-button>
        <el-button @click="goto('/jobs')">任务 ({{ sm.jobs ?? 0 }})</el-button>
        <el-button @click="goto('/cronjobs')">定时任务 ({{ sm.cronjobs ?? 0 }})</el-button>
      </div>
    </div>

    <div class="ks-card">
      <div class="ks-card-title">资源配额</div>
      <el-empty v-if="!quotas.length" description="该项目未设置资源配额" :image-size="60" />
      <el-table v-else :data="quotas" size="small">
        <el-table-column prop="name" label="配额名称" width="180" />
        <el-table-column label="资源用量 / 上限" min-width="400">
          <template #default="{ row }">
            <div v-for="(hard, k) in row.hard" :key="k" class="quota-line">
              <span class="quota-key">{{ k }}</span>
              <span>{{ row.used[k] || '0' }} / {{ hard }}</span>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'

const route = useRoute()
const router = useRouter()
const ns = route.params.namespace
const sm = ref({})
const quotas = ref([])
const loading = ref(true)

const totalWorkloads = computed(() => {
  const w = sm.value.workloads || {}
  return (w.deployments || 0) + (w.statefulsets || 0) + (w.daemonsets || 0)
})

function goto(path) {
  router.push({ path, query: { namespace: ns } })
}

onMounted(async () => {
  try {
    sm.value = await http.get(`/namespaces/${ns}/summary`)
    const q = await http.get('/resourcequotas', { params: { namespace: ns } })
    quotas.value = q.items || []
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.ns-title { font-size: 16px; font-weight: 600; }
.quota-line { font-size: 12px; line-height: 1.9; }
.quota-key { color: var(--ks-text-secondary); display: inline-block; width: 200px; }
</style>
