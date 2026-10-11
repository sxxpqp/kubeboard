<template>
  <ResourceListView
    kind="job" endpoint="/workloads" :endpoint-params="{ kind: 'job' }"
    :columns="columns" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 180, sortable: true, primary: true },
  { prop: 'ready', label: '完成', width: 90, align: 'center' },
  { prop: 'images', label: '镜像', type: 'list', minWidth: 240 },
  { prop: 'creationTime', label: '创建时间', type: 'time', width: 150, sortable: true },
]

const template = `apiVersion: batch/v1
kind: Job
metadata:
  name: my-job
  namespace: default
spec:
  completions: 1
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: job
          image: busybox:latest
          command: ["sh", "-c", "echo hello && sleep 10"]
`
</script>
