<template>
  <div>
    <div class="ks-card">
      <el-alert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon style="margin-bottom: 16px"><el-button link @click="load">重试</el-button></el-alert>
      <div class="ks-toolbar">
        <el-select
          v-if="namespaceScoped && !route.params.namespace"
          v-model="namespace" placeholder="全部项目" clearable @change="onNsChange" style="width: 170px"
        >
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <el-select v-if="statusColumn" v-model="selectedStatus" placeholder="全部状态" clearable style="width:160px"><el-option v-for="status in statusOptions" :key="status" :label="status" :value="status" /></el-select>
        <div style="flex: 1"></div>
        <slot name="toolbar-extra" />
        <el-button v-if="creatable" type="primary" :icon="'Plus'" @click="openCreate(template, namespace)">创建</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>

        <el-table
        :data="paged" v-loading="loading" stripe class="ks-table" :empty-text="keyword || selectedStatus ? '没有匹配结果，请清除筛选' : '暂无资源'"
        :highlight-current-row="detailEnabled" :row-style="detailEnabled ? { cursor: 'pointer' } : undefined"
        @row-click="detailEnabled && onRowClick($event)"
      >
        <el-table-column
          v-for="col in displayColumns" :key="col.prop"
          :prop="col.prop" :label="col.label"
          :width="col.width" :min-width="col.minWidth || 160"
          :align="col.align || 'left'" :sortable="col.sortable"
        >
          <template #default="{ row }">
            <slot :name="`col-${col.prop}`" :row="row">
              <span v-if="col.type === 'status'" class="ks-status">
                <span class="dot" :class="row[col.prop]"></span>{{ row[col.prop] }}
              </span>
              <el-tag v-else-if="col.type === 'tag'" size="small" :type="col.tagType || 'info'">{{ row[col.prop] }}</el-tag>
              <div v-else-if="col.type === 'list'" class="list-cell">
                <div v-for="(v, i) in asArray(row[col.prop])" :key="i">{{ v }}</div>
              </div>
              <div v-else-if="col.type === 'tags'" class="list-cell">
                <el-tag v-for="(v, i) in asArray(row[col.prop])" :key="i" size="small" type="info" style="margin: 0 4px 2px 0">{{ v }}</el-tag>
              </div>
              <span v-else-if="col.type === 'time'">{{ fmtTime(row[col.prop]) }}</span>
              <el-tag v-else-if="col.type === 'bool'" size="small" :type="row[col.prop] ? 'success' : 'info'">{{ row[col.prop] ? '是' : '否' }}</el-tag>
              <span v-else :class="{ 'resource-name': col.primary }">{{ row[col.prop] }}</span>
            </slot>
          </template>
        </el-table-column>

        <el-table-column label="操作" :width="opsWidth" fixed="right">
          <template #default="{ row }">
            <slot name="ops" :row="row" :edit="openEdit" :remove="remove">
              <el-button v-if="editable" link type="primary" size="small" @click.stop="openEdit(row.name, row.namespace)">编辑</el-button>
              <el-button v-if="deletable" link type="danger" size="small" @click.stop="remove(row.name, row.namespace)">删除</el-button>
            </slot>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">
        <span>共 {{ filtered.length }} 项</span>
        <el-pagination
          v-if="filtered.length > pageSize"
          v-model:current-page="page"
          :page-size="pageSize" :total="filtered.length"
          layout="prev, pager, next, jumper" background size="small"
        />
      </div>
    </div>

    <YamlDialog
      v-if="creatable || editable"
      v-model="dialogVisible" :mode="dialogMode" :kind="kind"
      :namespace="currentNamespace" :name="currentName" :template="currentTemplate" :namespaces="nsList" @saved="load"
    />
    <slot name="extra" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import { projectPath } from '../router/projectScope'
import { useResourceCRUD } from '../api/crud'
import YamlDialog from './YamlDialog.vue'

const props = defineProps({
  kind: { type: String, required: true },
  endpoint: { type: String, required: true },
  endpointParams: { type: Object, default: () => ({}) },
  columns: { type: Array, required: true },
  namespaceScoped: { type: Boolean, default: true },
  creatable: { type: Boolean, default: true },
  editable: { type: Boolean, default: true },
  deletable: { type: Boolean, default: true },
  template: { type: String, default: '' },
  opsWidth: { type: Number, default: 140 },
  filterFn: { type: Function, default: null },
  detailEnabled: { type: Boolean, default: true },
  detailKind: { type: String, default: '' },
  detailFields: { type: Array, default: () => [] },
})

const emit = defineEmits(['loaded'])

const route = useRoute()
const router = useRouter()
const namespace = ref(String(route.params.namespace || ''))
function onNsChange(v) {
  router.replace({ path: projectPath(route.path, v), query: route.query })
}
watch(() => route.params.namespace, value => { namespace.value = String(value || ''); load() })
const keyword = ref(String(route.query.search || ''))
const selectedStatus=ref(String(route.query.status||''))
const statusColumn=computed(()=>props.columns.find(c=>c.type==='status'))
const statusOptions=computed(()=>[...new Set(items.value.map(i=>i[statusColumn.value?.prop]).filter(Boolean))])
const items = ref([])
const namespaces = ref([])
const loading = ref(true)
const loadError = ref('')
function onRowClick(row) {
  const kind = props.detailKind || props.kind
  router.push({ path: `/detail/${kind}/${encodeURIComponent(row.namespace || '_')}/${encodeURIComponent(row.name)}` })
}

const displayColumns = computed(() => {
  const columns = props.columns.filter(col => !namespace.value || col.prop !== 'namespace')
  return [...columns.filter(col => col.primary), ...columns.filter(col => !col.primary)]
})

const filtered = computed(() => {
  let list = items.value
  if (keyword.value) {
    const k = keyword.value.toLowerCase()
    list = list.filter((i) => i.name?.toLowerCase().includes(k))
  }
  if(selectedStatus.value && statusColumn.value)list=list.filter(i=>i[statusColumn.value.prop]===selectedStatus.value)
  return props.filterFn ? props.filterFn(list) : list
})

// 前端分页（P0-③ 第一步）：数据仍全量拉取（K8s List 语义），但渲染只出当前页，
// 避免 大列表一次渲染卡顿；后端分页（continue token）后续演进。
const pageSize = 20
const page = ref(Math.max(1,Number(route.query.page)||1))
const paged = computed(() => {
  const start = (page.value - 1) * pageSize
  return filtered.value.slice(start, start + pageSize)
})
// 过滤结果变化时回到第一页，避免停在超出范围的页码
watch([keyword,selectedStatus],()=>{page.value=1})
watch([keyword,selectedStatus,page],()=>router.replace({query:{...route.query,search:keyword.value||undefined,status:selectedStatus.value||undefined,page:page.value>1?page.value:undefined}}))

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openCreate, openEdit, remove, nsList,
} = useResourceCRUD(props.kind, load)

function asArray(v) {
  if (Array.isArray(v)) return v
  if (v == null) return []
  return [v]
}

function fmtTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

let loadSequence = 0
async function load() {
  const sequence = ++loadSequence
  loading.value = true
  try {
    const params = { ...props.endpointParams }
    if (props.namespaceScoped && namespace.value) params.namespace = namespace.value
    const data = await http.get(props.endpoint, { params })
    if (sequence !== loadSequence) return
    loadError.value = ''
    items.value = data.items || []
    page.value=Math.min(page.value,Math.max(1,Math.ceil(filtered.value.length/pageSize)))
    emit('loaded', items.value)
  } catch (e) {
    if (sequence === loadSequence) loadError.value = '资源加载失败：' + (e?.message || '请重试')
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

onMounted(async () => {
  if (props.namespaceScoped) {
    try {
      const nsData = await http.get('/namespaces')
      namespaces.value = nsData.items.map((i) => i.name)
    } catch { /* ignore */ }
  }
  await load()
})

defineExpose({ load, items })
</script>

<style scoped>
.resource-name { color: var(--ks-primary); font-weight: 500; }
.list-cell { font-size: 12px; color: var(--ks-text-secondary); line-height: 1.8; }
.table-footer { padding: 12px 4px 0; color: var(--ks-text-secondary); font-size: 12px; }
</style>
