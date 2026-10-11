<template>
  <ResourceListView
    kind="cronjob" endpoint="/workloads" :endpoint-params="{ kind: 'cronjob' }"
    :columns="columns" :template="template"
  />
</template>

<script setup>
import ResourceListView from '../components/ResourceListView.vue'

const columns = [
  { prop: 'namespace', label: '项目', width: 130, sortable: true },
  { prop: 'name', label: '名称', minWidth: 180, sortable: true, primary: true },
  { prop: 'schedule', label: '调度规则', width: 160 },
  { prop: 'active', label: '活跃任务', width: 100, align: 'center' },
  { prop: 'suspend', label: '已挂起', type: 'bool', width: 90, align: 'center' },
  { prop: 'images', label: '镜像', type: 'list', minWidth: 220 },
]

const template = `apiVersion: batch/v1
kind: CronJob
metadata:
  name: my-cronjob
  namespace: default
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: task
              image: busybox:latest
              command: ["sh", "-c", "date; echo hello"]
`
</script>
