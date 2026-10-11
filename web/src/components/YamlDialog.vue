<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'create' ? '创建资源' : `编辑: ${name}`"
    width="960px"
    top="4vh"
    :close-on-click-modal="false"
    @closed="$emit('update:modelValue', false)"
    class="yaml-dialog"
  >
    <!-- 模式切换 -->
    <div class="mode-bar">
      <el-radio-group v-model="view" size="small">
        <el-radio-button value="form" v-if="formAvailable">表单</el-radio-button>
        <el-radio-button value="yaml">YAML</el-radio-button>
      </el-radio-group>
      <span class="mode-hint">
        <template v-if="view === 'form'">可视化编辑字段，切换 YAML 可查看生成结果</template>
        <template v-else>直接编辑 YAML 原文{{ formAvailable ? '，含未在表单中暴露的字段' : '' }}</template>
      </span>
    </div>

    <!-- ============ 表单模式：简单资源（Service/ConfigMap/Secret）============ -->
    <div v-if="view === 'form' && isSimpleKind" class="form-wrap">
      <el-form label-width="130px" label-position="right" size="default" class="simple-form-scroll">
        <SimpleResourceForm :kind="kind" v-model="simpleForm" :namespaces="namespaces" />
      </el-form>
      <div class="yaml-preview-wrap">
        <div class="preview-head">
          <span>YAML 预览</span>
          <el-button link type="primary" size="small" @click="view = 'yaml'">在 YAML 中编辑</el-button>
        </div>
        <pre class="yaml-preview">{{ simpleFormYaml }}</pre>
      </div>
    </div>

    <!-- ============ 表单模式（工作负载）============ -->
    <div v-else-if="view === 'form'" class="form-wrap">
      <el-form :model="form" label-width="130px" label-position="right" size="default">
        <el-collapse v-model="openSections">
          <el-collapse-item name="basic">
            <template #title><span class="sec-title">基础信息</span></template>
            <el-form-item label="项目">
              <el-select v-model="form.namespace" filterable style="width: 260px">
                <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
              </el-select>
            </el-form-item>
            <el-form-item label="名称"><el-input v-model="form.name" style="width: 260px" /></el-form-item>
            <el-form-item label="描述">
              <el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" />
            </el-form-item>
            <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
            <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
          </el-collapse-item>

          <el-collapse-item name="containers">
            <template #title><span class="sec-title">容器配置（{{ form.containers.length }}）</span></template>
            <div v-for="(ctr, idx) in form.containers" :key="idx" class="ctr-block">
              <div class="ctr-head">
                <span class="ctr-no">容器 {{ idx + 1 }}</span>
                <el-button v-if="form.containers.length > 1" link type="danger" size="small" @click="form.containers.splice(idx, 1)">移除</el-button>
              </div>
              <el-form-item label="容器名"><el-input v-model="ctr.name" style="width: 220px" /></el-form-item>
              <el-form-item label="镜像" required><el-input v-model="ctr.image" style="width: 380px" /></el-form-item>
              <el-form-item label="拉取策略">
                <el-select v-model="ctr.imagePullPolicy" style="width: 180px">
                  <el-option label="IfNotPresent" value="IfNotPresent" />
                  <el-option label="Always" value="Always" />
                  <el-option label="Never" value="Never" />
                </el-select>
              </el-form-item>
              <el-form-item label="启动命令"><el-input v-model="ctr.command" style="width: 460px" placeholder="多个用空格分隔" /></el-form-item>
              <el-form-item label="参数"><el-input v-model="ctr.args" style="width: 460px" placeholder="多个用空格分隔" /></el-form-item>
              <el-form-item label="端口">
                <div class="port-list">
                  <div v-for="(p, pi) in ctr.ports" :key="pi" class="port-row">
                    <el-input v-model="p.name" placeholder="名称" style="width: 120px" />
                    <el-input-number v-model="p.containerPort" :min="1" :max="65535" controls-position="right" style="width: 130px" />
                    <el-select v-model="p.protocol" style="width: 100px">
                      <el-option label="TCP" value="TCP" /><el-option label="UDP" value="UDP" />
                    </el-select>
                    <el-button link type="danger" size="small" @click="ctr.ports.splice(pi, 1)">删除</el-button>
                  </div>
                  <el-button link type="primary" size="small" @click="ctr.ports.push({ name: '', containerPort: 80, protocol: 'TCP' })">添加端口</el-button>
                </div>
              </el-form-item>
              <el-form-item label="环境变量"><KeyValueEditor v-model="ctr.env" key-placeholder="变量名" /></el-form-item>
              <el-form-item label="资源配额">
                <div class="res-grid">
                  <div class="res-col">
                    <div class="res-sub">requests</div>
                    <el-input v-model="ctr.cpuReq" placeholder="CPU" style="width: 150px" />
                    <el-input v-model="ctr.memReq" placeholder="内存" style="width: 150px; margin-top: 6px" />
                  </div>
                  <div class="res-col">
                    <div class="res-sub">limits</div>
                    <el-input v-model="ctr.cpuLim" placeholder="CPU" style="width: 150px" />
                    <el-input v-model="ctr.memLim" placeholder="内存" style="width: 150px; margin-top: 6px" />
                  </div>
                </div>
              </el-form-item>
              <el-form-item label="健康检查">
                <div v-for="(probe, pname) in ctr.probes" :key="pname" class="probe-block">
                  <el-checkbox v-model="probe.enabled">{{ probeLabel(pname) }}</el-checkbox>
                  <div v-if="probe.enabled" class="probe-body">
                    <el-select v-model="probe.type" style="width: 120px">
                      <el-option label="HTTP GET" value="httpGet" />
                      <el-option label="TCP" value="tcpSocket" />
                      <el-option label="命令" value="exec" />
                    </el-select>
                    <template v-if="probe.type === 'httpGet'">
                      <el-input v-model="probe.path" placeholder="/healthz" style="width: 150px" />
                      <el-input-number v-model="probe.port" :min="1" :max="65535" controls-position="right" style="width: 120px" />
                    </template>
                    <template v-else-if="probe.type === 'tcpSocket'">
                      <el-input-number v-model="probe.port" :min="1" :max="65535" controls-position="right" style="width: 120px" />
                    </template>
                    <template v-else>
                      <el-input v-model="probe.command" placeholder="命令" style="width: 240px" />
                    </template>
                  </div>
                </div>
              </el-form-item>
            </div>
            <el-button link type="primary" @click="form.containers.push(emptyCtr())">添加容器</el-button>
          </el-collapse-item>

          <el-collapse-item name="storage">
            <template #title><span class="sec-title">存储与挂载</span></template>
            <el-form-item label="挂载卷">
              <div class="vol-list">
                <div v-for="(v, vi) in form.volumes" :key="vi" class="vol-row">
                  <el-select v-model="v.type" style="width: 130px">
                    <el-option label="持久卷 PVC" value="pvc" />
                    <el-option label="配置字典" value="configMap" />
                    <el-option label="保密字典" value="secret" />
                    <el-option label="宿主机路径" value="hostPath" />
                    <el-option label="临时目录" value="emptyDir" />
                  </el-select>
                  <el-input v-model="v.name" placeholder="卷名" style="width: 130px" />
                  <el-input v-if="v.type !== 'emptyDir'" v-model="v.source" placeholder="来源" style="width: 180px" />
                  <el-input v-model="v.mountPath" placeholder="挂载路径" style="width: 150px" />
                  <el-button link type="danger" size="small" @click="form.volumes.splice(vi, 1)">删除</el-button>
                </div>
                <el-button link type="primary" size="small" @click="form.volumes.push(emptyVol())">添加挂载</el-button>
              </div>
            </el-form-item>
          </el-collapse-item>

          <el-collapse-item name="advanced">
            <template #title><span class="sec-title">高级设置</span></template>
            <el-form-item label="副本数"><el-input-number v-model="form.replicas" :min="0" :max="200" controls-position="right" /></el-form-item>
            <el-form-item label="更新策略">
              <el-select v-model="form.strategy" style="width: 180px">
                <el-option label="RollingUpdate" value="RollingUpdate" />
                <el-option label="Recreate" value="Recreate" />
              </el-select>
            </el-form-item>
            <el-form-item label="maxSurge / maxUnavailable" v-if="form.strategy === 'RollingUpdate'">
              <el-input v-model="form.maxSurge" style="width: 130px" />
              <el-input v-model="form.maxUnavailable" style="width: 130px; margin-left: 8px" />
            </el-form-item>
            <el-form-item label="服务账号"><el-input v-model="form.serviceAccount" style="width: 260px" /></el-form-item>
            <el-form-item label="重启策略">
              <el-select v-model="form.restartPolicy" style="width: 180px">
                <el-option label="Always" value="Always" />
                <el-option label="OnFailure" value="OnFailure" />
                <el-option label="Never" value="Never" />
              </el-select>
            </el-form-item>
            <el-form-item label="节点选择"><KeyValueEditor v-model="form.nodeSelector" /></el-form-item>
          </el-collapse-item>
        </el-collapse>
      </el-form>
      <div class="yaml-preview-wrap">
        <div class="preview-head">
          <span>YAML 预览</span>
          <el-button link type="primary" size="small" @click="view = 'yaml'">在 YAML 中编辑</el-button>
        </div>
        <pre class="yaml-preview">{{ formYaml }}</pre>
      </div>
    </div>

    <!-- ============ YAML 模式 ============ -->
    <div v-else class="yaml-wrap">
      <div class="yaml-main">
        <el-input
          v-model="yamlText"
          type="textarea"
          :rows="24"
          placeholder="在此粘贴或编写 Kubernetes YAML"
          class="yaml-editor"
          spellcheck="false"
        />
        <div v-if="errorMsg" class="yaml-error">{{ errorMsg }}</div>
      </div>

      <!-- 字段说明侧栏 -->
      <div class="field-help">
        <div class="help-head">
          <span>字段说明</span>
          <el-button link size="small" @click="helpCollapsed = !helpCollapsed">{{ helpCollapsed ? '展开' : '收起' }}</el-button>
        </div>
        <div v-show="!helpCollapsed" class="help-body">
          <div class="help-kind">{{ helpKindLabel }}</div>
          <div v-for="f in fieldHelp" :key="f.path" class="help-item">
            <div class="help-path">
              <code>{{ f.path }}</code>
              <span v-if="f.required" class="tag-req">必填</span>
            </div>
            <div class="help-desc">{{ f.desc }}</div>
            <div v-if="f.example" class="help-ex"><code>{{ f.example }}</code></div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <span v-if="view === 'form' && !isSimpleKind" class="foot-note">表单未覆盖的字段（如 affinity、securityContext）请切到 YAML 模式编辑</span>
      <span v-else-if="view === 'form' && isSimpleKind" class="foot-note">表单未覆盖的字段请切到 YAML 模式编辑</span>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">
        {{ mode === 'create' ? '创建' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import KeyValueEditor from './KeyValueEditor.vue'
import {
  baseWorkloadForm, buildWorkloadYaml, parseWorkloadYaml,
  emptyCtr, emptyVol,
} from '../utils/workloadYaml'
import {
  FORM_KINDS as SIMPLE_KINDS, baseForm, buildFormYaml, parseFormYaml,
} from '../utils/simpleYaml'
import SimpleResourceForm from './SimpleResourceForm.vue'
import { fieldHelpFor } from '../utils/fieldHelp'

const props = defineProps({
  modelValue: Boolean,
  mode: { type: String, default: 'create' }, // create | edit
  kind: { type: String, required: true },
  namespace: { type: String, default: '' },
  name: { type: String, default: '' },
  template: { type: String, default: '' },
  namespaces: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const visible = ref(props.modelValue)
const view = ref('yaml')
const yamlText = ref('')
const form = ref(baseWorkloadForm())
const saving = ref(false)
const errorMsg = ref('')
const openSections = ref(['basic', 'containers'])
const helpCollapsed = ref(false)
const formAvailable = ref(false)
const simpleForm = ref(null)

// 可表单化的类型
const FORM_KINDS = ['deployment', 'statefulset']
const isSimpleKind = computed(() => SIMPLE_KINDS.includes(String(props.kind || '').toLowerCase()))

watch(() => props.modelValue, async (v) => {
  visible.value = v
  errorMsg.value = ''
  if (!v) return

  formAvailable.value = FORM_KINDS.includes(props.kind) || SIMPLE_KINDS.includes(String(props.kind || '').toLowerCase())
  view.value = formAvailable.value ? 'form' : 'yaml'
  if (isSimpleKind.value) simpleForm.value = baseForm(props.kind, props.namespace || 'default')

  if (props.mode === 'edit') {
    try {
      const params = { kind: props.kind, name: props.name }
      if (props.namespace) params.namespace = props.namespace
      const resp = await fetch('/api/resource?' + new URLSearchParams(params), {
        headers: { Authorization: 'Bearer ' + localStorage.getItem('kubeboard_token') },
      })
      if (!resp.ok) throw new Error((await resp.json()).error || '加载失败')
      yamlText.value = await resp.text()
      syncFormFromYaml()
    } catch (e) {
      ElMessage.error(e.message)
      visible.value = false
      emit('update:modelValue', false)
    }
  } else {
    yamlText.value = props.template || ''
    syncFormFromYaml()
  }
})

watch(visible, (v) => { if (!v) emit('update:modelValue', false) })

function syncFormFromYaml() {
  if (!formAvailable.value) return
  if (isSimpleKind.value) {
    const parsed = parseFormYaml(props.kind, yamlText.value)
    if (parsed) simpleForm.value = parsed
    return
  }
  const parsed = parseWorkloadYaml(yamlText.value)
  if (parsed) {
    // 保留用户可能已改的 ns
    form.value = parsed
  }
}

const formYaml = computed(() => buildWorkloadYaml(form.value))
const simpleFormYaml = computed(() => (simpleForm.value ? buildFormYaml(props.kind, simpleForm.value) : ''))

function probeLabel(k) {
  return { liveness: '存活检查', readiness: '就绪检查', startup: '启动检查' }[k]
}

// 表单 -> YAML：切到 YAML 模式时把表单结果写回
watch(view, (v) => {
  if (isSimpleKind.value) {
    if (v === 'yaml' && simpleForm.value) {
      const built = simpleFormYaml.value
      const reparsed = parseFormYaml(props.kind, yamlText.value)
      const same = reparsed && buildFormYaml(props.kind, reparsed).trim() === yamlText.value.trim()
      if (same || !yamlText.value.trim() || simpleDirty.value) {
        yamlText.value = built
        simpleDirty.value = false
      }
    } else if (v === 'form') {
      syncFormFromYaml()
    }
    return
  }
  if (v === 'yaml' && formAvailable.value) {
    const built = formYaml.value
    // 只在表单内容确实变化时覆盖，避免抹掉手工编辑的额外字段
    const reparsed = parseWorkloadYaml(yamlText.value)
    const same = reparsed && buildWorkloadYaml(reparsed).trim() === yamlText.value.trim()
    if (same || !yamlText.value.trim() || formDirty.value) {
      yamlText.value = built
      formDirty.value = false
    }
  }
  if (v === 'form' && formAvailable.value) {
    syncFormFromYaml()
  }
})

const formDirty = ref(false)
watch(form, () => { if (view.value === 'form') formDirty.value = true }, { deep: true })
const simpleDirty = ref(false)
watch(simpleForm, () => { if (view.value === 'form') simpleDirty.value = true }, { deep: true })

const helpKindLabel = computed(() => {
  const map = {
    deployment: 'Deployment', statefulset: 'StatefulSet', service: 'Service',
    configmap: 'ConfigMap', secret: 'Secret', ingress: 'Ingress',
    pvc: 'PersistentVolumeClaim', job: 'Job', cronjob: 'CronJob',
    daemonset: 'DaemonSet', namespace: 'Namespace', serviceaccount: 'ServiceAccount',
    networkpolicy: 'NetworkPolicy', hpa: 'HorizontalPodAutoscaler',
  }
  return map[props.kind] || props.kind
})
const fieldHelp = computed(() => fieldHelpFor(props.kind))

async function save() {
  saving.value = true
  errorMsg.value = ''
  try {
    const payload = view.value === 'form'
      ? (isSimpleKind.value ? simpleFormYaml.value : formYaml.value)
      : yamlText.value
    await http({
      method: props.mode === 'create' ? 'post' : 'put',
      url: '/resources',
      data: { kind: props.kind, yaml: payload },
    })
    ElMessage.success(props.mode === 'create' ? '创建成功' : '保存成功')
    visible.value = false
    emit('saved')
  } catch (e) {
    errorMsg.value = e.response?.data?.error || e.message
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.mode-bar { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; }
.mode-hint { color: var(--ks-text-secondary, #86909c); font-size: 12px; }
.sec-title { font-weight: 600; font-size: 14px; }
.form-wrap { display: flex; gap: 16px; }
.form-wrap > .el-form { flex: 1; min-width: 0; max-height: 62vh; overflow: auto; padding-right: 8px; }
.yaml-preview-wrap { width: 360px; flex-shrink: 0; display: flex; flex-direction: column; }
.preview-head { display: flex; justify-content: space-between; align-items: center; font-size: 13px; font-weight: 600; margin-bottom: 6px; }
.yaml-wrap { display: flex; gap: 16px; }
.yaml-main { flex: 1; min-width: 0; }
.field-help { width: 320px; flex-shrink: 0; border: 1px solid var(--ks-border, #e4e8ef); border-radius: 8px; padding: 10px 12px; max-height: 62vh; overflow: auto; background: var(--ks-bg-subtle, #fafbfc); }
.help-head { display: flex; justify-content: space-between; align-items: center; font-weight: 600; font-size: 13px; margin-bottom: 8px; }
.help-kind { font-size: 12px; font-weight: 600; color: var(--ks-primary, #2f6fec); margin-bottom: 8px; }
.help-item { padding: 7px 0; border-top: 1px dashed var(--ks-border, #e4e8ef); }
.help-item:first-of-type { border-top: none; }
.help-path { display: flex; align-items: center; gap: 6px; }
.help-path code { font-family: 'SF Mono', Menlo, monospace; font-size: 12px; color: #c7254e; background: #f8f8f8; padding: 1px 5px; border-radius: 3px; }
.tag-req { font-size: 10px; color: #fff; background: #e6443f; border-radius: 3px; padding: 0 4px; }
.help-desc { font-size: 12px; color: var(--ks-text-secondary, #666); line-height: 1.55; margin-top: 3px; }
.help-ex { margin-top: 3px; }
.help-ex code { font-family: 'SF Mono', Menlo, monospace; font-size: 11px; color: #2f6fec; }
.ctr-block { border: 1px solid var(--ks-border, #e4e8ef); border-radius: 8px; padding: 12px 12px 0; margin-bottom: 12px; }
.ctr-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.ctr-no { font-weight: 600; font-size: 13px; }
.port-row, .vol-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.res-grid { display: flex; gap: 24px; }
.res-sub { font-size: 12px; color: var(--ks-text-secondary, #86909c); margin-bottom: 6px; }
.probe-block { border: 1px dashed var(--ks-border, #e4e8ef); border-radius: 6px; padding: 8px 10px; margin-bottom: 8px; }
.probe-body { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 6px; }
.yaml-editor :deep(textarea) { font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace; font-size: 12px; line-height: 1.6; }
.yaml-preview { background: #1e1e1e; color: #d4d4d4; font-family: monospace; font-size: 11.5px; line-height: 1.5; padding: 10px; border-radius: 6px; max-height: 58vh; overflow: auto; white-space: pre-wrap; margin: 0; }
.yaml-error { margin-top: 8px; color: var(--ks-danger, #e6443f); font-size: 12px; }
.foot-note { color: var(--ks-text-secondary, #86909c); font-size: 12px; margin-right: 12px; }
</style>
