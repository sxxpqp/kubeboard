<template>
  <section class="ks-card">
    <el-alert v-if="error" :title="error" type="error" :closable="false"><el-button link @click="load">重试</el-button></el-alert>
    <div class="ks-toolbar"><el-input v-model="keyword" placeholder="搜索对象、原因或消息" clearable style="max-width:360px" /><el-select v-model="type" placeholder="全部级别" clearable style="width:160px"><el-option label="Normal" value="Normal" /><el-option label="Warning" value="Warning" /></el-select><el-button :loading="loading" @click="load">刷新</el-button></div>
    <el-table :data="paged" v-loading="loading" class="ks-table" :empty-text="keyword||type?'没有匹配结果，请清除筛选':'暂无事件（集群仅保留一定时间的事件）'">
      <el-table-column label="资源对象" min-width="240"><template #default="{row}"><router-link v-if="objectPath(row)" :to="objectPath(row)">{{row.object}}</router-link><span v-else>{{row.object}}</span></template></el-table-column>
      <el-table-column v-if="!namespace" prop="namespace" label="项目" width="150" />
      <el-table-column label="级别" width="110"><template #default="{row}"><el-tag :type="row.type==='Warning'?'warning':'info'">{{row.type}}</el-tag></template></el-table-column>
      <el-table-column prop="reason" label="原因" width="180" /><el-table-column prop="message" label="消息" min-width="350" /><el-table-column prop="count" label="次数" width="80" /><el-table-column label="最近发生" width="180"><template #default="{row}">{{row.lastSeen>0?new Date(row.lastSeen*1000).toLocaleString('zh-CN',{hour12:false}):'时间未记录'}}</template></el-table-column>
    </el-table><div class="table-footer"><span>已加载 {{items.length}} 条，匹配 {{filtered.length}} 条</span><el-pagination v-model:current-page="page" :page-size="20" :total="filtered.length" layout="prev,pager,next" /></div>
  </section>
</template>
<script setup>
import {computed,onMounted,ref,watch} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import http from '../api/http'
const route=useRoute(),router=useRouter(),namespace=String(route.params.namespace||''),keyword=ref(String(route.query.search||'')),type=ref(String(route.query.type||'')),page=ref(1),items=ref([]),loading=ref(false),error=ref('')
const filtered=computed(()=>items.value.filter(item=>(!type.value||item.type===type.value)&&(!keyword.value||[item.object,item.reason,item.message].some(value=>String(value||'').toLowerCase().includes(keyword.value.toLowerCase())))).sort((a,b)=>b.lastSeen-a.lastSeen))
const paged=computed(()=>filtered.value.slice((page.value-1)*20,page.value*20))
watch([keyword,type],()=>{page.value=1;router.replace({query:{...route.query,search:keyword.value||undefined,type:type.value||undefined}})})
const knownKinds=new Set(['pod','deployment','statefulset','daemonset','job','cronjob','replicaset','service','ingress','node','namespace','pvc','pv','storageclass','configmap','secret','serviceaccount','hpa','networkpolicy'])
function objectPath(row){const aliases={persistentvolumeclaim:'pvc',persistentvolume:'pv',horizontalpodautoscaler:'hpa'},kind=aliases[String(row.kind).toLowerCase()]||String(row.kind).toLowerCase();if(!knownKinds.has(kind)||!row.name)return '';const ns=['node','namespace','pv','storageclass'].includes(kind)?'_':row.namespace||'_';return `${['deployment','statefulset','daemonset'].includes(kind)?'/workloads':'/detail'}/${kind}/${encodeURIComponent(ns)}/${encodeURIComponent(row.name)}`}
async function load(){loading.value=true;error.value='';try{const data=await http.get('/events',{params:{namespace:namespace||undefined}});items.value=data.items||[];page.value=Math.min(page.value,Math.max(1,Math.ceil(filtered.value.length/20)))}catch(e){error.value=e.response?.data?.error||e.message}finally{loading.value=false}}
onMounted(load)
</script>
<style scoped>.table-footer{display:flex;justify-content:space-between;align-items:center;padding-top:16px;font-size:12px;color:var(--ks-text-secondary)}</style>
