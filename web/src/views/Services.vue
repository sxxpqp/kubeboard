<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-select v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 170px">
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <el-button type="primary" :icon="'Plus'" @click="openCreate(serviceTemplate)">创建</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>
      <el-table
        :data="filtered" v-loading="loading" stripe class="ks-table"
        highlight-current-row :row-style="{ cursor: 'pointer' }" @row-click="openDetail"
      >
        <el-table-column prop="namespace" label="项目" width="130" sortable />
        <el-table-column prop="name" label="名称" min-width="160" sortable>
          <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column prop="type" label="类型" width="110">
          <template #default="{ row }">
            <el-tag size="small" :type="row.type === 'LoadBalancer' ? 'warning' : row.type === 'NodePort' ? 'success' : 'info'">{{ row.type }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="clusterIP" label="Cluster IP" width="140" />
        <el-table-column prop="externalIP" label="External IP" width="130" />
        <el-table-column label="端口" min-width="200">
          <template #default="{ row }">
            <el-tag v-for="(p, i) in row.ports" :key="i" size="small" type="info" style="margin-right: 4px">
              {{ p.port }}/{{ p.protocol }}<template v-if="p.nodePort">→{{ p.nodePort }}</template>
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="testConnectivity(row)">连通性</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row.name, row.namespace)">编辑</el-button>
            <el-button link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <!-- 连通性测试结果 -->
    <el-dialog v-model="connVisible" :title="`连通性测试: ${connTarget.namespace}/${connTarget.name}`" width="560px">
      <div v-loading="connLoading" style="min-height: 160px">
        <template v-if="conn">
          <div class="conn-head">
            <el-tag :type="conn.healthy ? 'success' : 'danger'" effect="dark" size="large">
              {{ conn.healthy ? '健康：有就绪后端' : '异常：无就绪后端' }}
            </el-tag>
            <span class="conn-type">{{ conn.type }}</span>
          </div>
          <el-descriptions :column="2" border size="small" style="margin-bottom: 16px">
            <el-descriptions-item label="Cluster IP">{{ conn.clusterIP || '-' }}</el-descriptions-item>
            <el-descriptions-item label="External Name">{{ conn.externalName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="后端 Pod">{{ conn.backendPods }}</el-descriptions-item>
            <el-descriptions-item label="就绪 Pod">{{ conn.readyPods }}</el-descriptions-item>
            <el-descriptions-item label="Endpoints">{{ conn.endpointAddrs }}</el-descriptions-item>
            <el-descriptions-item label="选择器">
              <span v-if="Object.keys(conn.selector || {}).length">
                <el-tag v-for="(v, k) in conn.selector" :key="k" size="small" style="margin: 0 4px 2px 0">{{ k }}={{ v }}</el-tag>
              </span>
              <span v-else>-</span>
            </el-descriptions-item>
          </el-descriptions>
          <div class="conn-title">端口</div>
          <el-table :data="conn.ports || []" size="small">
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="port" label="端口" width="80" />
            <el-table-column prop="targetPort" label="目标端口" width="100" />
            <el-table-column prop="protocol" label="协议" width="80" />
            <el-table-column prop="nodePort" label="NodePort" width="100">
              <template #default="{ row }">{{ row.nodePort || '-' }}</template>
            </el-table-column>
          </el-table>
          <div class="conn-hint">
            {{ conn.readyPods > 0
              ? `✓ ${conn.readyPods} 个就绪后端，Service 可正常转发流量`
              : '✗ 无就绪后端，流量将无法转发（检查 Pod 是否 Ready）' }}
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="loadConn" :loading="connLoading">重新测试</el-button>
        <el-button type="primary" @click="connVisible = false">关闭</el-button>
      </template>
    </el-dialog>

    <YamlDialog
      v-model="dialogVisible"
      :mode="dialogMode"
      kind="service"
      :namespace="currentNamespace"
      :name="currentName"
      :template="currentTemplate"
      @saved="load"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import http from '../api/http'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from '../components/YamlDialog.vue'
import { ElMessage } from 'element-plus'

const router = useRouter()
const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)
const connVisible = ref(false)
const connLoading = ref(false)
const conn = ref(null)
const connTarget = ref({ namespace: '', name: '' })

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.toLowerCase().includes(keyword.value.toLowerCase()))
)

const serviceTemplate = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  selector:
    app: my-app
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: ClusterIP
`

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openCreate, openEdit, remove,
} = useResourceCRUD('service', load)

function testConnectivity(row) {
  connTarget.value = { namespace: row.namespace, name: row.name }
  conn.value = null
  connVisible.value = true
  loadConn()
}
async function loadConn() {
  connLoading.value = true
  try {
    conn.value = await http.get('/services/connectivity', {
      params: { namespace: connTarget.value.namespace, name: connTarget.value.name },
    })
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '连通性测试失败')
  } finally {
    connLoading.value = false
  }
}

function openDetail(row) {
  router.push(`/detail/service/${row.namespace}/${row.name}`)
}

async function load() {
  loading.value = true
  try {
    const params = {}
    if (namespace.value) params.namespace = namespace.value
    const data = await http.get('/services', { params })
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

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.image-text { font-size: 12px; color: var(--ks-text-secondary); }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
.conn-head { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.conn-type { color: var(--ks-text-secondary); }
.conn-title { font-weight: 600; font-size: 13px; margin-bottom: 8px; }
.conn-hint { margin-top: 14px; font-size: 13px; }
</style>
