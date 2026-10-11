<template>
  <div v-loading="loading">
    <!-- 状态摘要 -->
    <div class="lc-summary">
      <el-tag :type="phaseType" effect="dark" size="large">{{ detail.phase || '-' }}</el-tag>
      <span class="lc-item">QoS: <b>{{ detail.qosClass || '-' }}</b></span>
      <span class="lc-item">重启策略: <b>{{ detail.restartPolicy || '-' }}</b></span>
      <span class="lc-item">重启次数: <b :class="{ warn: detail.restarts > 0 }">{{ detail.restarts ?? 0 }}</b></span>
    </div>

    <!-- 时间轴 -->
    <div class="lc-section">
      <div class="lc-title">时间线</div>
      <el-timeline>
        <el-timeline-item
          v-if="detail.creationTime"
          :timestamp="fmtTime(detail.creationTime)" placement="top" type="primary"
        >创建 Pod</el-timeline-item>
        <el-timeline-item
          v-if="condTime('PodScheduled')"
          :timestamp="fmtTime(condTime('PodScheduled'))" placement="top"
        >调度到节点 <b>{{ detail.nodeName }}</b></el-timeline-item>
        <el-timeline-item
          v-for="c in detail.containers || []"
          :key="'c-' + c.name"
          v-show="c.startedAt"
          :timestamp="fmtTime(c.startedAt)" placement="top" type="success"
        >容器 <b>{{ c.name }}</b> 启动</el-timeline-item>
        <el-timeline-item
          v-if="condTime('Ready')"
          :timestamp="fmtTime(condTime('Ready'))" placement="top" type="success"
        >Pod 就绪（Ready）</el-timeline-item>
        <el-timeline-item
          v-if="detail.deletionTime"
          :timestamp="fmtTime(detail.deletionTime)" placement="top" type="danger"
        >开始终止（Terminating）</el-timeline-item>
      </el-timeline>
    </div>

    <!-- 条件链 -->
    <div class="lc-section">
      <div class="lc-title">Pod 条件</div>
      <div class="cond-list">
        <div v-for="c in detail.conditions || []" :key="c.type" class="cond-item">
          <span class="cond-icon" :class="c.status === 'True' ? 'ok' : 'bad'">
            {{ c.status === 'True' ? '✓' : '✕' }}
          </span>
          <span class="cond-type">{{ c.type }}</span>
          <span class="cond-msg">{{ c.reason || c.message || '-' }}</span>
          <span class="cond-time">{{ fmtTime(c.lastTransitionTime) }}</span>
        </div>
      </div>
    </div>

    <!-- 容器明细 -->
    <div class="lc-section">
      <div class="lc-title">容器状态</div>
      <el-table :data="detail.containers || []" size="small">
        <el-table-column prop="name" label="容器" min-width="130" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="stateType(row.state)" size="small">{{ row.state }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="就绪" width="70" align="center">
          <template #default="{ row }">
            <span :style="{ color: row.ready ? 'var(--ks-success)' : 'var(--ks-danger)' }">
              {{ row.ready ? '✓' : '✕' }}
            </span>
          </template>
        </el-table-column>
        <el-table-column prop="restartCount" label="重启" width="70" align="center" />
        <el-table-column label="启动时间" width="160">
          <template #default="{ row }">{{ fmtTime(row.startedAt) }}</template>
        </el-table-column>
        <el-table-column label="上次终止" min-width="180">
          <template #default="{ row }">
            <span v-if="row.lastTerminated">
              {{ row.lastTerminated.reason }}（退出码 {{ row.lastTerminated.exitCode }}，{{ fmtTime(row.lastTerminated.finishedAt) }}）
            </span>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
      </el-table>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const props = defineProps({
  namespace: { type: String, required: true },
  name: { type: String, required: true },
})

const detail = ref({})
const loading = ref(false)

const phaseType = computed(() => {
  const p = detail.value.phase
  if (p === 'Running') return 'success'
  if (p === 'Pending') return 'warning'
  if (p === 'Failed' || p === 'Evicted') return 'danger'
  return 'info'
})

function stateType(s) {
  if (s === 'Running') return 'success'
  if (s === 'Waiting') return 'warning'
  if (s === 'Terminated') return 'danger'
  return 'info'
}

function fmtTime(ts) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}

function condTime(type) {
  const c = (detail.value.conditions || []).find((x) => x.type === type)
  return c ? c.lastTransitionTime : 0
}

async function load() {
  loading.value = true
  try {
    detail.value = await http.get('/pods/detail', { params: { namespace: props.namespace, name: props.name } })
  } catch (e) {
    ElMessage.error('获取 Pod 生命周期失败: ' + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.lc-summary { display: flex; align-items: center; gap: 18px; margin-bottom: 18px; flex-wrap: wrap; }
.lc-item { font-size: 13px; color: var(--ks-text-secondary); }
.lc-item b { color: var(--ks-text, #2b3a4a); }
.lc-item b.warn { color: var(--ks-danger); }
.lc-section { margin-bottom: 22px; }
.lc-title { font-weight: 600; font-size: 13px; margin-bottom: 12px; }
.cond-list { display: flex; flex-direction: column; gap: 8px; }
.cond-item { display: flex; align-items: center; gap: 10px; font-size: 12px;
  padding: 8px 12px; background: var(--ks-bg-light, #f7f9fc); border-radius: 6px; }
.cond-icon { width: 18px; height: 18px; border-radius: 50%; display: flex;
  align-items: center; justify-content: center; color: #fff; font-size: 11px; flex-shrink: 0; }
.cond-icon.ok { background: var(--ks-success, #16a34a); }
.cond-icon.bad { background: var(--ks-danger, #dc2626); }
.cond-type { font-weight: 600; min-width: 170px; }
.cond-msg { color: var(--ks-text-secondary); flex: 1; }
.cond-time { color: var(--ks-text-secondary); font-size: 11px; }
.muted { color: var(--ks-text-secondary); }
</style>
