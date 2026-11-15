<template>
  <ResourceDetailPage wide default-tab="runtime" :kind-label="label" :extra-tabs="tabs">
    <template #operation-bar="{reload}"><WorkloadActions :kind="kind" :namespace="namespace" :name="name" @changed="reload(); runtime?.refresh()" /></template>
    <template #tab-runtime><WorkloadRuntime ref="runtime" :kind="kind" :namespace="namespace" :name="name" /></template>
    <template #tab-services><WorkloadNetwork :kind="kind" :namespace="namespace" :name="name" type="service" /></template>
    <template #tab-ingresses><WorkloadNetwork :kind="kind" :namespace="namespace" :name="name" type="ingress" /></template>
  </ResourceDetailPage>
</template>
<script setup>
import {computed,ref} from 'vue'
import {useRoute} from 'vue-router'
import ResourceDetailPage from '../components/ResourceDetailPage.vue'
import WorkloadActions from '../components/WorkloadActions.vue'
import WorkloadRuntime from '../components/WorkloadRuntime.vue'
import WorkloadNetwork from '../components/WorkloadNetwork.vue'
const route=useRoute(),runtime=ref(null)
const kind=computed(()=>String(route.params.kind)),namespace=computed(()=>String(route.params.namespace)),name=computed(()=>String(route.params.name))
const label=computed(()=>({deployment:'部署',statefulset:'有状态副本集',daemonset:'守护进程集'}[kind.value]||kind.value))
const tabs=[{name:'runtime',label:'运行时'},{name:'services',label:'服务'},{name:'ingresses',label:'应用路由'}]
</script>
