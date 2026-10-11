<template>
  <ResourceDetailPage :extra-tabs="extraTabs">
    <template #overview-extra="{ detail }">
      <!-- ========== Pod 专属：容器 / 卷 / 调度 ========== -->
      <template v-if="kind === 'pod' && detail.spec">
        <div class="rd-card" style="margin-top:16px">
          <div class="rd-card-title">容器（{{ (detail.spec.containers || []).length }}）</div>
          <el-table :data="detail.spec.containers || []" size="small">
            <el-table-column prop="name" label="名称" width="140" />
            <el-table-column prop="image" label="镜像" min-width="240" />
            <el-table-column prop="imagePullPolicy" label="拉取策略" width="120" />
            <el-table-column label="命令" min-width="200">
              <template #default="{ row }"><span class="mono-sm">{{ [...(row.command||[]), ...(row.args||[])].join(' ') || '-' }}</span></template>
            </el-table-column>
            <el-table-column label="资源" width="200">
              <template #default="{ row }">
                <div class="res-line">CPU: {{ row.resources?.requests?.cpu || '-' }} / {{ row.resources?.limits?.cpu || '-' }}</div>
                <div class="res-line">MEM: {{ row.resources?.requests?.memory || '-' }} / {{ row.resources?.limits?.memory || '-' }}</div>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <div class="rd-card" style="margin-top:16px" v-if="(detail.spec.volumes || []).length">
          <div class="rd-card-title">卷（{{ detail.spec.volumes.length }}）</div>
          <el-table :data="detail.spec.volumes" size="small">
            <el-table-column prop="name" label="卷名" width="180" />
            <el-table-column label="类型" width="140">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ volType(row) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="来源" min-width="240">
              <template #default="{ row }"><span class="mono-sm">{{ volSource(row) }}</span></template>
            </el-table-column>
          </el-table>
        </div>

        <div class="rd-card" style="margin-top:16px">
          <div class="rd-card-title">调度信息</div>
          <el-descriptions :column="2" border size="small">
            <el-descriptions-item label="所在节点">
              <el-link v-if="detail.spec.nodeName" type="primary" @click="gotoKind('node', detail.spec.nodeName, '')">{{ detail.spec.nodeName }}</el-link>
              <span v-else class="muted">未调度</span>
            </el-descriptions-item>
            <el-descriptions-item label="调度器">{{ detail.spec.schedulerName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="服务账号">{{ detail.spec.serviceAccount || '-' }}</el-descriptions-item>
            <el-descriptions-item label="重启策略">{{ detail.spec.restartPolicy || '-' }}</el-descriptions-item>
            <el-descriptions-item label="QoS 等级">
              <el-tag size="small" :type="detail.status?.qosClass === 'Guaranteed' ? 'success' : (detail.status?.qosClass === 'Burstable' ? 'warning' : 'info')">
                {{ detail.status?.qosClass || '-' }}
              </el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="Pod IP"><span class="mono-sm">{{ detail.status?.podIP || '-' }}</span></el-descriptions-item>
            <el-descriptions-item label="节点选择" :span="2">
              <span v-if="Object.keys(detail.spec.nodeSelector || {}).length">
                <el-tag v-for="(v,k) in detail.spec.nodeSelector" :key="k" size="small" style="margin:0 4px 2px 0">{{ k }}={{ v }}</el-tag>
              </span>
              <span v-else class="muted">无</span>
            </el-descriptions-item>
          </el-descriptions>
        </div>
      </template>

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
            <template #default="{ row }"><el-tag :type="row.status === 'True' ? 'success' : 'warning'" size="small">{{ row.status }}</el-tag></template>
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
        <div v-loading="loadingEndpoints">
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

      <!-- ConfigMap/Secret 专属 -->
      <div v-if="(kind === 'configmap' || kind === 'secret') && detail.spec" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title">数据键（{{ (detail.spec.keys || []).length }}）</div>
        <el-table :data="dataRows(detail)" size="small">
          <el-table-column prop="key" label="键" width="220" />
          <el-table-column prop="value" label="值" min-width="300">
            <template #default="{ row }"><span class="mono-sm">{{ row.value }}</span></template>
          </el-table-column>
        </el-table>
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

      <!-- 存储专属 -->
      <div v-if="(kind === 'pvc' || kind === 'pv') && detail.spec" class="rd-card" style="margin-top:16px">
        <div class="rd-card-title">存储</div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="访问模式">{{ (detail.spec.accessModes || []).join(', ') }}</el-descriptions-item>
          <el-descriptions-item label="存储类">{{ detail.spec.storageClass || '-' }}</el-descriptions-item>
          <el-descriptions-item label="绑定卷">{{ detail.spec.volumeName || '-' }}</el-descriptions-item>
          <el-descriptions-item label="容量">{{ detail.spec.capacity || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="rd-card-title" style="margin-top:16px">绑定与使用</div>
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="绑定状态">
            <el-tag size="small" :type="detail.status?.phase === 'Bound' ? 'success' : 'warning'">{{ detail.status?.phase || '-' }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="实际容量">{{ fmtCapacity(detail.status?.capacity) }}</el-descriptions-item>
          <el-descriptions-item label="绑定卷" :span="2">
            <el-link v-if="detail.spec.volumeName" type="primary" @click="gotoKind('pv', detail.spec.volumeName, '')">{{ detail.spec.volumeName }}</el-link>
            <span v-else class="muted">未绑定</span>
          </el-descriptions-item>
        </el-descriptions>
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
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import ResourceDetailPage from '../components/ResourceDetailPage.vue'

const route = useRoute()
const router = useRouter()
const kind = computed(() => (route.params.kind || '').toLowerCase())
const namespace = computed(() => route.params.namespace || '')
const resourceName = computed(() => route.params.name || '')

const endpoints = ref(null)
const loadingEndpoints = ref(false)

async function loadEndpoints() {
  if (kind.value !== 'service') return
  loadingEndpoints.value = true
  try {
    endpoints.value = await http.get('/service-endpoints', { params: { namespace: namespace.value, name: resourceName.value } })
  } catch { endpoints.value = null } finally { loadingEndpoints.value = false }
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
  else router.push(`/detail/${k}/${ns}/${name}`)
}
function volType(v) {
  if (v.configMap) return 'ConfigMap'
  if (v.secret) return 'Secret'
  if (v.pvc) return 'PVC'
  if (v.hostPath) return 'HostPath'
  if (v.emptyDir) return 'EmptyDir'
  return '其他'
}
function volSource(v) {
  if (v.configMap) return v.configMap
  if (v.secret) return v.secret
  if (v.pvc) return v.pvc
  if (v.hostPath) return v.hostPath
  if (v.emptyDir) return '临时目录'
  return '-'
}
function fmtCapacity(cap) {
  if (!cap) return '-'
  if (typeof cap === 'string') return cap
  return cap.storage || JSON.stringify(cap)
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
function dataRows(detail) {
  const data = detail.spec?.data || {}
  return Object.keys(data).map((k) => ({ key: k, value: data[k] }))
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
