<template>
  <div v-loading="loading">
    <el-alert v-if="error" :title="error" type="error" :closable="false" show-icon style="margin-bottom: 16px">
      <el-button link @click="load">重新加载</el-button>
    </el-alert>
    <div class="overview-banner">
      <div><span class="ks-eyebrow">CLUSTER STATUS</span><h2>集群运行概况</h2><p>Kubernetes {{ ov.version || '—' }} · {{ updatedAt ? '更新于 ' + updatedAt : '等待获取数据' }}</p></div>
      <el-button @click="load" :loading="loading">刷新状态</el-button>
    </div>
    <div class="ks-stat-row overview-stats">
      <router-link class="ks-stat" to="/nodes"><div class="label">节点就绪</div><div class="value">{{ ov.nodes?.ready ?? '—' }} <small>/ {{ ov.nodes?.total ?? '—' }}</small></div><div class="sub">查看节点状态 →</div></router-link>
      <router-link class="ks-stat" to="/pods"><div class="label">失败的容器组</div><div class="value" :class="{ danger: ov.pods?.failed > 0 }">{{ ov.pods?.failed ?? '—' }}</div><div class="sub">进入容器组排查 →</div></router-link>
      <router-link class="ks-stat" to="/namespaces"><div class="label">项目</div><div class="value">{{ ov.namespaces ?? '—' }}</div><div class="sub">命名空间资源 →</div></router-link>
      <router-link class="ks-stat" to="/workloads"><div class="label">部署</div><div class="value">{{ ov.deployments ?? '—' }}</div><div class="sub">查看应用副本与更新 →</div></router-link>
    </div>
    <div class="overview-grid">
      <section class="ks-card"><div class="ks-card-title">容器组运行状态</div>
        <div v-for="state in states" :key="state.key" class="pod-state-row"><span><i :style="{ background: state.color }"></i>{{ state.label }}</span><el-progress :percentage="percentage(state.key)" :color="state.color" :show-text="false" /><strong>{{ ov.pods?.[state.key] ?? '—' }}</strong></div>
        <div class="overview-total">容器组总数 <strong>{{ ov.pods?.total ?? '—' }}</strong></div>
      </section>
      <section class="ks-card"><div class="ks-card-title">集群能力与资源</div>
        <div class="capability"><span>资源指标服务</span><el-tag :type="ov.metrics ? 'success' : 'info'">{{ ov.metrics === undefined ? '等待确认' : ov.metrics ? '已配置' : '不可用' }}</el-tag></div>
        <p class="muted">{{ ov.metrics ? '在节点与容器组页面查看可用的实时指标。' : '当前不展示资源使用数据，运行状态仍可查看。' }}</p>
        <div class="capability"><span>服务</span><router-link to="/services">{{ ov.services ?? '—' }} →</router-link></div>
        <div class="capability"><span>近期事件</span><router-link to="/events">{{ ov.recentEvents ?? '—' }} →</router-link></div>
        <div class="diagnostic-note">排障建议：先查看异常资源，再进入具体容器检查日志及事件。</div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref, onUnmounted } from 'vue'
import http from '../api/http'
const ov = ref({})
const loading = ref(false)
const error = ref('')
const updatedAt = ref('')
const states = [
  { key: 'running', label: '运行中', color: '#23865b' },
  { key: 'pending', label: '等待中', color: '#a66b00' },
  { key: 'failed', label: '失败', color: '#c9363e' },
  { key: 'succeeded', label: '已完成', color: '#79879c' },
]
function percentage(key) { return ov.value.pods?.total ? Math.min(100, Math.round((ov.value.pods[key] || 0) / ov.value.pods.total * 100)) : 0 }
let timer
async function load() {
  if (loading.value) return
  loading.value = true
  try {
    ov.value = await http.get('/overview')
    updatedAt.value = new Date().toLocaleTimeString('zh-CN', { hour12: false })
    error.value = ''
  } catch (e) { error.value = '概览加载失败：' + (e?.message || '请重试') }
  finally { loading.value = false }
}
onMounted(() => { load(); timer = setInterval(load, 15000) })
onUnmounted(() => clearInterval(timer))
</script>

<style scoped>
.overview-banner { display: flex; align-items: center; justify-content: space-between; background: #e8eef3; border-radius: 8px; padding: 28px; margin-bottom: 24px; }
h2 { margin: 8px 0; font-size: 22px; }
p { color: var(--ks-text-secondary); font-size: 13px; line-height: 1.7; }
.overview-stats { margin-bottom: 24px; }
.value small { font-size: 16px; color: var(--ks-text-secondary); }
.value.danger { color: #c9363e; }
.overview-grid { display: grid; grid-template-columns: 1.2fr 1fr; gap: 24px; }
.pod-state-row { display: grid; grid-template-columns: 90px 1fr 40px; align-items: center; gap: 16px; margin: 26px 0; }
.pod-state-row i { width: 8px; height: 8px; display: inline-block; border-radius: 50%; margin-right: 8px; }
.pod-state-row strong { text-align: right; }
.overview-total, .capability { display: flex; justify-content: space-between; align-items: center; padding: 16px 0; border-bottom: 1px solid var(--ks-border); }
.diagnostic-note { padding: 16px; margin-top: 20px; background: #f3f5f8; color: var(--ks-text-secondary); font-size: 12px; line-height: 1.8; border-radius: 4px; }
@media(max-width: 1100px) { .overview-grid { grid-template-columns: 1fr; } }
</style>
