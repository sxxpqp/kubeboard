<template>
  <div ref="el" :style="{ height: height + 'px', width: '100%' }"></div>
</template>

<script setup>
import { onMounted, onUnmounted, ref, watch, nextTick } from 'vue'
import * as echarts from 'echarts'

const props = defineProps({
  option: { type: Object, required: true },
  height: { type: Number, default: 280 },
})

const el = ref(null)
let chart = null
let ro = null

function render() {
  if (!chart && el.value) {
    chart = echarts.init(el.value)
  }
  if (chart) chart.setOption(props.option, true)
}

watch(() => props.option, () => render(), { deep: true })

onMounted(async () => {
  await nextTick()
  render()
  ro = new ResizeObserver(() => chart && chart.resize())
  if (el.value) ro.observe(el.value)
})

onUnmounted(() => {
  if (ro) ro.disconnect()
  if (chart) { chart.dispose(); chart = null }
})
</script>
