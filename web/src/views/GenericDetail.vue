<template>
  <ResourceDetailPage :extra-tabs="extraTabs">
    <template #actions><el-button v-if="kind==='pod'" @click="consoleVisible=true">日志 / 终端</el-button></template>
    <template #tab-lifecycle><PodLifecycle :namespace="namespace" :name="resourceName" /></template>
    <template #overview-extra="{ detail }">
      <!-- 节点专属 -->
      <div v-if="kind === 'node' && detail.status" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title">容量 / 可分配</div>
        <el-table :data="nodeResources(detail)" size="small">
          <el-table-column prop="key" label="资源" width="200" />
          <el-table-column prop="capacity" label="容量" min-width="160" />
          <el-table-column prop="allocatable" label="可分配" min-width="160" />
        </el-table>
        <div class="rd-card-title" style="margin-top:16px">节点条件</div>
        <el-table :data="detail.status.conditions || []" size="small">
          <el-table-column prop="type" label="类型" width="180" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><el-tag :type="conditionType(row)" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" width="160" />
          <el-table-column prop="message" label="消息" min-width="220" />
        </el-table>

        <div class="rd-card-title" style="margin-top:16px">污点（Taints）</div>
        <el-table v-if="(detail.spec.taints || []).length" :data="detail.spec.taints" size="small">
          <el-table-column prop="key" label="键" min-width="200" />
          <el-table-column prop="value" label="值" min-width="140">
            <template #default="{ row }">{{ row.value || '-' }}</template>
          </el-table-column>
          <el-table-column prop="effect" label="影响" width="160" />
        </el-table>
        <div v-else class="muted" style="padding: 6px 0">无污点</div>

        <div class="rd-card-title" style="margin-top:16px">节点地址 / 信息</div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item v-for="(a, i) in detail.status.addresses || []" :key="i" :label="a.type">
            <span class="mono-sm">{{ a.address }}</span>
          </el-descriptions-item>
          <el-descriptions-item label="容器运行时">{{ detail.status.nodeInfo?.containerRuntimeVersion || '-' }}</el-descriptions-item>
          <el-descriptions-item label="内核版本">{{ detail.status.nodeInfo?.kernelVersion || '-' }}</el-descriptions-item>
          <el-descriptions-item label="操作系统">
            {{ (detail.status.nodeInfo?.osImage || '-') }} ({{ detail.status.nodeInfo?.architecture || '-' }})
          </el-descriptions-item>
          <el-descriptions-item label="Pod CIDR">{{ detail.spec.podCIDR || '-' }}</el-descriptions-item>
          <el-descriptions-item label="可调度">
            <el-tag size="small" :type="detail.spec.unschedulable ? 'danger' : 'success'">{{ detail.spec.unschedulable ? '已封锁' : '正常' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Kubelet">{{ detail.status.nodeInfo?.kubeletVersion || '-' }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <!-- 服务专属 -->
      <div v-if="kind === 'service' && detail.spec" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title">端口</div>
        <el-table :data="detail.spec.ports || []" size="small">
          <el-table-column prop="name" label="名称" width="120" />
          <el-table-column prop="port" label="端口" width="90" />
          <el-table-column prop="targetPort" label="目标端口" width="110" />
          <el-table-column prop="protocol" label="协议" width="90" />
          <el-table-column prop="nodePort" label="NodePort" width="110">
            <template #default="{ row }">{{ row.nodePort || '-' }}</template>
          </el-table-column>
        </el-table>
        <el-descriptions :column="2" border size="small" style="margin-top:12px">
          <el-descriptions-item label="类型">{{ detail.spec.type }}</el-descriptions-item>
          <el-descriptions-item label="Cluster IP">{{ detail.spec.clusterIP }}</el-descriptions-item>
          <el-descriptions-item label="会话保持">{{ detail.spec.sessionAffinity }}</el-descriptions-item>
          <el-descriptions-item label="选择器">
            <span v-if="Object.keys(detail.spec.selector || {}).length">
              <el-tag v-for="(v,k) in detail.spec.selector" :key="k" size="small" style="margin:0 4px 2px 0">{{ k }}={{ v }}</el-tag>
            </span>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>

        <div class="rd-card-title" style="margin-top:16px">
          Endpoints 后端
          <el-tag v-if="endpoints" size="small" :type="endpoints.readyCount ? 'success' : 'danger'" style="margin-left:8px">
            就绪 {{ endpoints.readyCount }}{{ endpoints.notReadyCount ? ` / 未就绪 ${endpoints.notReadyCount}` : '' }}
          </el-tag>
          <el-button link type="primary" size="small" style="margin-left:8px" @click="loadEndpoints">刷新</el-button>
        </div>
        <div v-loading="loadingEndpoints"><el-alert v-if="endpointsError" :title="endpointsError" type="error" :closable="false" />
          <el-alert v-if="endpoints && !endpoints.readyCount && !endpoints.notReadyCount" type="warning" :closable="false"
            title="无后端地址：可能选择器不匹配任何 Pod" />
          <el-table v-if="endpoints && (endpoints.readyCount || endpoints.notReadyCount)" :data="endpointRows" size="small">
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag size="small" :type="row.ready ? 'success' : 'warning'">{{ row.ready ? '就绪' : '未就绪' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="ip" label="IP" width="150" />
            <el-table-column prop="port" label="端口" width="90" />
            <el-table-column prop="nodeName" label="节点" width="160" />
            <el-table-column label="后端对象" min-width="240">
              <template #default="{ row }">
                <el-link v-if="row.targetRef" type="primary" @click="gotoKind(row.targetRef.kind.toLowerCase(), row.targetRef.name, row.targetRef.namespace)">
                  {{ row.targetRef.kind }}/{{ row.targetRef.name }}
                </el-link>
                <span v-else class="muted">-</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>

      <!-- Ingress 专属 -->
      <div v-if="kind === 'ingress' && detail.spec" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title">路由规则</div>
        <div v-for="(r, i) in detail.spec.rules || []" :key="i" class="ing-rule">
          <div class="ing-host">Host: <b>{{ r.host || '*' }}</b></div>
          <el-table :data="r.paths || []" size="small">
            <el-table-column prop="path" label="路径" width="180" />
            <el-table-column prop="pathType" label="类型" width="140" />
            <el-table-column prop="service" label="后端服务" width="180" />
            <el-table-column prop="port" label="端口" width="100" />
          </el-table>
        </div>
        <el-descriptions :column="1" border size="small" style="margin-top:12px">
          <el-descriptions-item label="IngressClass">{{ detail.spec.ingressClassName || '-' }}</el-descriptions-item>
        </el-descriptions>
      </div>

      <div v-if="kind === 'pvc'" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title" style="margin-top:16px">正在使用该 PVC 的 Pod</div>
        <el-table v-if="(detail.usedByPods || []).length" :data="detail.usedByPods" size="small">
          <el-table-column prop="namespace" label="项目" width="140" />
          <el-table-column label="Pod" min-width="220">
            <template #default="{ row }">
              <el-link type="primary" @click="gotoKind('pod', row.name, row.namespace)">{{ row.name }}</el-link>
            </template>
          </el-table-column>
        </el-table>
        <div v-else class="muted" style="padding: 6px 0">未发现使用中的 Pod</div>
      </div>
    </template>
  </ResourceDetailPage>
  <WorkloadConsole v-if="kind==='pod'" v-model="consoleVisible" kind="pod" :namespace="namespace" :name="resourceName" />
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import ResourceDetailPage from '../components/ResourceDetailPage.vue'
import PodLifecycle from '../components/PodLifecycle.vue'
import WorkloadConsole from '../components/WorkloadConsole.vue'
import { conditionType } from '../utils/resourceContent'

const consoleVisible = ref(false)
const route = useRoute()
const router = useRouter()
const kind = computed(() => (route.params.kind || '').toLowerCase())
const namespace = computed(() => (route.params.namespace === '_' ? '' : route.params.namespace) || '')
const resourceName = computed(() => route.params.name || '')

const endpoints = ref(null)
const loadingEndpoints = ref(false)
const endpointsError = ref('')

async function loadEndpoints() {
  if (kind.value !== 'service') return
  loadingEndpoints.value = true
  endpointsError.value = ''
  try {
    endpoints.value = await http.get('/service-endpoints', { params: { namespace: namespace.value, name: resourceName.value } })
  } catch (e) { endpoints.value = null; endpointsError.value = e.response?.data?.error || e.message } finally { loadingEndpoints.value = false }
}

const endpointRows = computed(() => {
  const rows = []
  for (const s of endpoints.value?.subsets || []) {
    for (const a of s.addresses || []) {
      rows.push({ ...a, port: (s.ports || []).map((p) => p.port).join(',') || '-', ready: true })
    }
    for (const a of s.notReadyAddresses || []) {
      rows.push({ ...a, port: (s.ports || []).map((p) => p.port).join(',') || '-', ready: false })
    }
  }
  return rows
})

function gotoKind(k, name, ns) {
  if (!k) return
  const wk = ['deployment', 'statefulset', 'daemonset']
  if (wk.includes(k)) router.push(`/workloads/${k}/${ns}/${name}`)
  else router.push(`/detail/${k}/${ns || '_'}/${encodeURIComponent(name)}`)
}
onMounted(loadEndpoints)

const extraTabs = computed(() => {
  if (kind.value === 'pod') return [{ name: 'lifecycle', label: '生命周期' }]
  return []
})

function nodeResources(detail) {
  const cap = detail.status?.capacity || {}
  const alloc = detail.status?.allocatable || {}
  const keys = new Set([...Object.keys(cap), ...Object.keys(alloc)])
  return [...keys].map((k) => ({ key: k, capacity: cap[k] || '-', allocatable: alloc[k] || '-' }))
}
</script>

<style scoped>
.rd-card { background: var(--ks-bg-light, #f9fbfd); border-radius: 8px; padding: 16px; }
.rd-card-title { font-weight: 600; font-size: 14px; margin-bottom: 12px; }
.mono-sm { font-family: monospace; font-size: 12px; }
.ing-rule { margin-bottom: 14px; }
.res-line { font-size: 12px; font-family: monospace; }
.ing-host { font-size: 13px; margin-bottom: 6px; }
</style>
