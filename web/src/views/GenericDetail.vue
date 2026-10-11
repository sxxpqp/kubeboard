<template>
  <ResourceDetailPage :extra-tabs="extraTabs">
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
            <template #default="{ row }"><el-tag :type="row.status === 'True' ? 'success' : 'warning'" size="small">{{ row.status }}</el-tag></template>
          </el-table-column>
          <el-table-column prop="reason" label="原因" width="160" />
          <el-table-column prop="message" label="消息" min-width="220" />
        </el-table>
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
      </div>
    </template>
  </ResourceDetailPage>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ResourceDetailPage from '../components/ResourceDetailPage.vue'

const route = useRoute()
const kind = computed(() => (route.params.kind || '').toLowerCase())

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
.ing-host { font-size: 13px; margin-bottom: 6px; }
</style>
