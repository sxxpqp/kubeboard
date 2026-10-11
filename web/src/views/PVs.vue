<template>
  <ResourceListView
    kind="pv" endpoint="/pvs" :namespace-scoped="false" :columns="columns" :template="template"
    :ops-width="90"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'name', label: '名称', minWidth: 220, sortable: true, primary: true },
  { prop: 'status', label: '状态', type: 'status', width: 110 },
  { prop: 'capacity', label: '容量', width: 100 },
  { prop: 'accessModes', label: '访问模式', type: 'tags', minWidth: 180 },
  { prop: 'storageClass', label: '存储类', width: 140 },
  { prop: 'claim', label: '绑定的 PVC', width: 200 },
  { prop: 'reclaimPolicy', label: '回收策略', width: 110 },
]

const template = `apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  capacity:
    storage: 5Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /mnt/data
`
</script>
