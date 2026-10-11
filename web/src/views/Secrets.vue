<template>
  <ResourceListView
    kind="secret" endpoint="/secrets" :columns="columns" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 200, sortable: true, primary: true },
  { prop: 'type', label: '类型', width: 210 },
  { prop: 'keys', label: '数据键', type: 'tags', minWidth: 240 },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const template = `apiVersion: v1
kind: Secret
metadata:
  name: my-secret
  namespace: default
type: Opaque
stringData:
  username: admin
  password: changeme
`
</script>
