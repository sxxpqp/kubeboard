<template>
  <ResourceDetailPage kind-label="部署" :extra-tabs="[{ name: 'rollout', label: '滚动更新' }, { name: 'versions', label: '版本历史' }]">
    <template #actions>
      <el-button size="small" @click="doScale">伸缩</el-button>
      <el-button size="small" @click="doRestart">重启</el-button>
      <el-button size="small" @click="doRollout">滚动更新状态</el-button>
    </template>

    <!-- 概览补充：副本状态 -->
    <template #overview-extra="{ detail }">
      <div class="rd-card" style="margin-top: 16px" v-if="detail.spec">
        <div class="rd-card-title">副本状态</div>
        <div class="stat-row">
          <div class="stat"><span>期望</span><b>{{ detail.spec.replicas ?? '-' }}</b></div>
          <div class="stat"><span>已更新</span><b>{{ detail.status?.updatedReplicas ?? 0 }}</b></div>
          <div class="stat"><span>就绪</span><b>{{ detail.status?.readyReplicas ?? 0 }}</b></div>
          <div class="stat"><span>可用</span><b>{{ detail.status?.availableReplicas ?? 0 }}</b></div>
          <div class="stat"><span>不可用</span><b :class="{ bad: (detail.status?.unavailableReplicas || 0) > 0 }">{{ detail.status?.unavailableReplicas ?? 0 }}</b></div>
        </div>
        <el-descriptions :column="2" border size="small" style="margin-top: 12px">
          <el-descriptions-item label="更新策略">{{ detail.spec.strategy || '-' }}</el-descriptions-item>
          <el-descriptions-item label="maxSurge">{{ detail.spec.maxSurge || '-' }}</el-descriptions-item>
          <el-descriptions-item label="maxUnavailable">{{ detail.spec.maxUnavailable || '-' }}</el-descriptions-item>
          <el-descriptions-item label="暂停">{{ detail.spec.paused ? '是' : '否' }}</el-descriptions-item>
        </el-descriptions>
        <div class="rd-card-title" style="margin-top: 16px">容器</div>
        <el-table :data="detail.spec.containers || []" size="small">
          <el-table-column prop="name" label="名称" width="140" />
          <el-table-column prop="image" label="镜像" min-width="220" />
          <el-table-column label="端口" width="140">
            <template #default="{ row }">
              <el-tag v-for="(p, i) in row.ports || []" :key="i" size="small" style="margin-right: 4px">{{ p.containerPort }}</el-tag>
              <span v-if="!(row.ports || []).length" class="muted">-</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </template>

    <!-- 滚动更新 Tab -->
    <template #tab-rollout>
      <div v-loading="rolloutLoading">
        <template v-if="rollout">
          <div class="ro-head">
            <el-tag :type="rollout.inProgress ? 'warning' : 'success'" effect="dark">{{ rollout.message }}</el-tag>
            <span class="muted">策略: {{ rollout.strategy }}</span>
          </div>
          <el-progress :percentage="rollout.progress" :status="rollout.inProgress ? undefined : 'success'" :stroke-width="18" text-inside />
          <div class="stat-row" style="margin: 16px 0">
            <div class="stat"><span>期望</span><b>{{ rollout.desired }}</b></div>
            <div class="stat"><span>已更新</span><b>{{ rollout.updatedReplicas }}</b></div>
            <div class="stat"><span>就绪</span><b>{{ rollout.readyReplicas }}</b></div>
            <div class="stat"><span>可用</span><b>{{ rollout.availableReplicas }}</b></div>
          </div>
          <el-table :data="rollout.conditions || []" size="small">
            <el-table-column prop="type" label="条件" width="140" />
            <el-table-column label="状态" width="80">
              <template #default="{ row }"><el-tag :type="row.status === 'True' ? 'success' : 'info'" size="small">{{ row.status }}</el-tag></template>
            </el-table-column>
            <el-table-column prop="reason" label="原因" width="150" />
            <el-table-column prop="message" label="消息" min-width="200" />
          </el-table>
        </template>
        <el-empty v-else description="暂无滚动更新数据" />
      </div>
    </template>

    <!-- 版本历史 Tab -->
    <template #tab-versions>
      <VersionHistory v-if="rKind && rName" :kind="rKind" :namespace="rNs" :name="rName" />
    </template>
  </ResourceDetailPage>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import ResourceDetailPage from '../components/ResourceDetailPage.vue'
import VersionHistory from '../components/VersionHistory.vue'

const route = useRoute()
const router = useRouter()
const rKind = computed(() => route.params.kind || 'deployment')
const rName = computed(() => route.params.name || '')
const rNs = computed(() => route.params.namespace || '')

const rollout = ref(null)
const rolloutLoading = ref(false)

async function loadRollout() {
  rolloutLoading.value = true
  try {
    rollout.value = await http.get('/workloads/rollout', { params: { kind: rKind.value, namespace: rNs.value, name: rName.value } })
  } catch { rollout.value = null } finally { rolloutLoading.value = false }
}

function doScale() {
  ElMessageBox.prompt('输入新的副本数', '伸缩', { inputPattern: /^\d+$/, inputErrorMessage: '请输入数字' })
    .then(async ({ value }) => {
      await http.post('/workloads/scale', { kind: rKind.value, namespace: rNs.value, name: rName.value, replicas: Number(value) })
      ElMessage.success('副本数已更新')
    }).catch(() => {})
}
async function doRestart() {
  try { await ElMessageBox.confirm('确定触发滚动重启？', '重启确认', { type: 'warning' }) } catch { return }
  await http.post('/workloads/restart', { kind: rKind.value, namespace: rNs.value, name: rName.value })
  ElMessage.success('已触发滚动重启')
}
function doRollout() { loadRollout(); ElMessage.info('已刷新滚动更新状态，请查看「滚动更新」标签') }

onMounted(loadRollout)
</script>

<style scoped>
.rd-card { background: var(--ks-bg-light, #f9fbfd); border-radius: 8px; padding: 16px; }
.rd-card-title { font-weight: 600; font-size: 14px; margin-bottom: 12px; }
.stat-row { display: flex; gap: 12px; }
.stat { flex: 1; text-align: center; background: #fff; border-radius: 6px; padding: 10px; }
.stat span { display: block; font-size: 12px; color: var(--ks-text-secondary); margin-bottom: 4px; }
.stat b { font-size: 18px; }
.stat b.bad { color: var(--ks-danger); }
.muted { color: var(--ks-text-secondary); font-size: 12px; }
.ro-head { display: flex; align-items: center; gap: 12px; margin-bottom: 14px; }
</style>
