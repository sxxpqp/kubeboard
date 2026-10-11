// 工作负载表单 <-> YAML 双向转换（Deployment / StatefulSet）
// 被 WorkloadDeployDialog（创建）与 YamlDialog（编辑，表单模式）共用

export function kvToObj(list) {
  const o = {}
  for (const { key, value } of list || []) if (key) o[key] = value
  return o
}

export function objToKv(obj) {
  if (!obj) return []
  return Object.entries(obj).map(([key, value]) => ({ key, value: String(value ?? '') }))
}

export function splitShell(s) {
  if (!s || !Array.isArray(s) && typeof s !== 'string') return []
  if (Array.isArray(s)) return s
  return s.match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
}

export function emptyProbe() {
  return {
    enabled: false, type: 'httpGet', path: '/', port: 80, command: '',
    initialDelaySeconds: 5, periodSeconds: 10, timeoutSeconds: 2, failureThreshold: 3,
  }
}

export function emptyCtr() {
  return {
    name: 'app', image: '', imagePullPolicy: 'IfNotPresent',
    command: '', args: '', ports: [{ name: 'http', containerPort: 80, protocol: 'TCP' }],
    env: [], cpuReq: '', memReq: '', ephemeralReq: '', cpuLim: '', memLim: '', ephemeralLim: '',
    probes: { liveness: emptyProbe(), readiness: emptyProbe(), startup: emptyProbe() },
  }
}

export function emptyVol() {
  return { type: 'pvc', name: 'data', source: '', mountPath: '/data', readOnly: false }
}

export function baseWorkloadForm(kind = 'deployment', ns = 'default') {
  return {
    kind, namespace: ns, name: '', description: '',
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

function splitScalarToNum(v) {
  // '80' -> 80；'80%' 保持字符串
  if (v == null || v === '') return undefined
  return Number.isNaN(Number(v)) ? v : Number(v)
}

function buildProbe(p) {
  const o = {}
  if (p.type === 'httpGet') {
    o.httpGet = { path: p.path || '/', port: splitScalarToNum(p.port) }
  } else if (p.type === 'tcpSocket') {
    o.tcpSocket = { port: splitScalarToNum(p.port) }
  } else {
    o.exec = { command: splitShell(p.command) }
  }
  o.initialDelaySeconds = Number(p.initialDelaySeconds) || 0
  o.periodSeconds = Number(p.periodSeconds) || 10
  o.timeoutSeconds = Number(p.timeoutSeconds) || 1
  o.failureThreshold = Number(p.failureThreshold) || 3
  return o
}

function buildContainer(c) {
  const o = { name: c.name || 'app', image: c.image || '<image>' }
  if (c.imagePullPolicy) o.imagePullPolicy = c.imagePullPolicy
  const cmds = splitShell(c.command)
  if (cmds.length) o.command = cmds
  const args = splitShell(c.args)
  if (args.length) o.args = args
  if (c.ports && c.ports.length) {
    o.ports = c.ports.map((p) => {
      const port = { containerPort: Number(p.containerPort) }
      if (p.name) port.name = p.name
      port.protocol = p.protocol || 'TCP'
      return port
    })
  }
  const env = kvToObj(c.env)
  if (Object.keys(env).length) {
    o.env = Object.entries(env).map(([name, value]) => ({ name, value: String(value) }))
  }
  const req = {}, lim = {}
  if (c.cpuReq) req.cpu = c.cpuReq
  if (c.memReq) req.memory = c.memReq
  if (c.ephemeralReq) req['ephemeral-storage'] = c.ephemeralReq
  if (c.cpuLim) lim.cpu = c.cpuLim
  if (c.memLim) lim.memory = c.memLim
  if (c.ephemeralLim) lim['ephemeral-storage'] = c.ephemeralLim
  if (Object.keys(req).length || Object.keys(lim).length) {
    o.resources = {}
    if (Object.keys(req).length) o.resources.requests = req
    if (Object.keys(lim).length) o.resources.limits = lim
  }
  for (const pname of ['liveness', 'readiness', 'startup']) {
    const p = c.probes?.[pname]
    if (p?.enabled) o[`${pname}Probe`] = buildProbe(p)
  }
  return o
}

export function buildWorkloadYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const kindName = f.kind === 'statefulset' ? 'StatefulSet' : 'Deployment'
  const labels = kvToObj(f.labels)
  const appLabel = labels.app || f.name || '<name>'
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description

  const meta = { name: f.name || '<name>', namespace: f.namespace }
  if (Object.keys(labels).length) meta.labels = { ...labels }
  if (Object.keys(annotations).length) meta.annotations = { ...annotations }

  const podSpec = { containers: f.containers.map(buildContainer) }

  // 卷 -> spec.volumes；挂载 -> 每个容器 volumeMounts
  const mounts = []
  if (f.volumes && f.volumes.length) {
    podSpec.volumes = f.volumes.map((v) => {
      const vol = { name: v.name }
      if (v.type === 'pvc') vol.persistentVolumeClaim = { claimName: v.source || v.name }
      else if (v.type === 'configMap') vol.configMap = { name: v.source }
      else if (v.type === 'secret') vol.secret = { secretName: v.source }
      else if (v.type === 'hostPath') vol.hostPath = { path: v.source }
      else if (v.type === 'emptyDir') vol.emptyDir = {}
      if (v.mountPath) {
        const m = { name: v.name, mountPath: v.mountPath }
        if (v.readOnly) m.readOnly = true
        mounts.push(m)
      }
      return vol
    })
  }
  if (mounts.length) {
    // 挂载注入到第一个容器（表单以单挂载点为准；多容器共享同一挂载时按第一个容器处理）
    podSpec.containers[0].volumeMounts = [...mounts]
  }

  if (f.imagePullSecrets) {
    podSpec.imagePullSecrets = f.imagePullSecrets.split(',').map((s) => ({ name: s.trim() })).filter((x) => x.name)
  }
  if (f.serviceAccount) podSpec.serviceAccountName = f.serviceAccount
  if (f.restartPolicy) podSpec.restartPolicy = f.restartPolicy
  const nsSel = kvToObj(f.nodeSelector)
  if (Object.keys(nsSel).length) podSpec.nodeSelector = { ...nsSel }
  const tol = kvToObj(f.tolerations)
  if (Object.keys(tol).length) {
    podSpec.tolerations = Object.entries(tol).map(([key, value]) => ({
      key, operator: 'Equal', value, effect: 'NoSchedule',
    }))
  }

  const spec = {}
  if (f.kind === 'statefulset') spec.serviceName = f.serviceName || f.name || '<name>'
  spec.replicas = Number(f.replicas) || 0
  spec.selector = { matchLabels: { app: appLabel } }
  if (f.kind === 'deployment') {
    if (f.strategy) spec.strategy = { type: f.strategy }
    if (f.strategy === 'RollingUpdate' && (f.maxSurge || f.maxUnavailable)) {
      spec.strategy.rollingUpdate = {}
      if (f.maxSurge) spec.strategy.rollingUpdate.maxSurge = f.maxSurge
      if (f.maxUnavailable) spec.strategy.rollingUpdate.maxUnavailable = f.maxUnavailable
    }
  }
  spec.template = { metadata: { labels: { app: appLabel } }, spec: podSpec }
  if (f.kind === 'statefulset' && f.storageSize) {
    const vctSpec = {
      accessModes: ['ReadWriteOnce'],
      resources: { requests: { storage: f.storageSize } },
    }
    if (f.storageClass) vctSpec.storageClassName = f.storageClass
    spec.volumeClaimTemplates = [{ metadata: { name: 'data' }, spec: vctSpec }]
  }

  const docs = [{ apiVersion: 'apps/v1', kind: kindName, metadata: meta, spec }]

  if (f.createService) {
    const svc = {
      apiVersion: 'v1',
      kind: 'Service',
      metadata: { name: f.svcName || f.name, namespace: f.namespace },
      spec: {
        type: f.svcType || 'ClusterIP',
        selector: { app: appLabel },
        ports: [{ port: Number(f.svcPort) || 80, targetPort: Number(f.svcTargetPort) || 80, protocol: 'TCP' }],
      },
    }
    docs.push(svc)
  }
  if (f.createHPA) {
    const hpa = {
      apiVersion: 'autoscaling/v2',
      kind: 'HorizontalPodAutoscaler',
      metadata: { name: `${f.name}-hpa`, namespace: f.namespace },
      spec: {
        scaleTargetRef: { apiVersion: 'apps/v1', kind: kindName, name: f.name },
        minReplicas: Number(f.hpaMin) || 1,
        maxReplicas: Number(f.hpaMax) || 5,
        metrics: [{
          type: 'Resource',
          resource: { name: 'cpu', target: { type: 'Utilization', averageUtilization: Number(f.hpaCPU) || 70 } },
        }],
      },
    }
    docs.push(hpa)
  }

  return docs.map((d) => dumpYaml(d).trimEnd()).join('\n---\n')
}

// ---------- YAML -> 表单（供编辑模式回填）----------
// 注意：只解析表单覆盖的字段；其余字段在 YamlDialog 里以「保留原文」策略处理
export function parseWorkloadYaml(yamlText) {
  if (!yamlText || !yamlText.trim()) return null
  let doc
  try {
    const docs = parseYamlDocs(yamlText)
    doc = docs.find((d) => d && (d.kind === 'Deployment' || d.kind === 'StatefulSet')) || docs[0]
  } catch {
    return null
  }
  if (!doc || !doc.kind) return null
  const kind = String(doc.kind).toLowerCase()
  if (kind !== 'deployment' && kind !== 'statefulset') return null

  const f = baseWorkloadForm(kind === 'statefulset' ? 'statefulset' : 'deployment', doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  f.replicas = doc.spec?.replicas ?? 1

  // 顶层标签：优先 metadata.labels，为空时回退到 selector.matchLabels（真实清单常见）
  const metaLabels = doc.metadata?.labels || {}
  const selLabels = doc.spec?.selector?.matchLabels || {}
  const labels = Object.keys(metaLabels).length ? metaLabels : selLabels
  f.labels = objToKv(labels)
  // 清理 K8s 自动注入的注解，避免回写时污染
  const ann = { ...(doc.metadata?.annotations || {}) }
  for (const k of [
    'deployment.kubernetes.io/revision', 'kubectl.kubernetes.io/last-applied-configuration',
    'field.cattle.io/publicEndpoints', 'kubernetes.io/change-cause',
  ]) delete ann[k]
  if (ann['kubesphere.io/description']) {
    f.description = ann['kubesphere.io/description']
    delete ann['kubesphere.io/description']
  }
  f.annotations = objToKv(ann)

  const podSpec = doc.spec?.template?.spec || {}
  f.serviceAccount = podSpec.serviceAccountName || ''
  f.restartPolicy = podSpec.restartPolicy || 'Always'
  f.nodeSelector = objToKv(podSpec.nodeSelector)
  f.tolerations = (podSpec.tolerations || []).filter((t) => t.key).map((t) => ({ key: t.key, value: t.value || '' }))
  f.imagePullSecrets = (podSpec.imagePullSecrets || []).map((s) => s.name).filter(Boolean).join(', ')

  if (doc.spec?.strategy?.type) f.strategy = doc.spec.strategy.type
  if (doc.spec?.strategy?.rollingUpdate?.maxSurge != null) f.maxSurge = String(doc.spec.strategy.rollingUpdate.maxSurge)
  if (doc.spec?.strategy?.rollingUpdate?.maxUnavailable != null) f.maxUnavailable = String(doc.spec.strategy.rollingUpdate.maxUnavailable)

  if (kind === 'statefulset') {
    f.serviceName = doc.spec?.serviceName || ''
    const vct = doc.spec?.volumeClaimTemplates?.[0]
    if (vct) {
      f.storageSize = vct.spec?.resources?.requests?.storage || ''
      f.storageClass = vct.spec?.storageClassName || ''
    }
  }

  const cs = podSpec.containers || []
  if (cs.length) {
    f.containers = cs.map((c) => {
      const ctr = emptyCtr()
      ctr.name = c.name || 'app'
      ctr.image = c.image || ''
      ctr.imagePullPolicy = c.imagePullPolicy || 'IfNotPresent'
      ctr.command = (c.command || []).join(' ')
      ctr.args = (c.args || []).join(' ')
      ctr.ports = (c.ports || []).map((p) => ({
        name: p.name || '', containerPort: p.containerPort, protocol: p.protocol || 'TCP',
      }))
      ctr.env = (c.env || []).filter((e) => e.name).map((e) => ({ key: e.name, value: e.value ?? '' }))
      const r = c.resources || {}
      ctr.cpuReq = r.requests?.cpu || ''
      ctr.memReq = r.requests?.memory || ''
      ctr.ephemeralReq = r.requests?.['ephemeral-storage'] || ''
      ctr.cpuLim = r.limits?.cpu || ''
      ctr.memLim = r.limits?.memory || ''
      ctr.ephemeralLim = r.limits?.['ephemeral-storage'] || ''
      for (const pname of ['liveness', 'readiness', 'startup']) {
        const p = c[`${pname}Probe`]
        if (p) {
          const ep = emptyProbe()
          ep.enabled = true
          if (p.httpGet) { ep.type = 'httpGet'; ep.path = p.httpGet.path || '/'; ep.port = p.httpGet.port }
          else if (p.tcpSocket) { ep.type = 'tcpSocket'; ep.port = p.tcpSocket.port }
          else if (p.exec) { ep.type = 'exec'; ep.command = (p.exec.command || []).join(' ') }
          ep.initialDelaySeconds = p.initialDelaySeconds ?? 5
          ep.periodSeconds = p.periodSeconds ?? 10
          ep.timeoutSeconds = p.timeoutSeconds ?? 2
          ep.failureThreshold = p.failureThreshold ?? 3
          ctr.probes[pname] = ep
        }
      }
      return ctr
    })
  }

  // 卷 & 挂载
  const vols = [...(podSpec.volumes || [])]
  const mountByVol = {}
  for (const c of cs) {
    for (const m of c.volumeMounts || []) mountByVol[m.name] = m
  }
  // volumeClaimTemplates（StatefulSet）：作为 pvc 类型卷入列
  for (const vct of doc.spec?.volumeClaimTemplates || []) {
    const name = vct?.metadata?.name
    if (name && !vols.some((v) => v.name === name)) {
      vols.push({ name, persistentVolumeClaim: { claimName: name } })
    }
  }
  // 仅有 volumeMounts、无对应卷定义的（如 STS 模板卷）也补进来，避免挂载信息丢失
  for (const [name, m] of Object.entries(mountByVol)) {
    if (!vols.some((v) => v?.name === name)) {
      vols.push({ name, __mountOnly: true })
    }
  }
  f.volumes = vols.map((v) => {
    const vol = emptyVol()
    vol.name = v.name
    if (v.persistentVolumeClaim) { vol.type = 'pvc'; vol.source = v.persistentVolumeClaim.claimName || '' }
    else if (v.configMap) { vol.type = 'configMap'; vol.source = v.configMap.name || '' }
    else if (v.secret) { vol.type = 'secret'; vol.source = v.secret.secretName || '' }
    else if (v.hostPath) { vol.type = 'hostPath'; vol.source = v.hostPath.path || '' }
    else if (v.emptyDir) { vol.type = 'emptyDir'; vol.source = '' }
    else if (v.__mountOnly) { vol.type = 'emptyDir'; vol.source = '' } // 仅有挂载的模板卷
    const m = mountByVol[v.name]
    if (m) { vol.mountPath = m.mountPath || ''; vol.readOnly = !!m.readOnly }
    return vol
  })

  return f
}

// ---- YAML 解析（使用 js-yaml，多文档安全）----
import * as yamlLib from 'js-yaml'

export function parseYamlSubset(text) {
  const docs = []
  yamlLib.loadAll(String(text), (d) => { docs.push(d) })
  return docs.length ? docs[0] : {}
}

export function parseYamlDocs(text) {
  const docs = []
  try {
    yamlLib.loadAll(String(text), (d) => { if (d) docs.push(d) })
  } catch (e) {
    throw new Error('YAML 解析失败: ' + e.message)
  }
  return docs
}

export function dumpYaml(obj) {
  return yamlLib.dump(obj, { noRefs: true, lineWidth: 200, quotingType: '"' })
}

function splitKV(s) {
  const m = s.match(/^([^:]+):\s?(.*)$/)
  if (!m) return { key: s, val: '' }
  return { key: m[1].trim(), val: m[2] }
}

function scalar(v) {
  if (v === '' || v === undefined) return ''
  if (v === '{}') return {}
  if (v === '[]') return []
  if (/^".*"$/.test(v)) return v.slice(1, -1).replace(/\\"/g, '"')
  if (/^'.*'$/.test(v)) return v.slice(1, -1).replace(/''/g, "'")
  if (v === 'true') return true
  if (v === 'false') return false
  if (v === 'null' || v === '~') return null
  if (/^-?\d+$/.test(v)) return parseInt(v, 10)
  if (/^-?\d*\.\d+$/.test(v)) return parseFloat(v)
  if (v.startsWith('[') && v.endsWith(']')) {
    const inner = v.slice(1, -1).trim()
    if (!inner) return []
    return inner.split(',').map((x) => scalar(x.trim()))
  }
  return v
}
