// Service / ConfigMap / Secret 表单 <-> YAML 双向转换
// 与 workloadYaml.js 同构：解析用 js-yaml，生成也用 js-yaml dump

import * as yamlLib from 'js-yaml'
import { kvToObj } from './workloadYaml.js'

const dump = (o) => yamlLib.dump(o, { noRefs: true, lineWidth: 200, quotingType: '"' })
const loadFirst = (text) => {
  const docs = []
  yamlLib.loadAll(String(text), (d) => { if (d) docs.push(d) })
  return docs[0] || null
}

// ---------- Service ----------
export function baseServiceForm(ns = 'default') {
  return {
    namespace: ns, name: '', description: '',
    labels: [], annotations: [],
    type: 'ClusterIP', selector: [],
    ports: [{ name: '', port: 80, targetPort: 80, protocol: 'TCP', nodePort: null }],
    sessionAffinity: 'None', externalTrafficPolicy: 'Cluster',
  }
}

export function buildServiceYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const labels = kvToObj(f.labels)
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description
  const meta = { name: f.name || '<name>', namespace: f.namespace }
  if (Object.keys(labels).length) meta.labels = { ...labels }
  if (Object.keys(annotations).length) meta.annotations = { ...annotations }

  const spec = { type: f.type || 'ClusterIP' }
  const sel = kvToObj(f.selector)
  if (Object.keys(sel).length) spec.selector = { ...sel }
  spec.ports = (f.ports || []).filter((p) => p.port).map((p) => {
    const port = { port: Number(p.port) }
    if (p.name) port.name = p.name
    if (p.targetPort !== '' && p.targetPort != null) port.targetPort = Number.isNaN(Number(p.targetPort)) ? p.targetPort : Number(p.targetPort)
    port.protocol = p.protocol || 'TCP'
    if (f.type === 'NodePort' && p.nodePort) port.nodePort = Number(p.nodePort)
    return port
  })
  if (f.sessionAffinity && f.sessionAffinity !== 'None') {
    spec.sessionAffinity = f.sessionAffinity
  }
  if ((f.type === 'NodePort' || f.type === 'LoadBalancer') && f.externalTrafficPolicy) {
    spec.externalTrafficPolicy = f.externalTrafficPolicy
  }
  return dump({ apiVersion: 'v1', kind: 'Service', metadata: meta, spec }).trimEnd()
}

export function parseServiceYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'Service') return null
  const f = baseServiceForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  f.type = doc.spec?.type || 'ClusterIP'
  f.sessionAffinity = doc.spec?.sessionAffinity || 'None'
  f.externalTrafficPolicy = doc.spec?.externalTrafficPolicy || 'Cluster'
  f.labels = Object.entries(doc.metadata?.labels || {}).map(([key, value]) => ({ key, value: String(value) }))
  const ann = { ...(doc.metadata?.annotations || {}) }
  if (ann['kubesphere.io/description']) {
    f.description = ann['kubesphere.io/description']
    delete ann['kubesphere.io/description']
  }
  for (const k of ['kubectl.kubernetes.io/last-applied-configuration', 'field.cattle.io/publicEndpoints']) delete ann[k]
  f.annotations = Object.entries(ann).map(([key, value]) => ({ key, value: String(value) }))
  f.selector = Object.entries(doc.spec?.selector || {}).map(([key, value]) => ({ key, value: String(value) }))
  f.ports = (doc.spec?.ports || []).map((p) => ({
    name: p.name || '', port: p.port, targetPort: p.targetPort ?? p.port,
    protocol: p.protocol || 'TCP', nodePort: p.nodePort || null,
  }))
  if (!f.ports.length) f.ports = [{ name: '', port: 80, targetPort: 80, protocol: 'TCP', nodePort: null }]
  return f
}

// ---------- ConfigMap ----------
export function baseConfigMapForm(ns = 'default') {
  return { namespace: ns, name: '', description: '', labels: [], annotations: [], data: [], immutable: false }
}

export function buildConfigMapYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const labels = kvToObj(f.labels)
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description
  const meta = { name: f.name || '<name>', namespace: f.namespace }
  if (Object.keys(labels).length) meta.labels = { ...labels }
  if (Object.keys(annotations).length) meta.annotations = { ...annotations }
  const doc = { apiVersion: 'v1', kind: 'ConfigMap', metadata: meta }
  const data = kvToObj(f.data)
  if (Object.keys(data).length) doc.data = { ...data }
  if (f.immutable) doc.immutable = true
  return dump(doc).trimEnd()
}

export function parseConfigMapYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'ConfigMap') return null
  const f = baseConfigMapForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  f.immutable = !!doc.immutable
  f.labels = Object.entries(doc.metadata?.labels || {}).map(([key, value]) => ({ key, value: String(value) }))
  const ann = { ...(doc.metadata?.annotations || {}) }
  if (ann['kubesphere.io/description']) {
    f.description = ann['kubesphere.io/description']
    delete ann['kubesphere.io/description']
  }
  for (const k of ['kubectl.kubernetes.io/last-applied-configuration']) delete ann[k]
  f.annotations = Object.entries(ann).map(([key, value]) => ({ key, value: String(value) }))
  f.data = Object.entries(doc.data || {}).map(([key, value]) => ({ key, value: String(value) }))
  if (!f.data.length) f.data = [{ key: '', value: '' }]
  return f
}

// ---------- Secret ----------
export const SECRET_TYPES = [
  { label: 'Opaque（通用）', value: 'Opaque' },
  { label: 'kubernetes.io/basic-auth（用户名密码）', value: 'kubernetes.io/basic-auth' },
  { label: 'kubernetes.io/ssh-auth（SSH 密钥）', value: 'kubernetes.io/ssh-auth' },
  { label: 'kubernetes.io/dockerconfigjson（镜像仓库）', value: 'kubernetes.io/dockerconfigjson' },
  { label: 'kubernetes.io/tls（TLS 证书）', value: 'kubernetes.io/tls' },
  { label: 'Service Account Token', value: 'kubernetes.io/service-account-token' },
]

export function baseSecretForm(ns = 'default') {
  return { namespace: ns, name: '', description: '', labels: [], annotations: [], type: 'Opaque', data: [], immutable: false }
}

export function buildSecretYaml(f) {
  if (!f || typeof f !== 'object') return ''
  const labels = kvToObj(f.labels)
  const annotations = kvToObj(f.annotations)
  if (f.description) annotations['kubesphere.io/description'] = f.description
  const meta = { name: f.name || '<name>', namespace: f.namespace }
  if (Object.keys(labels).length) meta.labels = { ...labels }
  if (Object.keys(annotations).length) meta.annotations = { ...annotations }
  const doc = { apiVersion: 'v1', kind: 'Secret', metadata: meta, type: f.type || 'Opaque' }
  const data = kvToObj(f.data)
  if (Object.keys(data).length) doc.data = { ...data }
  if (f.immutable) doc.immutable = true
  return dump(doc).trimEnd()
}

export function parseSecretYaml(text) {
  const doc = loadFirst(text)
  if (!doc || doc.kind !== 'Secret') return null
  const f = baseSecretForm(doc.metadata?.namespace || 'default')
  f.name = doc.metadata?.name || ''
  f.type = doc.type || 'Opaque'
  f.immutable = !!doc.immutable
  f.labels = Object.entries(doc.metadata?.labels || {}).map(([key, value]) => ({ key, value: String(value) }))
  const ann = { ...(doc.metadata?.annotations || {}) }
  if (ann['kubesphere.io/description']) {
    f.description = ann['kubesphere.io/description']
    delete ann['kubesphere.io/description']
  }
  for (const k of ['kubectl.kubernetes.io/last-applied-configuration']) delete ann[k]
  f.annotations = Object.entries(ann).map(([key, value]) => ({ key, value: String(value) }))
  f.data = Object.entries(doc.data || {}).map(([key, value]) => ({ key, value: String(value) }))
  if (!f.data.length) f.data = [{ key: '', value: '' }]
  return f
}

// 统一入口
export const FORM_KINDS = ['deployment', 'statefulset', 'service', 'configmap', 'secret']

export function parseFormYaml(kind, text) {
  const k = String(kind || '').toLowerCase()
  if (k === 'service') return parseServiceYaml(text)
  if (k === 'configmap') return parseConfigMapYaml(text)
  if (k === 'secret') return parseSecretYaml(text)
  return null
}

export function buildFormYaml(kind, form) {
  const k = String(kind || '').toLowerCase()
  if (k === 'service') return buildServiceYaml(form)
  if (k === 'configmap') return buildConfigMapYaml(form)
  if (k === 'secret') return buildSecretYaml(form)
  return ''
}

export function baseForm(kind, ns = 'default') {
  const k = String(kind || '').toLowerCase()
  if (k === 'service') return baseServiceForm(ns)
  if (k === 'configmap') return baseConfigMapForm(ns)
  if (k === 'secret') return baseSecretForm(ns)
  return null
}
