<template>
  <el-dialog v-model="visible" :title="title" width="900px" top="4vh" @closed="reset" class="deploy-dialog">
    <el-form :model="form" label-width="130px" label-position="right">
      <!-- ============ ① 基础信息 ============ -->
      <el-collapse v-model="openSections">
        <el-collapse-item name="basic">
          <template #title><span class="sec-title">基础信息</span></template>
          <el-form-item label="类型">
            <el-radio-group v-model="form.kind">
              <el-radio-button value="deployment">Deployment</el-radio-button>
              <el-radio-button value="statefulset">StatefulSet</el-radio-button>
            </el-radio-group>
            <span class="hint" v-if="form.kind === 'deployment'">无状态应用，支持滚动更新</span>
            <span class="hint" v-else>有状态应用，Pod 名有序、稳定网络标识</span>
          </el-form-item>
          <el-form-item label="项目" required>
            <el-select v-model="form.namespace" filterable :disabled="lockNamespace" style="width: 260px">
              <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
            </el-select>
            <span v-if="lockNamespace" class="hint">已锁定为当前项目</span>
          </el-form-item>
          <el-form-item label="名称" required>
            <el-input v-model="form.name" placeholder="my-app" style="width: 260px" />
          </el-form-item>
          <el-form-item label="描述">
            <el-input v-model="form.description" type="textarea" :rows="2" placeholder="可选，写入注解 kubesphere.io/description" style="width: 480px" />
          </el-form-item>
          <el-form-item label="标签">
            <KeyValueEditor v-model="form.labels" key-placeholder="如 app" value-placeholder="如 my-app" />
          </el-form-item>
          <el-form-item label="注解">
            <KeyValueEditor v-model="form.annotations" key-placeholder="键" value-placeholder="值" />
          </el-form-item>
        </el-collapse-item>

        <!-- ============ ② 容器 ============ -->
        <el-collapse-item name="containers">
          <template #title><span class="sec-title">容器配置（{{ form.containers.length }}）</span></template>
          <div v-for="(ctr, idx) in form.containers" :key="idx" class="ctr-block">
            <div class="ctr-head">
              <span class="ctr-no">容器 {{ idx + 1 }}</span>
              <el-button v-if="form.containers.length > 1" link type="danger" size="small" @click="form.containers.splice(idx, 1)">移除</el-button>
            </div>

            <el-form-item label="容器名" :required="idx === 0">
              <el-input v-model="ctr.name" placeholder="app" style="width: 220px" />
            </el-form-item>
            <el-form-item label="镜像" required>
              <el-input v-model="ctr.image" placeholder="nginx:1.25-alpine" style="width: 380px" />
            </el-form-item>
            <el-form-item label="拉取策略">
              <el-select v-model="ctr.imagePullPolicy" style="width: 180px">
                <el-option label="IfNotPresent（默认）" value="IfNotPresent" />
                <el-option label="Always" value="Always" />
                <el-option label="Never" value="Never" />
              </el-select>
            </el-form-item>

            <el-form-item label="端口">
              <div class="port-list">
                <div v-for="(p, pi) in ctr.ports" :key="pi" class="port-row">
                  <el-input v-model="p.name" placeholder="名称" style="width: 120px" />
                  <el-input-number v-model="p.containerPort" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 130px" />
                  <el-select v-model="p.protocol" style="width: 100px">
                    <el-option label="TCP" value="TCP" />
                    <el-option label="UDP" value="UDP" />
                  </el-select>
                  <el-button link type="danger" size="small" @click="ctr.ports.splice(pi, 1)">删除</el-button>
                </div>
                <el-button link type="primary" size="small" :icon="'Plus'" @click="ctr.ports.push({ name: '', containerPort: 80, protocol: 'TCP' })">添加端口</el-button>
              </div>
            </el-form-item>

            <el-form-item label="启动命令">
              <el-input v-model="ctr.command" placeholder="如 /bin/sh -c（留空用镜像默认，多个用空格分隔）" style="width: 460px" />
            </el-form-item>
            <el-form-item label="参数">
              <el-input v-model="ctr.args" placeholder="如 '-c' 'sleep 3600'" style="width: 460px" />
            </el-form-item>

            <el-form-item label="环境变量">
              <KeyValueEditor v-model="ctr.env" key-placeholder="变量名" value-placeholder="值" />
            </el-form-item>

            <el-form-item label="资源配额">
              <div class="res-grid">
                <div class="res-col">
                  <div class="res-sub">requests（请求）</div>
                  <el-input v-model="ctr.cpuReq" placeholder="CPU 如 100m" style="width: 150px" />
                  <el-input v-model="ctr.memReq" placeholder="内存 如 128Mi" style="width: 150px; margin-top: 6px" />
                  <el-input v-model="ctr.ephemeralReq" placeholder="临时存储 如 1Gi" style="width: 150px; margin-top: 6px" />
                </div>
                <div class="res-col">
                  <div class="res-sub">limits（上限）</div>
                  <el-input v-model="ctr.cpuLim" placeholder="CPU 如 500m" style="width: 150px" />
                  <el-input v-model="ctr.memLim" placeholder="内存 如 512Mi" style="width: 150px; margin-top: 6px" />
                  <el-input v-model="ctr.ephemeralLim" placeholder="临时存储 如 2Gi" style="width: 150px; margin-top: 6px" />
                </div>
              </div>
            </el-form-item>

            <el-form-item label="健康检查">
              <div class="probe-list">
                <div v-for="(probe, pname) in ctr.probes" :key="pname" class="probe-block">
                  <div class="probe-head">
                    <el-checkbox v-model="probe.enabled">{{ probeLabel(pname) }}</el-checkbox>
                  </div>
                  <div v-if="probe.enabled" class="probe-body">
                    <el-select v-model="probe.type" style="width: 120px">
                      <el-option label="HTTP GET" value="httpGet" />
                      <el-option label="TCP" value="tcpSocket" />
                      <el-option label="命令" value="exec" />
                    </el-select>
                    <template v-if="probe.type === 'httpGet'">
                      <el-input v-model="probe.path" placeholder="/healthz" style="width: 150px" />
                      <el-input-number v-model="probe.port" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 120px" />
                    </template>
                    <template v-else-if="probe.type === 'tcpSocket'">
                      <el-input-number v-model="probe.port" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 120px" />
                    </template>
                    <template v-else>
                      <el-input v-model="probe.command" placeholder="命令 如 cat /tmp/ok" style="width: 260px" />
                    </template>
                    <div class="probe-timing">
                      <span>初始延迟 <el-input-number v-model="probe.initialDelaySeconds" :min="0" size="small" controls-position="right" style="width: 90px" />s</span>
                      <span>周期 <el-input-number v-model="probe.periodSeconds" :min="1" size="small" controls-position="right" style="width: 90px" />s</span>
                      <span>超时 <el-input-number v-model="probe.timeoutSeconds" :min="1" size="small" controls-position="right" style="width: 90px" />s</span>
                      <span>失败阈值 <el-input-number v-model="probe.failureThreshold" :min="1" size="small" controls-position="right" style="width: 90px" /></span>
                    </div>
                  </div>
                </div>
              </div>
            </el-form-item>
          </div>
          <el-button link type="primary" :icon="'Plus'" @click="form.containers.push(emptyCtr())">添加容器</el-button>
        </el-collapse-item>

        <!-- ============ ③ 存储与挂载 ============ -->
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
                <el-input v-if="v.type !== 'emptyDir'" v-model="v.source" placeholder="来源(PVC/CM/Secret 名或路径)" style="width: 200px" />
                <el-input v-model="v.mountPath" placeholder="挂载路径 /data" style="width: 160px" />
                <el-checkbox v-model="v.readOnly" style="margin: 0 8px">只读</el-checkbox>
                <el-button link type="danger" size="small" @click="form.volumes.splice(vi, 1)">删除</el-button>
              </div>
              <el-button link type="primary" size="small" :icon="'Plus'" @click="form.volumes.push(emptyVol())">添加挂载</el-button>
            </div>
          </el-form-item>
          <el-form-item label="持久卷模板" v-if="form.kind === 'statefulset'">
            <el-input v-model="form.storageSize" placeholder="如 1Gi（留空则不生成 volumeClaimTemplates）" style="width: 200px" />
            <el-input v-model="form.storageClass" placeholder="存储类（可选）" style="width: 180px; margin-left: 8px" />
          </el-form-item>
        </el-collapse-item>

        <!-- ============ ④ 高级设置 ============ -->
        <el-collapse-item name="advanced">
          <template #title><span class="sec-title">高级设置</span></template>
          <el-form-item label="副本数" v-if="form.kind === 'deployment'">
            <el-input-number v-model="form.replicas" :min="0" :max="200" controls-position="right" />
          </el-form-item>
          <el-form-item label="副本数" v-else>
            <el-input-number v-model="form.replicas" :min="0" :max="200" controls-position="right" />
          </el-form-item>
          <el-form-item label="Headless 服务" v-if="form.kind === 'statefulset'">
            <el-input v-model="form.serviceName" placeholder="留空自动用名称" style="width: 260px" />
          </el-form-item>

          <el-form-item label="更新策略">
            <el-select v-model="form.strategy" style="width: 180px">
              <el-option label="RollingUpdate（滚动）" value="RollingUpdate" />
              <el-option label="Recreate（重建）" value="Recreate" />
            </el-select>
          </el-form-item>
          <el-form-item label="maxSurge / maxUnavailable" v-if="form.strategy === 'RollingUpdate'">
            <el-input v-model="form.maxSurge" placeholder="25%" style="width: 130px" />
            <el-input v-model="form.maxUnavailable" placeholder="25%" style="width: 130px; margin-left: 8px" />
          </el-form-item>

          <el-form-item label="镜像拉取密钥">
            <el-input v-model="form.imagePullSecrets" placeholder="多个用逗号分隔" style="width: 340px" />
          </el-form-item>
          <el-form-item label="服务账号">
            <el-input v-model="form.serviceAccount" placeholder="默认 default" style="width: 260px" />
          </el-form-item>
          <el-form-item label="重启策略">
            <el-select v-model="form.restartPolicy" style="width: 180px">
              <el-option label="Always（默认）" value="Always" />
              <el-option label="OnFailure" value="OnFailure" />
              <el-option label="Never" value="Never" />
            </el-select>
          </el-form-item>

          <el-form-item label="节点选择">
            <KeyValueEditor v-model="form.nodeSelector" key-placeholder="标签键 如 disktype" value-placeholder="值 如 ssd" />
          </el-form-item>
          <el-form-item label="容忍（Tolerations）">
            <KeyValueEditor v-model="form.tolerations" key-placeholder="键" value-placeholder="值（如 NoSchedule）" />
          </el-form-item>

          <el-form-item label="同时创建 Service">
            <el-switch v-model="form.createService" />
            <span v-if="form.createService" class="hint">为该工作负载暴露服务</span>
          </el-form-item>
          <template v-if="form.createService">
            <el-form-item label="Service 名称">
              <el-input v-model="form.svcName" :placeholder="form.name || '自动用工作负载名'" style="width: 260px" />
            </el-form-item>
            <el-form-item label="Service 类型">
              <el-select v-model="form.svcType" style="width: 200px">
                <el-option label="ClusterIP（集群内）" value="ClusterIP" />
                <el-option label="NodePort（节点端口）" value="NodePort" />
                <el-option label="LoadBalancer" value="LoadBalancer" />
              </el-select>
            </el-form-item>
            <el-form-item label="Service 端口">
              <el-input-number v-model="form.svcPort" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 130px" />
              <span class="hint">→ 目标端口</span>
              <el-input-number v-model="form.svcTargetPort" :min="1" :max="65535" controls-position="right" placeholder="目标" style="width: 130px; margin-left: 8px" />
            </el-form-item>
          </template>

          <el-form-item label="同时创建 HPA">
            <el-switch v-model="form.createHPA" />
            <span v-if="form.createHPA" class="hint">基于 CPU 自动扩缩容</span>
          </el-form-item>
          <template v-if="form.createHPA">
            <el-form-item label="HPA 副本范围">
              <el-input-number v-model="form.hpaMin" :min="1" controls-position="right" style="width: 110px" />
              <span class="hint">～</span>
              <el-input-number v-model="form.hpaMax" :min="1" controls-position="right" style="width: 110px" />
            </el-form-item>
            <el-form-item label="CPU 目标利用率">
              <el-input-number v-model="form.hpaCPU" :min="1" :max="100" controls-position="right" style="width: 110px" />
              <span class="hint">%</span>
            </el-form-item>
          </template>
        </el-collapse-item>
      </el-collapse>
    </el-form>

    <!-- YAML 预览 -->
    <el-collapse v-model="previewOpen" style="margin-top: 10px">
      <el-collapse-item name="preview" title="预览生成的 YAML">
        <pre class="yaml-preview">{{ generatedYaml }}</pre>
      </el-collapse-item>
    </el-collapse>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">部署</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'
import KeyValueEditor from './KeyValueEditor.vue'

const props = defineProps({
  modelValue: Boolean,
  defaultKind: { type: String, default: 'deployment' },
  defaultNamespace: { type: String, default: 'default' },
  namespaces: { type: Array, default: () => [] },
  lockNamespace: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'deployed'])

const visible = ref(props.modelValue)
watch(() => props.modelValue, (v) => (visible.value = v))
watch(visible, (v) => emit('update:modelValue', v))

const openSections = ref(['basic', 'containers'])
const previewOpen = ref([])

const emptyProbe = () => ({
  enabled: false, type: 'httpGet', path: '/', port: 80, command: '',
  initialDelaySeconds: 5, periodSeconds: 10, timeoutSeconds: 2, failureThreshold: 3,
})
const emptyCtr = () => ({
  name: 'app', image: '', imagePullPolicy: 'IfNotPresent',
  command: '', args: '', ports: [{ name: 'http', containerPort: 80, protocol: 'TCP' }],
  env: [], cpuReq: '', memReq: '', ephemeralReq: '', cpuLim: '', memLim: '', ephemeralLim: '',
  probes: { liveness: emptyProbe(), readiness: emptyProbe(), startup: emptyProbe() },
})
const emptyVol = () => ({ type: 'pvc', name: 'data', source: '', mountPath: '/data', readOnly: false })

function baseForm() {
  return {
    kind: props.defaultKind, namespace: props.defaultNamespace, name: '', description: '',
    labels: [], annotations: [],
    containers: [emptyCtr()],
    volumes: [],
    replicas: 2, serviceName: '',
    storageSize: '', storageClass: '',
    strategy: 'RollingUpdate', maxSurge: '25%', maxUnavailable: '25%',
    imagePullSecrets: '', serviceAccount: '', restartPolicy: 'Always',
    nodeSelector: [], tolerations: [],
    createService: false, svcName: '', svcType: 'ClusterIP', svcPort: 80, svcTargetPort: 80,
    createHPA: false, hpaMin: 1, hpaMax: 5, hpaCPU: 70,
  }
}
const form = ref(baseForm())
const saving = ref(false)

const title = computed(() => form.value.kind === 'statefulset' ? '部署 StatefulSet' : '部署 Deployment')
function probeLabel(k) {
  return { liveness: '存活检查 (liveness)', readiness: '就绪检查 (readiness)', startup: '启动检查 (startup)' }[k]
}

function reset() { form.value = baseForm(); openSections.value = ['basic', 'containers']; previewOpen.value = [] }

// ---- 生成 YAML ----
function kvToObj(list) {
  const o = {}
  for (const { key, value } of list) if (key) o[key] = value
  return o
}
function splitShell(s) {
  if (!s) return []
  return s.match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
}
function probeYaml(p) {
  let lines = []
  if (p.type === 'httpGet') {
    lines.push(`    httpGet:`)
    lines.push(`      path: ${p.path || '/'}`)
    lines.push(`      port: ${p.port}`)
  } else if (p.type === 'tcpSocket') {
    lines.push(`    tcpSocket:\n      port: ${p.port}`)
  } else {
    lines.push(`    exec:\n      command: ${JSON.stringify(splitShell(p.command))}`)
  }
  lines.push(`    initialDelaySeconds: ${p.initialDelaySeconds}`)
  lines.push(`    periodSeconds: ${p.periodSeconds}`)
  lines.push(`    timeoutSeconds: ${p.timeoutSeconds}`)
  lines.push(`    failureThreshold: ${p.failureThreshold}`)
  return lines.join('\n')
}
function resourcesYaml(c) {
  const req = [], lim = []
  if (c.cpuReq) req.push(`        cpu: ${c.cpuReq}`)
  if (c.memReq) req.push(`        memory: ${c.memReq}`)
  if (c.ephemeralReq) req.push(`        ephemeral-storage: ${c.ephemeralReq}`)
  if (c.cpuLim) lim.push(`        cpu: ${c.cpuLim}`)
  if (c.memLim) lim.push(`        memory: ${c.memLim}`)
  if (c.ephemeralLim) lim.push(`        ephemeral-storage: ${c.ephemeralLim}`)
  let out = ''
  if (req.length || lim.length) {
    out += `      resources:\n`
    if (req.length) out += `        requests:\n${req.join('\n')}\n`
    if (lim.length) out += `        limits:\n${lim.join('\n')}\n`
  }
  return out
}
function containerYaml(c, indent = '    ') {
  let y = `${indent}- name: ${c.name || 'app'}\n`
  y += `${indent}  image: ${c.image || '<image>'}\n`
  if (c.imagePullPolicy) y += `${indent}  imagePullPolicy: ${c.imagePullPolicy}\n`
  const cmds = splitShell(c.command)
  if (cmds.length) y += `${indent}  command: ${JSON.stringify(cmds)}\n`
  const args = splitShell(c.args)
  if (args.length) y += `${indent}  args: ${JSON.stringify(args)}\n`
  if (c.ports.length) {
    y += `${indent}  ports:\n`
    for (const p of c.ports) y += `${indent}    - containerPort: ${p.containerPort}${p.name ? `\n${indent}      name: ${p.name}` : ''}\n${indent}      protocol: ${p.protocol}\n`
  }
  const env = kvToObj(c.env)
  if (Object.keys(env).length) {
    y += `${indent}  env:\n`
    for (const [k, v] of Object.entries(env)) y += `${indent}    - name: ${k}\n${indent}      value: ${JSON.stringify(v)}\n`
  }
  // resources（缩进固定 6 空格，容器缩进 4）
  const res = resourcesYaml(c)
  if (res) y += res
  for (const [pname, p] of Object.entries(c.probes)) {
    if (p.enabled) y += `${indent}  ${pname}Probe:\n${probeYaml(p)}\n`
  }
  return y
}
function volumesYaml() {
  if (!form.value.volumes.length) return ''
  let y = `      volumes:\n`
  for (const v of form.value.volumes) {
    y += `        - name: ${v.name}\n`
    if (v.type === 'pvc') y += `          persistentVolumeClaim:\n            claimName: ${v.source || v.name}\n`
    else if (v.type === 'configMap') y += `          configMap:\n            name: ${v.source}\n`
    else if (v.type === 'secret') y += `          secret:\n            secretName: ${v.source}\n`
    else if (v.type === 'hostPath') y += `          hostPath:\n            path: ${v.source}\n`
    else if (v.type === 'emptyDir') y += `          emptyDir: {}\n`
  }
  return y
}
function mountsYaml() {
  if (!form.value.volumes.length) return ''
  const byCtr = form.value.volumes.map((v) => `          - name: ${v.name}\n            mountPath: ${v.mountPath}${v.readOnly ? '\n            readOnly: true' : ''}`).join('\n')
  return `        volumeMounts:\n${byCtr}\n`
}

const generatedYaml = computed(() => {
  const f = form.value
  const kindName = f.kind === 'statefulset' ? 'StatefulSet' : 'Deployment'
  const labels = kvToObj(f.labels)
  const appLabel = labels.app || f.name || '<name>'
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description

  let meta = `  name: ${f.name || '<name>'}\n  namespace: ${f.namespace}\n`
  if (Object.keys(labels).length) {
    meta += `  labels:\n` + Object.entries(labels).map(([k, v]) => `    ${k}: ${v}`).join('\n') + '\n'
  }
  if (Object.keys(annotations).length) {
    meta += `  annotations:\n` + Object.entries(annotations).map(([k, v]) => `    ${k}: ${JSON.stringify(v)}`).join('\n') + '\n'
  }

  let spec = ''
  if (f.kind === 'statefulset') spec += `  serviceName: ${f.serviceName || f.name || '<name>'}\n`
  spec += `  replicas: ${f.replicas}\n`
  spec += `  selector:\n    matchLabels:\n      app: ${appLabel}\n`
  if (f.strategy && f.kind === 'deployment') {
    spec += `  strategy:\n    type: ${f.strategy}\n`
    if (f.strategy === 'RollingUpdate') {
      spec += `    rollingUpdate:\n      maxSurge: ${f.maxSurge}\n      maxUnavailable: ${f.maxUnavailable}\n`
    }
  }
  spec += `  template:\n    metadata:\n      labels:\n        app: ${appLabel}\n`
  spec += `    spec:\n`
  if (f.imagePullSecrets) {
    spec += `      imagePullSecrets:\n` + f.imagePullSecrets.split(',').map((s) => `        - name: ${s.trim()}`).join('\n') + '\n'
  }
  if (f.serviceAccount) spec += `      serviceAccountName: ${f.serviceAccount}\n`
  if (f.restartPolicy) spec += `      restartPolicy: ${f.restartPolicy}\n`
  const nsSel = kvToObj(f.nodeSelector)
  if (Object.keys(nsSel).length) spec += `      nodeSelector:\n` + Object.entries(nsSel).map(([k, v]) => `        ${k}: ${v}`).join('\n') + '\n'
  const tol = kvToObj(f.tolerations)
  if (Object.keys(tol).length) spec += `      tolerations:\n` + Object.entries(tol).map(([k, v]) => `        - key: ${k}\n          operator: Equal\n          value: ${v}\n          effect: NoSchedule`).join('\n') + '\n'
  const vol = volumesYaml()
  if (vol) spec += vol
  spec += `      containers:\n`
  spec += f.containers.map((c) => containerYaml(c)).join('')
  // 挂载注入到每个容器
  const mounts = mountsYaml()
  if (mounts) {
    spec = spec.replace(/(      containers:\n)([\s\S]*)$/, (m, p1, p2) => {
      const parts = p2.split('\n    - name: ')
      return p1 + parts.map((part, i) => (i === 0 ? part : '    - name: ' + part).replace(/(\n)(?=      (command|args|ports|env|resources|liveness|readiness|startup|imagePullPolicy)|\n*$)/, mounts)).join('')
    })
  }
  if (f.kind === 'statefulset' && f.storageSize) {
    spec += `  volumeClaimTemplates:\n    - metadata:\n        name: data\n      spec:\n        accessModes: [ReadWriteOnce]\n        resources:\n          requests:\n            storage: ${f.storageSize}\n`
    if (f.storageClass) spec += `        storageClassName: ${f.storageClass}\n`
  }

  let docs = `apiVersion: apps/v1\nkind: ${kindName}\nmetadata:\n${meta}spec:\n${spec}`

  if (f.createService) {
    docs += `\n---\napiVersion: v1\nkind: Service\nmetadata:\n  name: ${f.svcName || f.name}\n  namespace: ${f.namespace}\nspec:\n  type: ${f.svcType}\n  selector:\n    app: ${appLabel}\n  ports:\n    - port: ${f.svcPort}\n      targetPort: ${f.svcTargetPort}\n      protocol: TCP\n`
  }
  if (f.createHPA) {
    docs += `\n---\napiVersion: autoscaling/v2\nkind: HorizontalPodAutoscaler\nmetadata:\n  name: ${f.name}-hpa\n  namespace: ${f.namespace}\nspec:\n  scaleTargetRef:\n    apiVersion: apps/v1\n    kind: ${kindName}\n    name: ${f.name}\n  minReplicas: ${f.hpaMin}\n  maxReplicas: ${f.hpaMax}\n  metrics:\n    - type: Resource\n      resource:\n        name: cpu\n        target:\n          type: Utilization\n          averageUtilization: ${f.hpaCPU}\n`
  }
  return docs
})

async function submit() {
  const f = form.value
  if (!f.name) return ElMessage.warning('请填写名称')
  const bad = f.containers.find((c) => !c.image)
  if (bad) return ElMessage.warning('每个容器都需要镜像')
  saving.value = true
  try {
    // 多文档 YAML：逐个创建
    const docs = generatedYaml.value.split(/\n---\n/).filter((d) => d.trim())
    for (const doc of docs) {
      await http.post('/resources', { yaml: doc })
    }
    ElMessage.success(`部署成功${docs.length > 1 ? `（${docs.length} 个资源）` : ''}`)
    visible.value = false
    emit('deployed')
  } catch (e) {
    ElMessage.error('部署失败: ' + (e.response?.data?.error || e?.message || '未知错误'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.sec-title { font-weight: 600; font-size: 14px; }
.ctr-block { border: 1px solid var(--ks-border, #e4e8ef); border-radius: 8px; padding: 12px 12px 0; margin-bottom: 12px; }
.ctr-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 10px; }
.ctr-no { font-weight: 600; font-size: 13px; }
.hint { color: var(--ks-text-secondary); font-size: 12px; margin-left: 8px; }
.port-row, .vol-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.res-grid { display: flex; gap: 24px; }
.res-sub { font-size: 12px; color: var(--ks-text-secondary); margin-bottom: 6px; }
.probe-block { border: 1px dashed var(--ks-border, #e4e8ef); border-radius: 6px; padding: 8px 10px; margin-bottom: 8px; }
.probe-head { margin-bottom: 4px; }
.probe-body { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.probe-timing { display: flex; gap: 12px; align-items: center; font-size: 12px; color: var(--ks-text-secondary); flex-basis: 100%; margin-top: 6px; }
.yaml-preview { background: #1e1e1e; color: #d4d4d4; font-family: monospace; font-size: 12px; line-height: 1.55; padding: 12px; border-radius: 6px; max-height: 360px; overflow: auto; white-space: pre-wrap; }
</style>
