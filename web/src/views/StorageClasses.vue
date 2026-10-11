<template>
  <ResourceListView
    kind="storageclass" endpoint="/storageclasses" :namespace-scoped="false"
    :columns="columns" :template="template" :ops-width="90"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'name', label: '名称', minWidth: 200, sortable: true, primary: true },
  { prop: 'provisioner', label: '制备器', minWidth: 240 },
  { prop: 'reclaimPolicy', label: '回收策略', width: 120 },
  { prop: 'volumeBinding', label: '绑定模式', width: 120 },
  { prop: 'isDefault', label: '默认', type: 'bool', width: 80, align: 'center' },
]

const template = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: my-sc
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
`
</script>
