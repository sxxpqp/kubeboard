<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-radio-group v-model="kind" @change="load">
          <el-radio-button value="deployment">部署</el-radio-button>
          <el-radio-button value="statefulset">有状态副本集</el-radio-button>
          <el-radio-button value="daemonset">守护进程集</el-radio-button>
        </el-radio-group>
        <el-select v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 170px">
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <el-button type="primary" :icon="'Plus'" @click="openCreate(workloadTemplate)">创建</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>
      <el-table :data="filtered" v-loading="loading" stripe class="ks-table">
        <el-table-column prop="namespace" label="项目" width="130" sortable />
        <el-table-column prop="name" label="名称" min-width="160" sortable>
          <template #default="{ row }"><span class="resource-name">{{ row.name }}</span></template>
        </el-table-column>
        <el-table-column prop="ready" label="就绪副本" width="95" align="center">
          <template #default="{ row }">
            <span :class="readyClass(row)">{{ row.ready }}</span>
          </template>
        </el-table-column>
        <el-table-column label="镜像" min-width="240">
          <template #default="{ row }">
            <div v-for="img in row.images" :key="img" class="image-text">{{ img }}</div>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="150" sortable prop="creationTime">
          <template #default="{ row }">{{ fmtTime(row.creationTime) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="290" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" size="small" @click="openScale(row)">伸缩</el-button>
            <el-button link type="primary" size="small" @click="openImage(row)">镜像</el-button>
            <el-button link type="primary" size="small" @click="openLogs(row)">日志</el-button>
            <el-button link type="warning" size="small" @click="restart(row)">重启</el-button>
            <el-button link type="primary" size="small" @click="openEdit(row.name, row.namespace)">编辑</el-button>
            <el-button link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <YamlDialog
      v-model="dialogVisible" :mode="dialogMode" :kind="kind"
      :namespace="currentNamespace" :name="currentName" :template="currentTemplate" @saved="load"
    />

    <!-- 伸缩 -->
    <el-dialog v-model="scaleVisible" title="调整副本数" width="420px">
      <div class="dialog-label">工作负载: <b>{{ target.namespace }}/{{ target.name }}</b></div>
      <el-input-number v-model="scaleReplicas" :min="0" :max="100" />
      <template #footer>
        <el-button @click="scaleVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="doScale">确定</el-button>
      </template>
    </el-dialog>

    <!-- 镜像 -->
    <el-dialog v-model="imageVisible" title="更新镜像" width="520px">
      <div class="dialog-label">工作负载: <b>{{ target.namespace }}/{{ target.name }}</b></div>
      <el-input v-model="newImage" placeholder="例如 nginx:1.25" />
      <div class="dialog-hint">将更新该工作负载所有容器的镜像。</div>
      <template #footer>
        <el-button @click="imageVisible = false">取消</el-button>
        <el-button type="primary" :loading="acting" @click="doImage">确定</el-button>
      </template>
    </el-dialog>

    <LogViewer v-model="logVisible" :namespace="target.namespace" :pod-name="target.name" :containers="target.containers" />

    <!-- 选择 Pod 查看日志 -->
    <el-dialog v-model="podPickerVisible" title="选择 Pod" width="560px">
      <el-table :data="podList" size="small" @row-click="pickPod" style="cursor: pointer">
        <el-table-column prop="name" label="Pod 名称" min-width="280" />
        <el-table-column prop="status" label="状态" width="100" />
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from '../components/YamlDialog.vue'
import LogViewer from '../components/LogViewer.vue'

const kind = ref('deployment')
const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)
const acting = ref(false)

const scaleVisible = ref(false)
const imageVisible = ref(false)
const logVisible = ref(false)
const scaleReplicas = ref(1)
const newImage = ref('')
const target = ref({ namespace: '', name: '', containers: [] })

const filtered = computed(() =>
  items.value.filter((i) => !keyword.value || i.name.toLowerCase().includes(keyword.value.toLowerCase()))
)

const workloadTemplate = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  namespace: default
  labels:
    app: my-app
spec:
  replicas: 2
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: my-app
    spec:
      containers:
        - name: my-app
          image: nginx:latest
          ports:
            - containerPort: 80
`

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openCreate, openEdit, remove,
} = useResourceCRUD('deployment', load)

function fmtTime(ts) { return ts ? new Date(ts * 1000).toLocaleString() : '-' }
function readyClass(row) {
  const [r, d] = String(row.ready).split('/').map(Number)
  return r === d ? 'ready-ok' : 'ready-warn'
}

function openScale(row) {
  target.value = { namespace: row.namespace, name: row.name, containers: row.images }
  scaleReplicas.value = Number(String(row.ready).split('/')[1]) || row.replicas || 1
  scaleVisible.value = true
}
async function doScale() {
  acting.value = true
  try {
    await http.post('/workloads/scale', {
      kind: kind.value, namespace: target.value.namespace, name: target.value.name, replicas: scaleReplicas.value,
    })
    ElMessage.success('副本数已更新')
    scaleVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  } finally { acting.value = false }
}

function openImage(row) {
  target.value = { namespace: row.namespace, name: row.name, containers: row.images }
  newImage.value = row.images[0] || ''
  imageVisible.value = true
}
async function doImage() {
  acting.value = true
  try {
    await http.post('/workloads/image', {
      kind: kind.value, namespace: target.value.namespace, name: target.value.name, image: newImage.value,
    })
    ElMessage.success('镜像已更新')
    imageVisible.value = false
    load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  } finally { acting.value = false }
}

const podPickerVisible = ref(false)
const podList = ref([])

async function openLogs(row) {
  try {
    const data = await http.get('/workloads/pods', { params: { kind: kind.value, namespace: row.namespace, name: row.name } })
    if (!data.items.length) {
      ElMessage.info('该工作负载当前没有运行中的 Pod')
      return
    }
    podList.value = data.items
    target.value = { namespace: row.namespace, name: '', containers: [] }
    podPickerVisible.value = true
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '查询 Pod 失败')
  }
}

function pickPod(pod) {
  podPickerVisible.value = false
  target.value = { namespace: pod.namespace, name: pod.name, containers: pod.containers }
  logVisible.value = true
}

async function restart(row) {
  try {
    await ElMessageBox.confirm(`确定重启 ${row.name} 吗？将触发滚动重启。`, '重启确认', { type: 'warning' })
  } catch { return }
  try {
    await http.post('/workloads/restart', { kind: kind.value, namespace: row.namespace, name: row.name })
    ElMessage.success('已触发重启')
    load()
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '操作失败')
  }
}

async function load() {
  loading.value = true
  try {
    const params = { kind: kind.value }
    if (namespace.value) params.namespace = namespace.value
    const data = await http.get('/workloads', { params })
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
.image-text { font-size: 12px; color: var(--ks-text-secondary); line-height: 1.8; }
.ready-ok { color: var(--ks-success); font-weight: 600; }
.ready-warn { color: var(--ks-danger); font-weight: 600; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
.dialog-label { margin-bottom: 10px; color: var(--ks-text-secondary); }
.dialog-hint { margin-top: 8px; font-size: 12px; color: var(--ks-text-secondary); }
</style>
