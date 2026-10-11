<template>
  <div>
    <div class="ks-card">
      <div class="ks-toolbar">
        <el-select
          v-if="namespaceScoped"
          v-model="namespace" placeholder="全部项目" clearable @change="load" style="width: 170px"
        >
          <el-option v-for="ns in namespaces" :key="ns" :label="ns" :value="ns" />
        </el-select>
        <el-input v-model="keyword" placeholder="搜索名称" clearable style="width: 200px" :prefix-icon="'Search'" />
        <div style="flex: 1"></div>
        <slot name="toolbar-extra" />
        <el-button v-if="creatable" type="primary" :icon="'Plus'" @click="openCreate(template)">创建</el-button>
        <el-button :icon="'Refresh'" @click="load" />
      </div>

      <el-table :data="filtered" v-loading="loading" stripe class="ks-table">
        <el-table-column
          v-for="col in columns" :key="col.prop"
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
            <slot name="ops" :row="row">
              <el-button v-if="editable" link type="primary" size="small" @click="openEdit(row.name, row.namespace)">编辑</el-button>
              <el-button v-if="deletable" link type="danger" size="small" @click="remove(row.name, row.namespace)">删除</el-button>
            </slot>
          </template>
        </el-table-column>
      </el-table>
      <div class="table-footer">共 {{ filtered.length }} 项</div>
    </div>

    <YamlDialog
      v-if="creatable || editable"
      v-model="dialogVisible" :mode="dialogMode" :kind="kind"
      :namespace="currentNamespace" :name="currentName" :template="currentTemplate" @saved="load"
    />
    <slot name="extra" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import http from '../api/http'
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
})

const emit = defineEmits(['loaded'])

const namespace = ref('')
const keyword = ref('')
const items = ref([])
const namespaces = ref([])
const loading = ref(true)

const filtered = computed(() => {
  let list = items.value
  if (keyword.value) {
    const k = keyword.value.toLowerCase()
    list = list.filter((i) => i.name?.toLowerCase().includes(k))
  }
  return props.filterFn ? props.filterFn(list) : list
})

const {
  dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
  openCreate, openEdit, remove,
} = useResourceCRUD(props.kind, load)

function asArray(v) {
  if (Array.isArray(v)) return v
  if (v == null) return []
  return [v]
}

function fmtTime(ts) {
  return ts ? new Date(ts * 1000).toLocaleString() : '-'
}

async function load() {
  loading.value = true
  try {
    const params = { ...props.endpointParams }
    if (props.namespaceScoped && namespace.value) params.namespace = namespace.value
    const data = await http.get(props.endpoint, { params })
    items.value = data.items || []
    emit('loaded', items.value)
  } finally {
    loading.value = false
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
