// Ingress / Job / CronJob / DaemonSet 表单 <-> YAML 双向转换
// 与 simpleYaml.js 同构：解析用 js-yaml，生成也用 js-yaml dump

import * as yamlLib from 'js-yaml'
import { kvToObj } from './workloadYaml.js'

const dump = (o) => yamlLib.dump(o, { noRefs: true, lineWidth: 200, quotingType: '"' })
const loadFirst = (text) => {
  const docs = []
  yamlLib.loadAll(String(text), (d) => { if (d) docs.push(d) })
  return docs[0] || null
}

// 清理 K8s 自动注入的注解
const INJECTED_ANNOT = [
  'kubectl.kubernetes.io/last-applied-configuration',
  'field.cattle.io/publicEndpoints',
  'deployment.kubernetes.io/revision',
  'kubernetes.io/change-cause',
]
function splitMeta(meta = {}) {
  const labels = Object.entries(meta.labels || {}).map(([key, value]) => ({ key, value: String(value) }))
  const ann = { ...(meta.annotations || {}) }
  let description = ''
  if (ann['kubesphere.io/description']) {
    description = ann['kubesphere.io/description']
    delete ann['kubesphere.io/description']
  }
  for (const k of INJECTED_ANNOT) delete ann[k]
  const annotations = Object.entries(ann).map(([key, value]) => ({ key, value: String(value) }))
  return { labels, annotations, description }
}
function joinMeta(f) {
  const labels = kvToObj(f.labels)
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description
  const meta = {}
  if (Object.keys(labels).length) meta.labels = { ...labels }
  if (Object.keys(annotations).length) meta.annotations = { ...annotations }
  return meta
}

// ==================== Ingress ====================
export function baseIngressForm(ns = 'default') {
  return {
    namespace: ns, name: '', description: '', labels: [], annotations: [],
    ingressClassName: '',
    tls: [],
    rules: [{
      host: '',
      paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }],
    }],
  }
}

export function buildIngressYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const meta = { name: f.name || '<name>', namespace: f.namespace, ...joinMeta(f) }
  const spec = {}
  if (f.ingressClassName) spec.ingressClassName = f.ingressClassName
  const tls = (f.tls || []).filter((t) => t.secretName)
  if (tls.length) {
    spec.tls = tls.map((t) => {
      const o = { hosts: String(t.hosts || '').split(',').map((s) => s.trim()).filter(Boolean) }
      o.secretName = t.secretName
      return o
    })
  }
  spec.rules = (f.rules || []).filter((r) => r.paths?.length).map((r) => {
    const rule = {}
    if (r.host) rule.host = r.host
    rule.http = {
      paths: r.paths.filter((p) => p.path !== '' || p.serviceName).map((p) => {
        const path = { path: p.path || '/', pathType: p.pathType || 'Prefix' }
        path.backend = {
          service: {
            name: p.serviceName || '<service>',
            port: { number: Number(p.servicePort) || 80 },
          },
        }
        return path
      }),
    }
    return rule
  })
  return dump({ apiVersion: 'networking.k8s.io/v1', kind: 'Ingress', metadata: meta, spec }).trimEnd()
}

export function parseIngressYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'Ingress') return null
  const f = baseIngressForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  f.ingressClassName = doc.spec?.ingressClassName || ''
  Object.assign(f, splitMeta(doc.metadata))
  f.tls = (doc.spec?.tls || []).map((t) => ({
    hosts: (t.hosts || []).join(', '), secretName: t.secretName || '',
  }))
  f.rules = (doc.spec?.rules || []).map((r) => ({
    host: r.host || '',
    paths: (r.http?.paths || []).map((p) => ({
      path: p.path || '/',
      pathType: p.pathType || 'Prefix',
      serviceName: p.backend?.service?.name || '',
      servicePort: p.backend?.service?.port?.number ?? p.backend?.service?.port?.name ?? 80,
    })),
  }))
  if (!f.rules.length) f.rules = baseIngressForm().rules
  return f
}

// ==================== Job ====================
export function baseJobForm(ns = 'default') {
  return {
    namespace: ns, name: '', description: '', labels: [], annotations: [],
    completions: 1, parallelism: 1, backoffLimit: 6,
    activeDeadlineSeconds: null, ttlSecondsAfterFinished: null,
    restartPolicy: 'OnFailure',
    containers: [{
      name: 'job', image: '', command: '', args: '',
      env: [], cpuReq: '', memReq: '', cpuLim: '', memLim: '',
    }],
  }
}

function buildJobContainers(containers) {
  return (containers || []).map((c) => {
    const o = { name: c.name || 'job', image: c.image || '<image>' }
    const cmds = String(c.command || '').match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
    if (cmds.length) o.command = cmds
    const args = String(c.args || '').match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
    if (args.length) o.args = args
    const env = kvToObj(c.env)
    if (Object.keys(env).length) o.env = Object.entries(env).map(([name, value]) => ({ name, value: String(value) }))
    const req = {}, lim = {}
    if (c.cpuReq) req.cpu = c.cpuReq
    if (c.memReq) req.memory = c.memReq
    if (c.cpuLim) lim.cpu = c.cpuLim
    if (c.memLim) lim.memory = c.memLim
    if (Object.keys(req).length || Object.keys(lim).length) {
      o.resources = {}
      if (Object.keys(req).length) o.resources.requests = req
      if (Object.keys(lim).length) o.resources.limits = lim
    }
    return o
  })
}

export function buildJobYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const meta = { name: f.name || '<name>', namespace: f.namespace, ...joinMeta(f) }
  const spec = {}
  if (f.completions != null && f.completions !== '') spec.completions = Number(f.completions)
  if (f.parallelism != null && f.parallelism !== '') spec.parallelism = Number(f.parallelism)
  if (f.backoffLimit != null && f.backoffLimit !== '') spec.backoffLimit = Number(f.backoffLimit)
  if (f.activeDeadlineSeconds) spec.activeDeadlineSeconds = Number(f.activeDeadlineSeconds)
  if (f.ttlSecondsAfterFinished) spec.ttlSecondsAfterFinished = Number(f.ttlSecondsAfterFinished)
  spec.template = {
    metadata: { labels: { job: f.name || 'job' } },
    spec: {
      restartPolicy: f.restartPolicy || 'OnFailure',
      containers: buildJobContainers(f.containers),
    },
  }
  return dump({ apiVersion: 'batch/v1', kind: 'Job', metadata: meta, spec }).trimEnd()
}

export function parseJobYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'Job') return null
  const f = baseJobForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  Object.assign(f, splitMeta(doc.metadata))
  f.completions = doc.spec?.completions ?? 1
  f.parallelism = doc.spec?.parallelism ?? 1
  f.backoffLimit = doc.spec?.backoffLimit ?? 6
  f.activeDeadlineSeconds = doc.spec?.activeDeadlineSeconds ?? null
  f.ttlSecondsAfterFinished = doc.spec?.ttlSecondsAfterFinished ?? null
  f.restartPolicy = doc.spec?.template?.spec?.restartPolicy || 'OnFailure'
  const cs = doc.spec?.template?.spec?.containers || []
  if (cs.length) {
    f.containers = cs.map((c) => {
      const o = baseJobForm().containers[0]
      return {
        name: c.name || 'job', image: c.image || '',
        command: (c.command || []).join(' '), args: (c.args || []).join(' '),
        env: (c.env || []).filter((e) => e.name).map((e) => ({ key: e.name, value: e.value ?? '' })),
        cpuReq: c.resources?.requests?.cpu || '', memReq: c.resources?.requests?.memory || '',
        cpuLim: c.resources?.limits?.cpu || '', memLim: c.resources?.limits?.memory || '',
      }
    })
  }
  return f
}

// ==================== CronJob ====================
export function baseCronJobForm(ns = 'default') {
  return {
    ...baseJobForm(ns),
    schedule: '0 2 * * *',
    concurrencyPolicy: 'Allow',
    successfulJobsHistoryLimit: 3,
    failedJobsHistoryLimit: 1,
    suspend: false,
    startingDeadlineSeconds: null,
    name: '',
  }
}

export function buildCronJobYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const meta = { name: f.name || '<name>', namespace: f.namespace, ...joinMeta(f) }
  const jobSpec = {}
  if (f.backoffLimit != null && f.backoffLimit !== '') jobSpec.backoffLimit = Number(f.backoffLimit)
  if (f.activeDeadlineSeconds) jobSpec.activeDeadlineSeconds = Number(f.activeDeadlineSeconds)
  if (f.ttlSecondsAfterFinished) jobSpec.ttlSecondsAfterFinished = Number(f.ttlSecondsAfterFinished)
  jobSpec.template = {
    metadata: { labels: { job: f.name || 'job' } },
    spec: {
      restartPolicy: f.restartPolicy || 'OnFailure',
      containers: buildJobContainers(f.containers),
    },
  }
  const spec = {
    schedule: f.schedule || '0 2 * * *',
    jobTemplate: { spec: jobSpec },
  }
  if (f.concurrencyPolicy) spec.concurrencyPolicy = f.concurrencyPolicy
  if (f.successfulJobsHistoryLimit != null) spec.successfulJobsHistoryLimit = Number(f.successfulJobsHistoryLimit)
  if (f.failedJobsHistoryLimit != null) spec.failedJobsHistoryLimit = Number(f.failedJobsHistoryLimit)
  if (f.startingDeadlineSeconds) spec.startingDeadlineSeconds = Number(f.startingDeadlineSeconds)
  if (f.suspend) spec.suspend = true
  return dump({ apiVersion: 'batch/v1', kind: 'CronJob', metadata: meta, spec }).trimEnd()
}

export function parseCronJobYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'CronJob') return null
  const f = baseCronJobForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  Object.assign(f, splitMeta(doc.metadata))
  f.schedule = doc.spec?.schedule || '0 2 * * *'
  f.concurrencyPolicy = doc.spec?.concurrencyPolicy || 'Allow'
  f.successfulJobsHistoryLimit = doc.spec?.successfulJobsHistoryLimit ?? 3
  f.failedJobsHistoryLimit = doc.spec?.failedJobsHistoryLimit ?? 1
  f.suspend = !!doc.spec?.suspend
  f.startingDeadlineSeconds = doc.spec?.startingDeadlineSeconds ?? null
  const js = doc.spec?.jobTemplate?.spec || {}
  f.backoffLimit = js.backoffLimit ?? 6
  f.activeDeadlineSeconds = js.activeDeadlineSeconds ?? null
  f.ttlSecondsAfterFinished = js.ttlSecondsAfterFinished ?? null
  f.restartPolicy = js.template?.spec?.restartPolicy || 'OnFailure'
  const cs = js.template?.spec?.containers || []
  if (cs.length) {
    f.containers = cs.map((c) => ({
      name: c.name || 'job', image: c.image || '',
      command: (c.command || []).join(' '), args: (c.args || []).join(' '),
      env: (c.env || []).filter((e) => e.name).map((e) => ({ key: e.name, value: e.value ?? '' })),
      cpuReq: c.resources?.requests?.cpu || '', memReq: c.resources?.requests?.memory || '',
      cpuLim: c.resources?.limits?.cpu || '', memLim: c.resources?.limits?.memory || '',
    }))
  }
  return f
}

// ==================== DaemonSet ====================
export function baseDaemonSetForm(ns = 'default') {
  return {
    namespace: ns, name: '', description: '', labels: [], annotations: [],
    updateStrategy: 'RollingUpdate', maxUnavailable: 1,
    containers: [{
      name: 'app', image: '', imagePullPolicy: 'IfNotPresent',
      command: '', args: '',
      env: [], ports: [],
      cpuReq: '', memReq: '', cpuLim: '', memLim: '',
      hostNetwork: false, hostPID: false,
    }],
    nodeSelector: [], tolerations: [],
    serviceAccount: '',
  }
}

function buildDaemonContainers(containers) {
  return (containers || []).map((c) => {
    const o = { name: c.name || 'app', image: c.image || '<image>' }
    if (c.imagePullPolicy) o.imagePullPolicy = c.imagePullPolicy
    const cmds = String(c.command || '').match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
    if (cmds.length) o.command = cmds
    const args = String(c.args || '').match(/"[^"]*"|'[^']*'|\S+/g)?.map((x) => x.replace(/^["']|["']$/g, '')) || []
    if (args.length) o.args = args
    if (c.ports?.length) {
      o.ports = c.ports.filter((p) => p.containerPort).map((p) => {
        const port = { containerPort: Number(p.containerPort) }
        if (p.name) port.name = p.name
        port.protocol = p.protocol || 'TCP'
        return port
      })
    }
    const env = kvToObj(c.env)
    if (Object.keys(env).length) o.env = Object.entries(env).map(([name, value]) => ({ name, value: String(value) }))
    const req = {}, lim = {}
    if (c.cpuReq) req.cpu = c.cpuReq
    if (c.memReq) req.memory = c.memReq
    if (c.cpuLim) lim.cpu = c.cpuLim
    if (c.memLim) lim.memory = c.memLim
    if (Object.keys(req).length || Object.keys(lim).length) {
      o.resources = {}
      if (Object.keys(req).length) o.resources.requests = req
      if (Object.keys(lim).length) o.resources.limits = lim
    }
    return o
  })
}

export function buildDaemonSetYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const labels = kvToObj(f.labels)
  const appLabel = labels.app || f.name || '<name>'
  const meta = { name: f.name || '<name>', namespace: f.namespace, ...joinMeta(f) }
  const podSpec = {
    containers: buildDaemonContainers(f.containers),
    restartPolicy: 'Always',
  }
  if (f.containers?.[0]?.hostNetwork) podSpec.hostNetwork = true
  if (f.containers?.[0]?.hostPID) podSpec.hostPID = true
  if (f.serviceAccount) podSpec.serviceAccountName = f.serviceAccount
  const nsSel = kvToObj(f.nodeSelector)
  if (Object.keys(nsSel).length) podSpec.nodeSelector = { ...nsSel }
  const tol = kvToObj(f.tolerations)
  if (Object.keys(tol).length) {
    podSpec.tolerations = Object.entries(tol).map(([key, value]) => ({ key, operator: 'Equal', value, effect: 'NoSchedule' }))
  }
  const spec = {
    selector: { matchLabels: { app: appLabel } },
    template: { metadata: { labels: { app: appLabel } }, spec: podSpec },
  }
  if (f.updateStrategy) {
    spec.updateStrategy = { type: f.updateStrategy }
    if (f.updateStrategy === 'RollingUpdate' && f.maxUnavailable) {
      // K8s 校验：maxUnavailable 为百分比时必须带 %，为数字时须是整数
      const v = String(f.maxUnavailable).trim()
      const norm = v.endsWith('%') ? v : (Number.isNaN(Number(v)) ? '1' : Number(v))
      spec.updateStrategy.rollingUpdate = { maxUnavailable: norm }
    }
  }
  return dump({ apiVersion: 'apps/v1', kind: 'DaemonSet', metadata: meta, spec }).trimEnd()
}

export function parseDaemonSetYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'DaemonSet') return null
  const f = baseDaemonSetForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  Object.assign(f, splitMeta(doc.metadata))
  const spec = doc.spec || {}
  f.updateStrategy = spec.updateStrategy?.type || 'RollingUpdate'
  f.maxUnavailable = String(spec.updateStrategy?.rollingUpdate?.maxUnavailable ?? '1')
  const ps = spec.template?.spec || {}
  f.serviceAccount = ps.serviceAccountName || ''
  f.nodeSelector = Object.entries(ps.nodeSelector || {}).map(([key, value]) => ({ key, value: String(value) }))
  f.tolerations = (ps.tolerations || []).filter((t) => t.key).map((t) => ({ key: t.key, value: t.value || '' }))
  const cs = ps.containers || []
  if (cs.length) {
    f.containers = cs.map((c) => ({
      name: c.name || 'app', image: c.image || '',
      imagePullPolicy: c.imagePullPolicy || 'IfNotPresent',
      command: (c.command || []).join(' '), args: (c.args || []).join(' '),
      env: (c.env || []).filter((e) => e.name).map((e) => ({ key: e.name, value: e.value ?? '' })),
      ports: (c.ports || []).map((p) => ({ name: p.name || '', containerPort: p.containerPort, protocol: p.protocol || 'TCP' })),
      cpuReq: c.resources?.requests?.cpu || '', memReq: c.resources?.requests?.memory || '',
      cpuLim: c.resources?.limits?.cpu || '', memLim: c.resources?.limits?.memory || '',
      hostNetwork: !!ps.hostNetwork, hostPID: !!ps.hostPID,
    }))
  }
  return f
}

// 统一入口扩展
export const EXTRA_KINDS = ['ingress', 'job', 'cronjob', 'daemonset']

export function parseExtraYaml(kind, text) {
  const k = String(kind || '').toLowerCase()
  if (k === 'ingress') return parseIngressYaml(text)
  if (k === 'job') return parseJobYaml(text)
  if (k === 'cronjob') return parseCronJobYaml(text)
  if (k === 'daemonset') return parseDaemonSetYaml(text)
  return null
}

export function buildExtraYaml(kind, form) {
  const k = String(kind || '').toLowerCase()
  if (k === 'ingress') return buildIngressYaml(form)
  if (k === 'job') return buildJobYaml(form)
  if (k === 'cronjob') return buildCronJobYaml(form)
  if (k === 'daemonset') return buildDaemonSetYaml(form)
  return ''
}

export function baseExtraForm(kind, ns = 'default') {
  const k = String(kind || '').toLowerCase()
  if (k === 'ingress') return baseIngressForm(ns)
  if (k === 'job') return baseJobForm(ns)
  if (k === 'cronjob') return baseCronJobForm(ns)
  if (k === 'daemonset') return baseDaemonSetForm(ns)
  return null
}
