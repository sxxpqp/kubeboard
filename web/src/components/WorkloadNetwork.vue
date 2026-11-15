<template>
  <div v-loading="loading"><el-alert v-if="error" :title="error" type="error" :closable="false" /><el-table v-if="type==='service'" :data="services" class="ks-table" empty-text="未发现选择器匹配此工作负载的服务"><el-table-column label="服务" min-width="180"><template #default="{row}"><router-link :to="`/detail/service/${namespace}/${row.metadata.name}`">{{row.metadata.name}}</router-link></template></el-table-column><el-table-column prop="spec.type" label="类型" /><el-table-column prop="spec.clusterIP" label="集群 IP" /><el-table-column label="端口" min-width="220"><template #default="{row}"><div v-for="port in row.spec.ports" :key="port.port">{{port.port}} → {{port.targetPort}} / {{port.protocol}}</div></template></el-table-column></el-table><el-table v-else :data="routes" class="ks-table" empty-text="未发现指向关联服务的应用路由"><el-table-column label="应用路由" min-width="180"><template #default="{row}"><router-link :to="`/detail/ingress/${namespace}/${row.name}`">{{row.name}}</router-link></template></el-table-column><el-table-column prop="host" label="主机" /><el-table-column prop="path" label="路径" /><el-table-column prop="service" label="后端服务" /><el-table-column prop="port" label="端口" /></el-table></div>
</template>
<script setup>
import {computed,onMounted,ref} from 'vue'
import {load} from 'js-yaml'
import http from '../api/http'
const props=defineProps({kind:String,namespace:String,name:String,type:String})
const services=ref([]),ingresses=ref([]),error=ref(''),loading=ref(false)
const routes=computed(()=>ingresses.value.flatMap(ing=>(ing.rules||[]).filter(rule=>services.value.some(s=>s.metadata.name===rule.service)).map(rule=>({...rule,name:ing.name}))))
onMounted(async()=>{
 loading.value=true
 try{
  const data=await http.get('/related',{params:{kind:props.kind,namespace:props.namespace,name:props.name}})
  services.value=await Promise.all((data.items||[]).filter(item=>item.kind==='service').map(async item=>load(await http.get('/resource',{params:{kind:'service',namespace:props.namespace,name:item.name},responseType:'text'}))))
  if(props.type==='ingress'){const data=await http.get('/ingresses',{params:{namespace:props.namespace}});ingresses.value=data.items||[]}
 }catch(e){error.value=e.response?.data?.error||e.message}finally{loading.value=false}
})
</script>
