<template>
  <ResourceListView ref="list" kind="pod" endpoint="/pods" :columns="columns" :creatable="false" :ops-width="230" @loaded="loadMetrics">
    <template #col-cpu="{row}">{{metricsMap[row.namespace+'/'+row.name]?.cpuCores || '—'}}</template>
    <template #col-memory="{row}">{{metricsMap[row.namespace+'/'+row.name]?.memBytes || '—'}}</template>
    <template #ops="{row,edit,remove}"><el-button link type="primary" @click.stop="openConsole(row)">日志 / 终端</el-button><el-button link type="primary" @click.stop="edit(row.name,row.namespace)">YAML</el-button><el-button link type="danger" @click.stop="remove(row.name,row.namespace)">删除</el-button></template>
    <template #extra><WorkloadConsole :key="selected.namespace+'/'+selected.name" v-model="consoleVisible" kind="pod" :namespace="selected.namespace" :name="selected.name" /></template>
  </ResourceListView>
</template>
<script setup>
import {computed,ref} from 'vue'
import {useRoute} from 'vue-router'
import ResourceListView from '../components/ResourceListView.vue'
import WorkloadConsole from '../components/WorkloadConsole.vue'
import {usePodMetrics} from '../composables/useMetrics'
const route=useRoute(),list=ref(null),consoleVisible=ref(false),selected=ref({namespace:'',name:''})
const {items:metrics,load}=usePodMetrics()
const metricsMap=computed(()=>Object.fromEntries(metrics.value.map(item=>[item.namespace+'/'+item.name,item])))
function loadMetrics(){load(String(route.params.namespace||''))}
function openConsole(row){selected.value={namespace:row.namespace,name:row.name};consoleVisible.value=true}
const columns=[
 {prop:'name',label:'名称',primary:true,minWidth:260,sortable:true},
 {prop:'namespace',label:'项目',width:150},
 {prop:'status',label:'状态',type:'status',width:140},
 {prop:'restarts',label:'重启次数',width:100,sortable:true},
 {prop:'node',label:'节点',width:160},
 {prop:'ip',label:'Pod IP',width:140},
 {prop:'cpu',label:'CPU',width:100},
 {prop:'memory',label:'内存',width:120},
 {prop:'images',label:'镜像',type:'list',minWidth:240},
]
</script>
