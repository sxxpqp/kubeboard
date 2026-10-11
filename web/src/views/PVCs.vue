<template>
  <ResourceListView
    detail-enabled
    kind="pvc" endpoint="/pvcs" :columns="columns" :detail-fields="detailFields" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 180, sortable: true, primary: true },
  { prop: 'status', label: '状态', type: 'status', width: 110 },
  { prop: 'capacity', label: '容量', width: 100 },
  { prop: 'accessModes', label: '访问模式', type: 'tags', minWidth: 200 },
  { prop: 'storageClass', label: '存储类', width: 140 },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'status', label: '状态', type: 'status' },
  { key: 'capacity', label: '容量' },
  { key: 'accessModes', label: '访问模式', type: 'tags' },
  { key: 'storageClass', label: '存储类' },
  { key: 'volumeName', label: '绑定 PV' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const template = `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
  namespace: default
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: standard
`
</script>
