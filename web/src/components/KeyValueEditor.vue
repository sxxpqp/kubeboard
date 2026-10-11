<template>
  <div class="kv-editor">
    <div v-for="(item, idx) in items" :key="idx" class="kv-row">
      <el-input v-model="item.key" :placeholder="keyPlaceholder" style="width: 180px" />
      <span class="kv-eq">=</span>
      <el-input v-model="item.value" :placeholder="valuePlaceholder" style="width: 240px" />
      <el-button link type="danger" size="small" @click="remove(idx)">删除</el-button>
    </div>
    <el-button link type="primary" size="small" :icon="'Plus'" @click="add">添加</el-button>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  keyPlaceholder: { type: String, default: '键' },
  valuePlaceholder: { type: String, default: '值' },
})
const emit = defineEmits(['update:modelValue'])

const items = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

function add() { emit('update:modelValue', [...props.modelValue, { key: '', value: '' }]) }
function remove(idx) {
  const next = [...props.modelValue]
  next.splice(idx, 1)
  emit('update:modelValue', next)
}
</script>

<style scoped>
.kv-editor { width: 100%; }
.kv-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.kv-eq { color: var(--ks-text-secondary); }
</style>
