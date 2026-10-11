<template>
  <ResourceListView
    kind="hpa" endpoint="/hpas" :namespace-scoped="true"
    :columns="columns" :creatable="true" :deletable="true" :editable="false"
    :template="template" detail-enabled :detail-fields="detailFields"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 160, sortable: true, primary: true },
  { prop: 'target', label: '目标工作负载', minWidth: 200 },
  { prop: 'metric', label: '指标', width: 140 },
  { prop: 'min', label: '最小', width: 70, align: 'center' },
  { prop: 'max', label: '最大', width: 70, align: 'center' },
  { prop: 'current', label: '当前', width: 70, align: 'center' },
  { prop: 'desired', label: '期望', width: 70, align: 'center' },
]

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'target', label: '目标工作负载' },
  { key: 'metric', label: '伸缩指标' },
  { key: 'min', label: '最小副本' },
  { key: 'max', label: '最大副本' },
  { key: 'current', label: '当前副本' },
  { key: 'desired', label: '期望副本' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const template = `apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: my-hpa
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-app
  minReplicas: 1
  maxReplicas: 5
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
`
</script>
