<template>
  <section v-for="section in sections" :key="section.title" class="resource-section">
    <h3>{{ section.title }}</h3>
    <el-table v-if="section.columns" :data="section.rows" class="ks-table" empty-text="无条目">
      <el-table-column v-for="[key, label] in section.columns" :key="key" :prop="key" :label="label" min-width="140"><template #default="{ row }"><span class="content-value">{{ row[key] ?? '—' }}</span></template></el-table-column>
    </el-table>
    <dl v-else class="content-attributes"><template v-for="row in section.rows" :key="row.name"><dt>{{ row.name }}</dt><dd><router-link v-if="referenceTarget(row)" :to="referenceTarget(row)">{{ row.value }} →</router-link><template v-else>{{ row.value }}</template></dd></template></dl>
  </section>
</template>
<script setup>
import { computed } from 'vue'
import { resourceObject, resourceSections } from '../utils/resourceContent'
const props = defineProps({ detail: { type: Object, required: true } })
function referenceTarget(row) {
  const kinds = { '绑定 PV': 'pv', '存储类': 'storageclass' }
  if (!kinds[row.name] || ['未配置', '无', ''].includes(row.value)) return ''
  return `/detail/${kinds[row.name]}/_/${encodeURIComponent(row.value)}`
}
const sections = computed(() => resourceSections(resourceObject(props.detail)))
</script>
<style scoped>
.resource-section { margin-bottom: 24px; padding: 20px; border: 1px solid var(--ks-border); border-radius: 6px; }
h3 { margin: 0 0 20px; font-size: 16px; }
.content-attributes { display: grid; grid-template-columns: 150px minmax(0, 1fr); margin: 0; font-size: 13px; }
dt, dd { margin: 0; padding: 12px 0; border-bottom: 1px solid var(--ks-border); white-space: pre-wrap; overflow-wrap: anywhere; }
dt { color: var(--ks-text-secondary); }
.content-value { white-space: pre-wrap; overflow-wrap: anywhere; }
</style>
