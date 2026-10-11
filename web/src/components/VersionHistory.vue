<template>
  <div v-loading="loading">
    <el-alert
      v-if="!loading && !items.length"
      type="info" :closable="false"
      title="暂无版本历史（该工作负载尚未发生过更新）"
    />
    <el-table v-else :data="items" size="small" :row-class-name="rowClass">
      <el-table-column prop="revision" label="版本" width="80" align="center" sortable>
        <template #default="{ row }">
          <b>rev {{ row.revision }}</b>
          <el-tag v-if="row.current" size="small" type="success" effect="dark" style="margin-left: 6px">当前</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="镜像" min-width="240">
        <template #default="{ row }">
          <div v-for="img in row.images" :key="img" class="img-line">{{ img }}</div>
        </template>
      </el-table-column>
      <el-table-column label="副本" width="100" align="center">
        <template #default="{ row }">{{ row.ready }}/{{ row.replicas }}</template>
      </el-table-column>
      <el-table-column prop="changeCause" label="变更原因" min-width="160">
        <template #default="{ row }">{{ row.changeCause || '-' }}</template>
      </el-table-column>
      <el-table-column label="创建时间" width="160">
        <template #default="{ row }">{{ fmtTime(row.createdTime) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template #default="{ row }">
          <el-button
            v-if="!row.current" link type="primary" size="small"
            @click="confirmRollback(row)"
          >回退</el-button>
          <span v-else class="muted">-</span>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'

const props = defineProps({
  kind: { type: String, required: true },
  namespace: { type: String, default: '' },
  name: { type: String, required: true },
})
const emit = defineEmits(['rolledback'])

const items = ref([])
const loading = ref(false)

function fmtTime(ts) {
  if (!ts) return '-'
  return new Date(ts * 1000).toLocaleString('zh-CN', { hour12: false })
}
function rowClass({ row }) {
  return row.current ? 'current-row' : ''
}

async function load() {
  loading.value = true
  try {
    const d = await http.get('/workloads/revisions', {
      params: { kind: props.kind, namespace: props.namespace, name: props.name },
    })
    items.value = d.items || []
  } catch (e) {
    ElMessage.error('获取版本历史失败: ' + (e?.message || ''))
  } finally {
    loading.value = false
  }
}

async function confirmRollback(row) {
  try {
    await ElMessageBox.confirm(
      `确定回退到版本 rev ${row.revision}？镜像将变为：${row.images.join(', ')}`,
      '回退确认',
      { type: 'warning', confirmButtonText: '确认回退', cancelButtonText: '取消' }
    )
  } catch {
    return
  }
  try {
    await http.post('/workloads/rollback', {
      kind: props.kind, namespace: props.namespace, name: props.name, revision: row.revision,
    })
    ElMessage.success(`已回退到 rev ${row.revision}`)
    emit('rolledback')
    await load()
  } catch (e) {
    ElMessage.error('回退失败: ' + (e?.message || ''))
  }
}

defineExpose({ load })
onMounted(load)
</script>

<style scoped>
.img-line { font-size: 12px; font-family: monospace; }
.muted { color: var(--ks-text-secondary); }
:deep(.current-row) { background: var(--ks-success-bg, #e8f8f2) !important; }
</style>
