<template>
  <div v-loading="loading">
    <div class="ks-stat-row" style="margin-bottom: 16px">
      <div class="ks-stat">
        <div class="label">集群版本</div>
        <div class="value" style="font-size: 16px">{{ ov.version || '-' }}</div>
        <div class="sub">metrics-server: {{ ov.metrics ? '已启用' : '未安装' }}</div>
      </div>
      <div class="ks-stat">
        <div class="label">节点</div>
        <div class="value">{{ ov.nodes?.ready ?? '-' }} / {{ ov.nodes?.total ?? '-' }}</div>
        <div class="sub">就绪 / 总数</div>
      </div>
      <div class="ks-stat">
        <div class="label">项目 (Namespace)</div>
        <div class="value">{{ ov.namespaces ?? '-' }}</div>
        <div class="sub">运行中的命名空间</div>
      </div>
      <div class="ks-stat">
        <div class="label">部署</div>
        <div class="value">{{ ov.deployments ?? '-' }}</div>
        <div class="sub">Deployment 总数</div>
      </div>
    </div>

    <div class="ks-card">
      <div class="ks-card-title">容器组状态</div>
      <div class="ks-stat-row" style="grid-template-columns: repeat(5, 1fr)">
        <div class="ks-stat" style="box-shadow: none; background: var(--ks-body-bg)">
          <div class="label">运行中</div>
          <div class="value" style="color: var(--ks-success)">{{ ov.pods?.running ?? 0 }}</div>
        </div>
        <div class="ks-stat" style="box-shadow: none; background: var(--ks-body-bg)">
          <div class="label">等待中</div>
          <div class="value" style="color: var(--ks-warning)">{{ ov.pods?.pending ?? 0 }}</div>
        </div>
        <div class="ks-stat" style="box-shadow: none; background: var(--ks-body-bg)">
          <div class="label">失败</div>
          <div class="value" style="color: var(--ks-danger)">{{ ov.pods?.failed ?? 0 }}</div>
        </div>
        <div class="ks-stat" style="box-shadow: none; background: var(--ks-body-bg)">
          <div class="label">已完成</div>
          <div class="value">{{ ov.pods?.succeeded ?? 0 }}</div>
        </div>
        <div class="ks-stat" style="box-shadow: none; background: var(--ks-body-bg)">
          <div class="label">总数</div>
          <div class="value">{{ ov.pods?.total ?? 0 }}</div>
        </div>
      </div>
    </div>

    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px">
      <div class="ks-card">
        <div class="ks-card-title">服务与事件</div>
        <div class="ks-stat-row" style="grid-template-columns: 1fr 1fr">
          <div class="ks-stat" style="box-shadow: none">
            <div class="label">服务总数</div>
            <div class="value">{{ ov.services ?? 0 }}</div>
          </div>
          <div class="ks-stat" style="box-shadow: none">
            <div class="label">近期事件</div>
            <div class="value">{{ ov.recentEvents ?? 0 }}</div>
          </div>
        </div>
      </div>
      <div class="ks-card">
        <div class="ks-card-title">快捷入口</div>
        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px">
          <el-button @click="$router.push('/nodes')">节点管理</el-button>
          <el-button @click="$router.push('/workloads')">工作负载</el-button>
          <el-button @click="$router.push('/services')">服务</el-button>
          <el-button @click="$router.push('/events')">事件查询</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, onUnmounted } from 'vue'
import http from '../api/http'

const ov = ref({})
const loading = ref(true)
let timer

async function load() {
  try {
    ov.value = await http.get('/overview')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 15000)
})
onUnmounted(() => clearInterval(timer))
</script>
