import { load } from 'js-yaml'

export function resourceObject(detail) {
  try { return load(detail.rawYaml || '') || {} } catch { return {} }
}
export function displayValue(value) {
  if (value === undefined || value === null) return '未配置'
  if (typeof value === 'boolean') return value ? '是' : '否'
  if (Array.isArray(value)) return value.map(displayValue).join('\n') || '无'
  if (typeof value === 'object') return Object.entries(value).map(([k, v]) => `${k}: ${displayValue(v)}`).join('\n') || '无'
  return String(value)
}
export function environmentContainers(object) {
  const spec = object.spec?.template?.spec || object.spec?.jobTemplate?.spec?.template?.spec || object.spec || {}
  return [...(spec.initContainers || []), ...(spec.containers || [])].map(c => ({
    name: c.name,
    rows: [
      ...(c.env || []).map(e => ({ name: e.name, value: e.valueFrom ? displayValue(e.valueFrom) : e.value ?? '' })),
      ...(c.envFrom || []).map(e => ({ name: `${e.prefix || ''}*`, value: displayValue(e) })),
    ],
  }))
}
export function conditionType(condition) {
  if (!['True', 'False'].includes(condition.status)) return 'info'
  const negative = ['DiskPressure', 'MemoryPressure', 'PIDPressure', 'NetworkUnavailable', 'ReplicaFailure', 'Failed'].includes(condition.type)
  return (condition.status === 'True') !== negative ? 'success' : 'warning'
}
function policyPeers(peers) {
  if (!peers?.length) return '所有地址'
  return peers.map(peer => {
    const parts = []
    if (peer.ipBlock) parts.push(`网段 ${peer.ipBlock.cidr}${peer.ipBlock.except?.length ? '，排除 ' + peer.ipBlock.except.join(', ') : ''}`)
    if (peer.namespaceSelector) parts.push(`项目：${displayValue(peer.namespaceSelector)}`)
    if (peer.podSelector) parts.push(`容器组：${displayValue(peer.podSelector)}`)
    return parts.join(' 且 ') || '所有地址'
  }).join('\n或 ')
}
function policyPorts(ports) {
  return ports?.length ? ports.map(p => `${p.protocol || 'TCP'}/${p.port ?? '所有端口'}${p.endPort ? '–' + p.endPort : ''}`).join('\n') : '所有端口'
}
export function resourceSections(object) {
  const s = object.spec || {}, st = object.status || {}, kind = (object.kind || '').toLowerCase()
  const sections = []
  const attributes = (title, fields) => sections.push({ title, rows: fields.map(([name, value]) => ({ name, value: displayValue(value) })) })
  const table = (title, rows, columns) => sections.push({ title, rows: rows || [], columns })
  const containers = template => {
    table('容器配置', [...(template?.initContainers || []), ...(template?.containers || [])].map(c => ({
      name: c.name, image: c.image, resources: displayValue(c.resources), ports: displayValue(c.ports),
      probes: ['startupProbe', 'readinessProbe', 'livenessProbe'].filter(k => c[k]).join(' / ') || '未配置',
    })), [['name', '容器'], ['image', '镜像'], ['resources', '资源请求 / 限制'], ['ports', '端口'], ['probes', '探针']])
    table('存储卷', (template?.volumes || []).map(v => ({ name: v.name, source: displayValue(Object.fromEntries(Object.entries(v).filter(([k]) => k !== 'name'))) })), [['name', '卷名'], ['source', '来源']])
  }
  if (['deployment', 'statefulset', 'daemonset', 'replicaset'].includes(kind)) {
    attributes('运行与更新', kind === 'daemonset'
      ? [['期望节点', st.desiredNumberScheduled], ['当前节点', st.currentNumberScheduled], ['就绪节点', st.numberReady], ['不可用节点', st.numberUnavailable], ['更新策略', s.updateStrategy]]
      : [['期望副本', s.replicas ?? 1], ['就绪副本', st.readyReplicas ?? 0], ['已更新副本', st.updatedReplicas], ['更新策略', s.strategy || s.updateStrategy], ['服务名', s.serviceName]])
    containers(s.template?.spec)
    attributes('调度配置', [['节点选择器', s.template?.spec?.nodeSelector], ['亲和性', s.template?.spec?.affinity], ['容忍', s.template?.spec?.tolerations]])
  } else if (kind === 'pod') {
    table('容器运行状态', [...(st.initContainerStatuses || []), ...(st.containerStatuses || [])].map(c => {
      const state = c.state?.running ? '运行中' : c.state?.waiting ? '等待中' : c.state?.terminated ? '已终止' : '未知'
      const current = c.state?.waiting || c.state?.terminated || {}
      return { name: c.name, state, ready: c.ready ? '就绪' : '未就绪', restarts: c.restartCount ?? 0, reason: current.reason || '—', lastExit: c.lastState?.terminated ? `${c.lastState.terminated.reason || '退出'} (${c.lastState.terminated.exitCode})` : '—' }
    }), [['name', '容器'], ['state', '状态'], ['ready', '就绪'], ['restarts', '重启次数'], ['reason', '原因'], ['lastExit', '上次终止']])
    containers(s)
    table('调度与就绪', ['PodScheduled', 'Initialized', 'ContainersReady', 'Ready'].map(type => {
      const condition = (st.conditions || []).find(c => c.type === type)
      return { type, status: condition?.status || 'Unknown', reason: condition?.reason || '—', message: condition?.message || '—' }
    }), [['type', '检查项'], ['status', '状态'], ['reason', '原因'], ['message', '说明']])
  } else if (kind === 'pvc' || kind === 'persistentvolumeclaim') {
    attributes('存储与绑定', [['状态', st.phase], ['申请容量', s.resources?.requests?.storage], ['实际容量', st.capacity?.storage], ['访问模式', s.accessModes], ['卷模式', s.volumeMode ?? 'Filesystem'], ['存储类', s.storageClassName], ['绑定 PV', s.volumeName]])
  } else if (kind === 'persistentvolume' || kind === 'pv') {
    attributes('存储与回收', [['状态', st.phase], ['容量', s.capacity?.storage], ['访问模式', s.accessModes], ['卷模式', s.volumeMode ?? 'Filesystem'], ['存储类', s.storageClassName], ['回收策略', s.persistentVolumeReclaimPolicy], ['声明引用', s.claimRef], ['挂载选项', s.mountOptions]])
  } else if (kind === 'storageclass') {
    attributes('存储供应配置', [['供应商', object.provisioner], ['回收策略', object.reclaimPolicy ?? 'Delete'], ['绑定模式', object.volumeBindingMode ?? 'Immediate'], ['允许扩容', object.allowVolumeExpansion ?? false], ['参数', object.parameters], ['挂载选项', object.mountOptions]])
  } else if (kind === 'job' || kind === 'cronjob') {
    if (kind === 'cronjob') attributes('定时执行', [['表达式', s.schedule], ['时区', s.timeZone], ['暂停', s.suspend ?? false], ['并发策略', s.concurrencyPolicy ?? 'Allow'], ['成功历史数', s.successfulJobsHistoryLimit ?? 3], ['失败历史数', s.failedJobsHistoryLimit ?? 1], ['最近调度', st.lastScheduleTime]])
    else attributes('任务执行', [['完成目标', s.completions ?? 1], ['并行度', s.parallelism ?? 1], ['成功', st.succeeded ?? 0], ['失败', st.failed ?? 0], ['执行中', st.active ?? 0], ['重试上限', s.backoffLimit ?? 6], ['开始时间', st.startTime], ['完成时间', st.completionTime]])
    containers(kind === 'job' ? s.template?.spec : s.jobTemplate?.spec?.template?.spec)
  } else if (kind === 'horizontalpodautoscaler') {
    attributes('伸缩目标', [['目标', s.scaleTargetRef], ['最小副本', s.minReplicas ?? 1], ['最大副本', s.maxReplicas], ['当前副本', st.currentReplicas], ['期望副本', st.desiredReplicas], ['指标目标', s.metrics], ['伸缩行为', s.behavior]])
  } else if (kind === 'networkpolicy') {
    attributes('策略范围', [['Pod 选择器', s.podSelector], ['方向', s.policyTypes]])
    table('入站规则', (s.ingress || []).map(r => ({ peers: policyPeers(r.from), ports: policyPorts(r.ports) })), [['peers', '来源'], ['ports', '端口']])
    table('出站规则', (s.egress || []).map(r => ({ peers: policyPeers(r.to), ports: policyPorts(r.ports) })), [['peers', '目标'], ['ports', '端口']])
  } else if (kind === 'serviceaccount') {
    attributes('账号配置', [['自动挂载令牌', s.automountServiceAccountToken ?? object.automountServiceAccountToken ?? '继承 Pod 设置'], ['镜像拉取凭据', object.imagePullSecrets], ['关联凭据', object.secrets]])
  } else if (kind === 'configmap' || kind === 'secret') {
    table(kind === 'secret' ? '保密数据' : '配置数据', Object.entries(object.data || {}).map(([name, value]) => ({ name, value: kind === 'secret' ? '••••••••' : value })), [['name', '键'], ['value', '值']])
    if (kind === 'secret') attributes('凭据类型', [['类型', object.type]])
    if (object.binaryData) table('二进制数据', Object.keys(object.binaryData).map(name => ({ name, value: '二进制内容' })), [['name', '键'], ['value', '类型']])
  }
  return sections
}
