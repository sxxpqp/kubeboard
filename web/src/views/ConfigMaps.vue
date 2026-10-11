<template>
  <ResourceListView
    detail-enabled
    kind="configmap" endpoint="/configmaps" :columns="columns" :detail-fields="detailFields" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 220, sortable: true, primary: true },
  { prop: 'keys', label: '数据键', type: 'tags', minWidth: 300 },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'keys', label: '数据键', type: 'tags' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const template = `apiVersion: v1
kind: ConfigMap
metadata:
  name: my-config
  namespace: default
data:
  key1: value1
  app.properties: |
    server.port=8080
`
</script>
