<template>
  <el-dialog :model-value="modelValue" title="日志 / 终端" width="min(1100px, 94vw)" destroy-on-close @update:model-value="emit('update:modelValue', $event)">
    <p>{{ namespace }} / {{ name }}</p>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <div class="console-controls" v-loading="loading">
      <el-select v-model="podName" placeholder="选择 Pod"><el-option v-for="p in pods" :key="p.name" :label="p.name" :value="p.name" /></el-select>
      <el-select v-model="containerName" placeholder="选择容器"><el-option v-for="c in containers" :key="c" :label="c" :value="c" /></el-select>
      <el-button @click="refresh">刷新</el-button>
    </div>
    <el-empty v-if="!loading && !error && !pods.length" description="当前工作负载没有可用 Pod" />
    <el-tabs v-model="tab">
      <el-tab-pane label="日志" name="logs" />
      <el-tab-pane label="终端" name="terminal" />
    </el-tabs>
    <LogViewer v-if="modelValue && podName && containerName && tab === 'logs'" :key="podName + containerName" embedded :model-value="true" :namespace="namespace" :pod-name="podName" :containers="[containerName]" :initial-container="containerName" />
    <TerminalView v-if="modelValue && podName && containerName && tab === 'terminal'" :key="podName + containerName" embedded :model-value="true" :namespace="namespace" :pod-name="podName" :containers="[containerName]" :initial-container="containerName" />
  </el-dialog>
</template>
<script setup>
import { computed, ref, watch } from 'vue'
import http from '../api/http'
import {load} from 'js-yaml'
import LogViewer from './LogViewer.vue'
import TerminalView from './TerminalView.vue'
const props = defineProps({ modelValue:Boolean, kind:String, namespace:String, name:String, initialPod:String })
const emit = defineEmits(['update:modelValue'])
const pods=ref([]), podName=ref(''), containerName=ref(''), loading=ref(false), error=ref(''), tab=ref('logs')
const containers=computed(()=>pods.value.find(p=>p.name===podName.value)?.containers || [])
watch(containers, list=>{ if(!list.includes(containerName.value)) containerName.value=list[0] || '' })
let sequence=0
async function refresh(){ const id=++sequence; loading.value=true; error.value=''; try{ let data;if(props.kind==='pod'){const pod=load(await http.get('/resource',{params:{kind:'pod',namespace:props.namespace,name:props.name},responseType:'text'}));data={items:[{name:props.name,containers:[...(pod.spec.containers||[]),...(pod.spec.initContainers||[]),...(pod.spec.ephemeralContainers||[])].map(c=>c.name)}]}}else data=await http.get('/workloads/pods',{params:{kind:props.kind,namespace:props.namespace,name:props.name}}); if(id!==sequence||!props.modelValue)return; pods.value=data.items || []; if(!pods.value.some(p=>p.name===podName.value))podName.value=pods.value[0]?.name || '' }catch(e){if(id===sequence)error.value=e.response?.data?.error||e.message}finally{if(id===sequence)loading.value=false} }
watch(()=>props.modelValue,value=>{if(value){podName.value=props.initialPod || '';refresh()}else{sequence++;loading.value=false}},{immediate:true})
</script>
<style scoped>
.console-controls { display:flex; gap:12px; margin:16px 0; flex-wrap:wrap; }
</style>
