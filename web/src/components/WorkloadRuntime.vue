<template>
  <div v-loading="loading">
    <div class="runtime-head"><strong>运行时</strong><span class="muted">历史版本 → Pod → 生命周期</span><el-button :loading="loading" @click="refresh">刷新</el-button></div>
    <el-alert v-if="error" :title="error" type="error" :closable="false" />
    <div class="runtime-scroll"><div class="runtime-columns">
      <section class="runtime-column"><h3>历史版本</h3><el-button v-if="revisions.length>1" link type="primary" @click="openCompare">对比两个版本</el-button><p v-if="kind === 'daemonset'" class="muted">当前接口未提供 DaemonSet 版本记录</p><el-empty v-else-if="!loading && !revisions.length" description="暂无版本记录" :image-size="50" />
        <div v-for="revision in revisions" :key="revision.name" class="revision-card" :class="{ selected: selectedRevision === revision.name }">
          <button class="object-link" @click="selectedRevision = revision.name">rev {{ revision.revision }} <span v-if="revision.current">· 当前</span></button>
          <p>{{ revision.name }}</p><p v-for="image in revision.images" :key="image" class="muted">{{ image }}</p><p v-if="kind==='deployment'" class="muted">期望 {{ revision.replicas }} / 就绪 {{ revision.ready }}</p><p class="muted">{{revision.createdTime>0?new Date(revision.createdTime*1000).toLocaleString():'创建时间未记录'}}</p>
          <el-button v-if="!revision.current" link type="primary" :loading="rollingBack" @click="rollback(revision)">回退到此版本</el-button>
        </div>
      </section>
      <section class="runtime-column"><h3>版本对应的 Pod</h3><el-empty v-if="!loading && !visiblePods.length" description="此版本暂无存活 Pod" :image-size="50" />
        <button v-for="pod in visiblePods" :key="pod.metadata.name" class="pod-card" :class="{ selected: selectedPod === pod.metadata.name }" @click="selectedPod = pod.metadata.name">
          <strong>{{ pod.metadata.name }}</strong><p>{{ pod.status?.phase || '未知' }}</p><p class="muted">{{ pod.spec?.nodeName || '尚未调度' }}</p><p class="muted">{{ pod.status?.podIP || '尚无 IP' }}</p>
        </button>
      </section>
      <section class="runtime-column lifecycle-column"><h3>Pod 生命周期 · 最新状态在上</h3>
        <template v-if="selectedPod"><div class="runtime-pod-head"><strong>{{ selectedPod }}</strong><el-button @click="consoleVisible = true">日志 / 终端</el-button><router-link :to="`/detail/pod/${namespace}/${selectedPod}`">Pod 详情 →</router-link></div><PodLifecycle :key="selectedPod + refreshKey" :namespace="namespace" :name="selectedPod" /></template>
        <el-empty v-else description="选择一个 Pod 查看生命周期" :image-size="60" />
      </section>
    </div></div>
    <el-dialog v-model="compareVisible" title="版本配置对比" width="min(1150px,94vw)" destroy-on-close>
      <div class="comparison-selectors"><label>原版本 <el-select v-model="compareFrom"><el-option v-for="r in revisions" :key="r.name" :label="'rev '+r.revision" :value="r.name" /></el-select></label><span>→</span><label>目标版本 <el-select v-model="compareTo"><el-option v-for="r in revisions" :key="r.name" :label="'rev '+r.revision" :value="r.name" /></el-select></label></div>
      <el-alert v-if="!comparisonReady" title="缺少版本 Pod 模板，无法对比，请确认后端已更新并有读取权限" type="warning" :closable="false" />
      <template v-else><p class="muted">对比 Pod 模板配置，忽略控制器生成的 pod-template-hash；数组按顺序比较。共 {{differences.length}} 项变化。</p><el-table :data="differences" empty-text="这两个版本的 Pod 模板配置相同" max-height="500"><el-table-column prop="path" label="配置字段" min-width="260" /><el-table-column prop="type" label="变化" width="80" /><el-table-column label="原版本" min-width="240"><template #default="{row}"><pre class="diff-value">{{formatValue(row.before)}}</pre></template></el-table-column><el-table-column label="目标版本" min-width="240"><template #default="{row}"><pre class="diff-value">{{formatValue(row.after)}}</pre></template></el-table-column></el-table></template>
    </el-dialog>
    <WorkloadConsole v-model="consoleVisible" :kind="kind" :namespace="namespace" :name="name" :initial-pod="selectedPod" />
  </div>
</template>
<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { load, dump } from 'js-yaml'
import { compareRevisions, revisionTemplate } from '../utils/revisionComparison'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'
import PodLifecycle from './PodLifecycle.vue'
import WorkloadConsole from './WorkloadConsole.vue'
const props=defineProps({kind:String,namespace:String,name:String})
const revisions=ref([]),pods=ref([]),selectedRevision=ref(''),selectedPod=ref(''),loading=ref(false),error=ref(''),rollingBack=ref(false),consoleVisible=ref(false),refreshKey=ref(0)
const compareVisible=ref(false),compareFrom=ref(''),compareTo=ref('')
const beforeTemplate=computed(()=>revisionTemplate(revisions.value.find(r=>r.name===compareFrom.value)?.template)),afterTemplate=computed(()=>revisionTemplate(revisions.value.find(r=>r.name===compareTo.value)?.template))
const comparisonReady=computed(()=>!!beforeTemplate.value&&!!afterTemplate.value)
const differences=computed(()=>comparisonReady.value?compareRevisions(beforeTemplate.value,afterTemplate.value):[])
function openCompare(){compareTo.value=selectedRevision.value||revisions.value[0]?.name;compareFrom.value=revisions.value.find(r=>r.name!==compareTo.value)?.name;compareVisible.value=true}
function formatValue(value){return value===undefined?'未配置':typeof value==='object'?dump(value,{lineWidth:-1}):String(value)}
const visiblePods=computed(()=>!selectedRevision.value?pods.value:pods.value.filter(p=>props.kind==='deployment'?(p.metadata.ownerReferences||[]).some(o=>o.kind==='ReplicaSet'&&o.name===selectedRevision.value):p.metadata.labels?.['controller-revision-hash']===selectedRevision.value))
watch(visiblePods,list=>{if(!list.some(p=>p.metadata.name===selectedPod.value))selectedPod.value=list[0]?.metadata.name || ''})
let sequence=0
async function refresh(){const id=++sequence;loading.value=true;error.value='';const params={kind:props.kind,namespace:props.namespace,name:props.name};try{
 const [history,instances]=await Promise.all([props.kind==='daemonset'?Promise.resolve({items:[]}):http.get('/workloads/revisions',{params}),http.get('/workloads/pods',{params})]);
 const objects=await Promise.all((instances.items||[]).map(async pod=>load(await http.get('/resource',{params:{kind:'pod',namespace:props.namespace,name:pod.name},responseType:'text'}))));
 if(id!==sequence)return;revisions.value=history.items||[];pods.value=objects;
 if(!revisions.value.some(r=>r.name===selectedRevision.value))selectedRevision.value=revisions.value.find(r=>r.current)?.name||revisions.value[0]?.name||'';refreshKey.value++;
 }catch(e){if(id===sequence)error.value='运行时加载失败：'+(e.response?.data?.error||e.message)}finally{if(id===sequence)loading.value=false}}
async function rollback(revision){try{await ElMessageBox.confirm(`${props.namespace} / ${props.name}：使用 rev ${revision.revision} 的配置重新发布，将创建新修订并替换 Pod。`,'回退确认',{type:'warning'})}catch{return}rollingBack.value=true;try{await http.post('/workloads/rollback',{kind:props.kind,namespace:props.namespace,name:props.name,revision:revision.revision});ElMessage.success('回退请求已提交，请刷新查看实际就绪状态');await refresh()}catch(e){ElMessage.error(e.response?.data?.error||e.message)}finally{rollingBack.value=false}}
onMounted(refresh)
defineExpose({refresh})
</script>
<style scoped>
.comparison-selectors{display:flex;align-items:center;gap:20px;margin-bottom:20px}.comparison-selectors label{display:flex;gap:10px;align-items:center}.diff-value{white-space:pre-wrap;overflow-wrap:anywhere;font-size:12px}.runtime-head {display:flex;gap:16px;align-items:center;margin-bottom:20px}.runtime-head .el-button{margin-left:auto}.runtime-scroll{overflow-x:auto}.runtime-columns{display:grid;grid-template-columns:210px 220px minmax(340px,1fr);gap:20px;min-width:810px}.runtime-column{min-width:0}h3{font-size:14px;margin:0 0 18px}.revision-card,.pod-card{padding:16px;margin-bottom:12px;border:1px solid var(--ks-border);border-radius:6px;background:var(--ks-body-bg);overflow-wrap:anywhere}.selected{border-color:var(--ks-primary);background:var(--ks-primary-light)}.object-link{border:0;background:transparent;color:var(--ks-primary);text-align:left;font:inherit;font-weight:600;cursor:pointer}.pod-card{width:100%;text-align:left;cursor:pointer;color:var(--ks-text-primary);font:inherit}.muted{font-size:12px;color:var(--ks-text-secondary)}p{margin:6px 0}.lifecycle-column{border-left:1px solid var(--ks-border);padding-left:20px}.runtime-pod-head{display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin-bottom:20px}.runtime-pod-head strong{width:100%;overflow-wrap:anywhere}
</style>
