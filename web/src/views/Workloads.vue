<template>
  <section class="ks-card workload-list">
    <div class="list-title"><div><h2>工作负载</h2><p class="muted">查看版本、运行状态及容器实例</p></div><el-button type="primary" @click="create('form')">创建工作负载</el-button><el-button @click="create('yaml')">YAML 创建</el-button></div>
    <el-tabs v-model="kind" @tab-change="load"><el-tab-pane label="部署 Deployment" name="deployment" /><el-tab-pane label="有状态副本集 StatefulSet" name="statefulset" /><el-tab-pane label="守护进程集 DaemonSet" name="daemonset" /></el-tabs>
    <el-alert v-if="error" :title="error" type="error" :closable="false"><el-button link @click="load">重试</el-button></el-alert>
    <div class="ks-toolbar"><el-input v-model="keyword" placeholder="搜索名称" clearable style="max-width:300px" :prefix-icon="'Search'" /><el-select v-model="status" placeholder="全部状态" clearable style="width:160px"><el-option label="运行中" value="running" /><el-option label="更新中" value="updating" /><el-option label="已停止" value="stopped" /></el-select><el-button :loading="loading" @click="load">刷新</el-button></div>
    <el-table :data="paged" v-loading="loading" class="ks-table" :empty-text="keyword || status ? '没有匹配结果，请清除筛选' : '暂无工作负载，可创建或导入 YAML'">
      <el-table-column label="名称 / 镜像" min-width="260"><template #default="{row}"><router-link class="resource-name" :to="detailPath(row)">{{row.name}}</router-link><div v-for="image in row.images" :key="image" class="muted image-line">{{image}}</div></template></el-table-column>
      <el-table-column v-if="!namespace" prop="namespace" label="项目" width="150" sortable />
      <el-table-column label="状态" width="120"><template #default="{row}"><el-tag :type="row.state==='running'?'success':row.state==='updating'?'warning':'info'">{{stateLabel(row.state)}}</el-tag></template></el-table-column>
      <el-table-column :label="kind==='daemonset'?'就绪 / 期望节点':'就绪 / 期望副本'" width="160"><template #default="{row}">{{row.ready}}</template></el-table-column>
      <el-table-column label="更新时间" width="175"><template #default="{row}">{{formatTime(row.updatedAt || row.creationTime)}}</template></el-table-column>
      <el-table-column label="操作" width="160" fixed="right"><template #default="{row}"><div class="row-actions"><router-link :to="detailPath(row)">详情</router-link><WorkloadActions compact :kind="kind" :namespace="row.namespace" :name="row.name" @changed="load" /></div></template></el-table-column>
    </el-table>
    <div class="table-footer"><span>共 {{filtered.length}} 项</span><el-pagination v-model:current-page="page" :page-size="20" :total="filtered.length" layout="prev,pager,next" /></div>
  </section>
</template>
<script setup>
import {computed,onMounted,ref,watch} from 'vue'
import {useRoute,useRouter} from 'vue-router'
import {ElMessageBox} from 'element-plus'
import http from '../api/http'
import WorkloadActions from '../components/WorkloadActions.vue'
const route=useRoute(),router=useRouter(),namespace=String(route.params.namespace||''),kind=ref(['deployment','statefulset','daemonset'].includes(route.query.kind)?route.query.kind:'deployment'),keyword=ref(String(route.query.search||'')),status=ref(String(route.query.status||'')),page=ref(1),items=ref([]),loading=ref(false),error=ref('')
const filtered=computed(()=>items.value.filter(row=>(!keyword.value||row.name.toLowerCase().includes(keyword.value.toLowerCase()))&&(!status.value||row.state===status.value)))
const paged=computed(()=>filtered.value.slice((page.value-1)*20,page.value*20))
watch([keyword,status,kind],()=>{page.value=1;router.replace({query:{...route.query,kind:kind.value,search:keyword.value||undefined,status:status.value||undefined}})})
let sequence=0
async function load(){const id=++sequence;loading.value=true;error.value='';try{const data=await http.get('/workloads',{params:{kind:kind.value,namespace:namespace||undefined}});if(id===sequence)items.value=data.items||[]}catch(e){if(id===sequence)error.value=e.response?.data?.error||e.message}finally{if(id===sequence)loading.value=false}}
function detailPath(row){return `/workloads/${kind.value}/${encodeURIComponent(row.namespace)}/${encodeURIComponent(row.name)}`}
function stateLabel(value){return {running:'运行中',updating:'更新中',stopped:'已停止'}[value]||value||'未知'}
function formatTime(ts){return ts?new Date(ts*1000).toLocaleString('zh-CN',{hour12:false}):'—'}
async function create(mode){let ns=namespace;if(!ns){try{const result=await ElMessageBox.prompt('请选择目标命名空间名称，也可先在顶栏切换范围','创建范围',{inputValue:'default',inputPattern:/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/,inputErrorMessage:'请输入有效命名空间'});ns=result.value}catch{return}}router.push({path:`/projects/${encodeURIComponent(ns)}/workloads/create`,query:{mode,kind:kind.value}})}
onMounted(load)
</script>
<style scoped>
.list-title{display:flex;align-items:center;gap:12px;margin-bottom:24px;flex-wrap:wrap}.list-title>div{margin-right:auto}h2{font-size:18px;margin:0 0 6px}.muted{font-size:12px;color:var(--ks-text-secondary);line-height:1.8;margin:0}.image-line{overflow-wrap:anywhere}.row-actions{display:flex;align-items:center;gap:14px}.table-footer{display:flex;justify-content:space-between;align-items:center}
</style>
