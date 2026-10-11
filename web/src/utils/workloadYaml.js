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

function probeYaml(p) {
  const lines = []
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
  if (c.ports && c.ports.length) {
    y += `${indent}  ports:\n`
    for (const p of c.ports) {
      y += `${indent}    - containerPort: ${p.containerPort}\n`
      if (p.name) y += `${indent}      name: ${p.name}\n`
      y += `${indent}      protocol: ${p.protocol}\n`
    }
  }
  const env = kvToObj(c.env)
  if (Object.keys(env).length) {
    y += `${indent}  env:\n`
    for (const [k, v] of Object.entries(env)) y += `${indent}    - name: ${k}\n${indent}      value: ${JSON.stringify(v)}\n`
  }
  const res = resourcesYaml(c)
  if (res) y += res
  for (const [pname, p] of Object.entries(c.probes || {})) {
    if (p.enabled) y += `${indent}  ${pname}Probe:\n${probeYaml(p)}\n`
  }
  return y
}

function volumesYaml(f) {
  if (!f.volumes || !f.volumes.length) return ''
  let y = `      volumes:\n`
  for (const v of f.volumes) {
    y += `        - name: ${v.name}\n`
    if (v.type === 'pvc') y += `          persistentVolumeClaim:\n            claimName: ${v.source || v.name}\n`
    else if (v.type === 'configMap') y += `          configMap:\n            name: ${v.source}\n`
    else if (v.type === 'secret') y += `          secret:\n            secretName: ${v.source}\n`
    else if (v.type === 'hostPath') y += `          hostPath:\n            path: ${v.source}\n`
    else if (v.type === 'emptyDir') y += `          emptyDir: {}\n`
  }
  return y
}

function mountsYaml(f) {
  if (!f.volumes || !f.volumes.length) return ''
  return f.volumes
    .map((v) => `        - name: ${v.name}\n          mountPath: ${v.mountPath}${v.readOnly ? '\n          readOnly: true' : ''}`)
    .join('\n')
}

export function buildWorkloadYaml(f) {
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
  if (Object.keys(tol).length) {
    spec += `      tolerations:\n` + Object.entries(tol).map(([k, v]) => `        - key: ${k}\n          operator: Equal\n          value: ${v}\n          effect: NoSchedule`).join('\n') + '\n'
  }
  const vol = volumesYaml(f)
  if (vol) spec += vol

  const mounts = mountsYaml(f)
  spec += `      containers:\n`
  for (const c of f.containers) {
    let cy = containerYaml(c, '      ')
    // 把 volumeMounts 注入容器（放在 resources 之后、probe 之前的位置即可，K8s 不要求顺序）
    if (mounts) {
      const idx = cy.lastIndexOf('\n', cy.length - 2) + 1
      cy = cy.slice(0, idx) + `      volumeMounts:\n` + mounts.replace(/^ {8}/gm, '        ') + '\n' + cy.slice(idx)
    }
    spec += cy
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
  const vols = podSpec.volumes || []
  const mountByVol = {}
  for (const c of cs) {
    for (const m of c.volumeMounts || []) mountByVol[m.name] = m
  }
  f.volumes = vols.map((v) => {
    const vol = emptyVol()
    vol.name = v.name
    if (v.persistentVolumeClaim) { vol.type = 'pvc'; vol.source = v.persistentVolumeClaim.claimName || '' }
    else if (v.configMap) { vol.type = 'configMap'; vol.source = v.configMap.name || '' }
    else if (v.secret) { vol.type = 'secret'; vol.source = v.secret.secretName || '' }
    else if (v.hostPath) { vol.type = 'hostPath'; vol.source = v.hostPath.path || '' }
    else if (v.emptyDir) { vol.type = 'emptyDir'; vol.source = '' }
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
