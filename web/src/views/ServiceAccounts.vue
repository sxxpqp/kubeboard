<template>
  <ResourceListView
    detail-enabled
    kind="serviceaccount" endpoint="/serviceaccounts" :columns="columns" :detail-fields="detailFields" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 200, sortable: true, primary: true },
  { prop: 'secrets', label: '关联密钥', type: 'tags', minWidth: 300 },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'secrets', label: '关联密钥', type: 'tags' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const template = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-account
  namespace: default
`
</script>
