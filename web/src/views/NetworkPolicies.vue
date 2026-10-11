<template>
  <ResourceListView
    kind="networkpolicy" endpoint="/networkpolicies" :namespace-scoped="true"
    :columns="columns" :creatable="true" :deletable="true" :editable="false"
    :template="template" detail-enabled :detail-fields="detailFields"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 150, sortable: true },
  { prop: 'name', label: '名称', minWidth: 180, sortable: true, primary: true },
  { prop: 'policyTypes', label: '策略类型', type: 'tags', minWidth: 160 },
  { prop: 'podSelector', label: 'Pod 选择器', minWidth: 240 },
  { prop: 'ingress', label: '入站规则', width: 90, align: 'center' },
  { prop: 'egress', label: '出站规则', width: 90, align: 'center' },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const detailFields = [
  { key: 'namespace', label: '项目' },
  { key: 'name', label: '名称' },
  { key: 'policyTypes', label: '策略类型', type: 'tags' },
  { key: 'podSelector', label: 'Pod 选择器' },
  { key: 'ingress', label: '入站规则数' },
  { key: 'egress', label: '出站规则数' },
  { key: 'creationTime', label: '创建时间', type: 'time' },
]

const template = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: my-netpol
  namespace: default
spec:
  podSelector:
    matchLabels:
      app: my-app
  policyTypes:
    - Ingress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: allowed
`
</script>
