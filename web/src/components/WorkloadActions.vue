<template>
  <div v-if="!compact" class="workload-actions" aria-label="工作负载操作栏">
    <el-button @click="edit('form')">编辑</el-button>
    <el-button @click="edit('yaml')">预览 / 编辑 YAML</el-button>
    <el-button @click="openImage">调整镜像</el-button>
    <el-button v-if="kind !== 'daemonset'" @click="scale">伸缩</el-button>
    <el-button @click="consoleVisible = true">日志 / 终端</el-button>
    <el-button @click="copy">复制</el-button>
    <el-button type="warning" plain :loading="busy" @click="restart">重启</el-button>
    <el-button type="danger" plain :loading="busy" @click="remove">删除</el-button>
  </div>
  <el-dropdown v-else trigger="click" @command="runCommand"><el-button @click.stop>更多</el-button><template #dropdown><el-dropdown-menu><el-dropdown-item command="edit">编辑</el-dropdown-item><el-dropdown-item command="yaml">预览 / 编辑 YAML</el-dropdown-item><el-dropdown-item command="image">调整镜像</el-dropdown-item><el-dropdown-item v-if="kind !== 'daemonset'" command="scale">伸缩</el-dropdown-item><el-dropdown-item command="console">日志 / 终端</el-dropdown-item><el-dropdown-item command="copy">复制</el-dropdown-item><el-dropdown-item command="restart">重启</el-dropdown-item><el-dropdown-item divided command="delete">删除</el-dropdown-item></el-dropdown-menu></template></el-dropdown>
  <el-dialog v-model="imageVisible" title="调整容器镜像" width="min(560px, 90vw)">
    <p>{{ namespace }} / {{ name }}</p>
    <el-form label-position="top"><el-form-item label="目标容器"><el-select v-model="imageContainer"><el-option v-for="c in imageContainers" :key="c.name" :label="c.name" :value="c.name" /></el-select></el-form-item><el-form-item label="目标镜像"><el-input v-model="image" placeholder="registry/app:version" /></el-form-item></el-form>
    <p class="muted">仅修改选中容器；提交后触发工作负载更新。</p>
    <template #footer><el-button @click="imageVisible = false">取消</el-button><el-button type="primary" :loading="busy" @click="saveImage">更新镜像</el-button></template>
  </el-dialog>
  <WorkloadConsole v-model="consoleVisible" :kind="kind" :namespace="namespace" :name="name" />
</template>
<script setup>
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { load, dump } from 'js-yaml'
import http from '../api/http'
import WorkloadConsole from './WorkloadConsole.vue'
import { changeContainerImage, copyWorkload } from '../utils/workloadActions'
const props = defineProps({ compact: Boolean, kind: String, namespace: String, name: String })
const emit = defineEmits(['changed'])
const router = useRouter()
const busy = ref(false)
const consoleVisible = ref(false), imageVisible = ref(false), imageContainers = ref([]), imageContainer = ref(''), image = ref('')
const params = () => ({ kind: props.kind, namespace: props.namespace, name: props.name })
const source = async () => load(await http.get('/resource', { params: params(), responseType: 'text' }))
const error = e => ElMessage.error(e.response?.data?.error || e.message || '操作失败')
function edit(view) { router.push({path:`/workloads/${props.kind}/${encodeURIComponent(props.namespace)}/${encodeURIComponent(props.name)}/edit`,query:{mode:view}}) }
function runCommand(command) { const actions={edit:()=>edit('form'),yaml:()=>edit('yaml'),image:openImage,scale,console:()=>{consoleVisible.value=true},copy,restart,delete:remove};actions[command]?.() }
async function openImage() {
  try { const object = await source(); imageContainers.value = [...object.spec.template.spec.containers,...(object.spec.template.spec.initContainers||[])]; imageContainer.value = imageContainers.value[0]?.name || ''; image.value = imageContainers.value[0]?.image || ''; imageVisible.value = true } catch (e) { error(e) }
}
watch(imageContainer, name => { image.value = imageContainers.value.find(c => c.name === name)?.image || '' })
async function saveImage() {
  if(busy.value)return
  busy.value = true
  try { const object = changeContainerImage(await source(), imageContainer.value, image.value); await http.put('/resources', { kind: props.kind, yaml: dump(object) }); imageVisible.value = false; ElMessage.success('镜像配置已更新'); emit('changed') } catch(e) { error(e) } finally { busy.value = false }
}
async function scale() {
  let value
  try { ({ value } = await ElMessageBox.prompt(`目标：${props.namespace} / ${props.name}`, '调整副本数', { inputPattern: /^\d+$/, inputErrorMessage: '请输入非负整数' })) } catch { return }
  try { await http.post('/workloads/scale', { ...params(), replicas: Number(value) }); ElMessage.success('期望副本数已更新'); emit('changed') } catch(e) { error(e) }
}
async function copy() {
  let value
  try { ({ value } = await ElMessageBox.prompt('复制到当前命名空间，确认清单后再创建。PVC 和服务引用会保留，请检查是否需要调整。', '复制工作负载', { inputValue: props.name + '-copy' })) } catch { return }
  try { const draft=dump(copyWorkload(await source(), value));router.push({path:`/projects/${encodeURIComponent(props.namespace)}/workloads/create`,query:{mode:'yaml',kind:props.kind},state:{workloadDraft:draft}}) } catch(e) { error(e) }
}
async function confirmRun(action, run) {
  if(busy.value)return
  try { await ElMessageBox.confirm(`${props.namespace} / ${props.name}：${action === '删除' ? '删除工作负载及受控副本，可能中断服务。独立 PVC 不会由此删除。' : '触发滚动重启并替换 Pod。'}`, `${action}确认`, { type: 'warning' }) } catch { return }
  busy.value = true
  try { await run(); ElMessage.success(`已提交${action}请求`); emit('changed') } catch(e) { error(e) } finally { busy.value = false }
}
function restart() { return confirmRun('重启', () => http.post('/workloads/restart', params())) }
function remove() { return confirmRun('删除', async () => { await http.delete('/resources', { params: params() }); await router.push('/projects/' + encodeURIComponent(props.namespace) + '/workloads') }) }
</script>
<style scoped>
.workload-actions { display:flex; flex-wrap:wrap; gap:8px; }
.workload-actions .el-button { margin:0; }
.muted { color:var(--ks-text-secondary); font-size:12px; }
</style>
