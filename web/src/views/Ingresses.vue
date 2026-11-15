<template>
  <ResourceListView kind="ingress" endpoint="/ingresses" :columns="columns" :template="template">
    <template #col-rules="{row}"><div v-for="(rule,index) in row.rules" :key="index" class="rule-line">{{rule.path}} → {{rule.service}}:{{rule.port}}</div></template>
  </ResourceListView>
</template>
<script setup>
import ResourceListView from '../components/ResourceListView.vue'
const columns=[
 {prop:'name',label:'名称',primary:true,minWidth:200,sortable:true},
 {prop:'namespace',label:'项目',width:150},
 {prop:'hosts',label:'域名',type:'tags',minWidth:200},
 {prop:'rules',label:'路径 → 后端服务',minWidth:260},
 {prop:'className',label:'Ingress Class',width:160},
 {prop:'tls',label:'TLS',type:'bool',width:90}
]
const template = `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: default
spec:
  ingressClassName: nginx
  rules:
    - host: app.example.com
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-service
                port:
                  number: 80
`

</script>
<style scoped>.rule-line{font-size:12px;line-height:1.8;overflow-wrap:anywhere}</style>
