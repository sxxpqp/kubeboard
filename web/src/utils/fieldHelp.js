// 各资源类型的关键字段说明（YAML 编辑模式侧栏）
// 目的：让使用者知道字段含义、是否必填、怎么写，而不是面对空白文本框

const COMMON = [
  { path: 'apiVersion', required: true, desc: 'API 版本，决定该资源属于哪个 API 组', example: 'apps/v1' },
  { path: 'kind', required: true, desc: '资源类型', example: 'Deployment' },
  { path: 'metadata.name', required: true, desc: '资源名称，同一命名空间内唯一', example: 'my-app' },
  { path: 'metadata.namespace', required: false, desc: '所属命名空间；集群级资源无需填写', example: 'default' },
  { path: 'metadata.labels', required: false, desc: '标签，用于选择器匹配与分组筛选', example: 'app: my-app' },
  { path: 'metadata.annotations', required: false, desc: '注解，存放非标识性元数据', example: 'kubesphere.io/description: "说明"' },
]

const HELP = {
  deployment: [
    ...COMMON,
    { path: 'spec.replicas', required: false, desc: '期望的 Pod 副本数，默认 1', example: 'replicas: 3' },
    { path: 'spec.selector.matchLabels', required: true, desc: '副本选择器，必须与 template 的标签匹配', example: 'matchLabels:\n  app: my-app' },
    { path: 'spec.template', required: true, desc: 'Pod 模板，定义副本长什么样' },
    { path: 'spec.template.spec.containers[]', required: true, desc: '容器列表，至少一个' },
    { path: 'spec.template.spec.containers[].name', required: true, desc: '容器名，同一 Pod 内唯一', example: 'app' },
    { path: 'spec.template.spec.containers[].image', required: true, desc: '镜像地址，建议固定 tag 而非 latest', example: 'nginx:1.25-alpine' },
    { path: 'spec.template.spec.containers[].ports[]', required: false, desc: '容器暴露端口', example: 'containerPort: 80' },
    { path: 'spec.template.spec.containers[].env[]', required: false, desc: '环境变量', example: 'name/value 或 valueFrom' },
    { path: 'spec.template.spec.containers[].resources', required: false, desc: 'CPU/内存 requests 与 limits，影响调度与 QoS', example: 'requests: {cpu: 100m}' },
    { path: 'spec.strategy.type', required: false, desc: '更新策略：RollingUpdate（滚动）/ Recreate（先删后建）', example: 'RollingUpdate' },
    { path: 'spec.strategy.rollingUpdate.maxSurge', required: false, desc: '滚动更新时允许超出副本数的最大值，可为数字或百分比', example: '25%' },
    { path: 'spec.strategy.rollingUpdate.maxUnavailable', required: false, desc: '滚动更新时允许不可用的最大数量', example: '25%' },
    { path: 'spec.template.spec.nodeSelector', required: false, desc: '按节点标签调度到指定节点', example: 'kubernetes.io/os: linux' },
    { path: 'spec.template.spec.tolerations[]', required: false, desc: '容忍节点污点，使 Pod 可调度到带污点的节点' },
  ],
  statefulset: [
    ...COMMON,
    { path: 'spec.serviceName', required: true, desc: '管理该 StatefulSet 的 Headless Service 名称', example: 'my-app' },
    { path: 'spec.replicas', required: false, desc: '期望副本数；Pod 名为 <名称>-0、-1 有序排列', example: '3' },
    { path: 'spec.selector.matchLabels', required: true, desc: '副本选择器，须与 template 标签匹配' },
    { path: 'spec.template.spec.containers[]', required: true, desc: '容器列表，配置方式同 Deployment' },
    { path: 'spec.volumeClaimTemplates[]', required: false, desc: '每个 Pod 独立的存储卷模板，Pod 删除后 PVC 保留', example: 'storage: 1Gi' },
    { path: 'spec.podManagementPolicy', required: false, desc: 'OrderedReady（默认，按序启停）/ Parallel（并行）', example: 'OrderedReady' },
    { path: 'spec.updateStrategy.type', required: false, desc: '更新策略：RollingUpdate / OnDelete', example: 'RollingUpdate' },
  ],
  service: [
    ...COMMON,
    { path: 'spec.type', required: false, desc: '服务类型：ClusterIP（集群内）/ NodePort（节点端口）/ LoadBalancer / ExternalName', example: 'ClusterIP' },
    { path: 'spec.selector', required: false, desc: '后端 Pod 的标签选择器；不填则需手工管理 Endpoints', example: 'app: my-app' },
    { path: 'spec.ports[]', required: true, desc: '端口映射列表' },
    { path: 'spec.ports[].port', required: true, desc: 'Service 暴露的端口', example: '80' },
    { path: 'spec.ports[].targetPort', required: false, desc: '后端容器端口，默认与 port 相同', example: '8080' },
    { path: 'spec.ports[].protocol', required: false, desc: '协议：TCP（默认）/ UDP / SCTP', example: 'TCP' },
    { path: 'spec.ports[].nodePort', required: false, desc: 'NodePort 类型时对外端口，范围 30000-32767，留空自动分配' },
    { path: 'spec.sessionAffinity', required: false, desc: '会话保持：None（默认）/ ClientIP', example: 'None' },
  ],
  configmap: [
    ...COMMON,
    { path: 'data', required: false, desc: '键值对数据，值为字符串', example: 'config.yaml: |\n  key: value' },
    { path: 'binaryData', required: false, desc: '二进制数据，值为 base64 编码', example: 'logo.png: iVBORw0KG...' },
    { path: 'immutable', required: false, desc: '设为 true 后不可再修改，只能删除重建', example: 'false' },
  ],
  secret: [
    ...COMMON,
    { path: 'type', required: false, desc: '密钥类型：Opaque（默认）/ kubernetes.io/dockerconfigjson（镜像拉取）等', example: 'Opaque' },
    { path: 'data', required: false, desc: '数据，值必须是 base64 编码', example: 'password: cGFzc3dvcmQ=' },
    { path: 'stringData', required: false, desc: '写入时用明文，K8s 会自动转成 data 的 base64', example: 'password: mypass' },
    { path: 'immutable', required: false, desc: '设为 true 后不可修改', example: 'false' },
  ],
  ingress: [
    ...COMMON,
    { path: 'spec.ingressClassName', required: false, desc: '指定 Ingress 控制器，如 nginx', example: 'nginx' },
    { path: 'spec.rules[]', required: false, desc: '路由规则列表：按 host 与 path 转发到后端 Service' },
    { path: 'spec.rules[].host', required: false, desc: '域名，留空则匹配所有 Host', example: 'app.example.com' },
    { path: 'spec.rules[].http.paths[].path', required: true, desc: 'URL 路径', example: '/api' },
    { path: 'spec.rules[].http.paths[].pathType', required: true, desc: '路径匹配方式：Prefix（前缀）/ Exact（精确）/ ImplementationSpecific', example: 'Prefix' },
    { path: 'spec.rules[].http.paths[].backend.service', required: true, desc: '后端 Service 名称与端口', example: 'name: my-svc, port.number: 80' },
    { path: 'spec.tls[]', required: false, desc: 'TLS 配置：hosts 与 secretName（证书密钥名）' },
  ],
  pvc: [
    ...COMMON,
    { path: 'spec.accessModes[]', required: true, desc: '访问模式：ReadWriteOnce / ReadOnlyMany / ReadWriteMany / ReadWriteOncePod', example: '[ReadWriteOnce]' },
    { path: 'spec.resources.requests.storage', required: true, desc: '申请的存储容量', example: '10Gi' },
    { path: 'spec.storageClassName', required: false, desc: '存储类名称，留空用默认存储类', example: 'standard' },
    { path: 'spec.volumeMode', required: false, desc: '卷模式：Filesystem（默认）/ Block', example: 'Filesystem' },
    { path: 'spec.selector', required: false, desc: '按标签选择已存在的 PV（绑定约束）' },
  ],
  job: [
    ...COMMON,
    { path: 'spec.template', required: true, desc: 'Pod 模板，与 Deployment 结构相同' },
    { path: 'spec.completions', required: false, desc: '需要成功完成的 Pod 总数，默认 1', example: '1' },
    { path: 'spec.parallelism', required: false, desc: '并发运行的 Pod 数，默认 1', example: '2' },
    { path: 'spec.backoffLimit', required: false, desc: '失败重试次数上限，默认 6', example: '4' },
    { path: 'spec.activeDeadlineSeconds', required: false, desc: '任务超时时间（秒），超时即终止', example: '600' },
    { path: 'spec.ttlSecondsAfterFinished', required: false, desc: '完成后自动清理的延迟秒数', example: '3600' },
  ],
  cronjob: [
    ...COMMON,
    { path: 'spec.schedule', required: true, desc: 'Cron 表达式（分钟 小时 日 月 周）', example: '"0 2 * * *"（每天 2 点）' },
    { path: 'spec.jobTemplate.spec.template', required: true, desc: '每次执行创建的 Job 的 Pod 模板' },
    { path: 'spec.concurrencyPolicy', required: false, desc: '并发策略：Allow（默认）/ Forbid / Replace', example: 'Forbid' },
    { path: 'spec.successfulJobsHistoryLimit', required: false, desc: '保留的成功任务记录数，默认 3', example: '3' },
    { path: 'spec.failedJobsHistoryLimit', required: false, desc: '保留的失败任务记录数，默认 1', example: '1' },
    { path: 'spec.suspend', required: false, desc: '暂停调度（true 时不触发）', example: 'false' },
    { path: 'spec.startingDeadlineSeconds', required: false, desc: '错过调度时间后的容忍秒数' },
  ],
  daemonset: [
    ...COMMON,
    { path: 'spec.selector.matchLabels', required: true, desc: '副本选择器，须与 template 标签匹配' },
    { path: 'spec.template', required: true, desc: 'Pod 模板；每个（匹配的）节点上运行一个 Pod' },
    { path: 'spec.updateStrategy.type', required: false, desc: '更新策略：RollingUpdate（默认）/ OnDelete', example: 'RollingUpdate' },
    { path: 'spec.template.spec.tolerations[]', required: false, desc: '常需容忍节点污点才能在全部节点上运行' },
  ],
  namespace: [
    { path: 'apiVersion', required: true, desc: '固定为 v1', example: 'v1' },
    { path: 'kind', required: true, desc: '资源类型', example: 'Namespace' },
    { path: 'metadata.name', required: true, desc: '命名空间名称', example: 'my-project' },
    { path: 'metadata.labels', required: false, desc: '标签，可用于项目分组筛选' },
    { path: 'metadata.annotations', required: false, desc: '注解' },
    { path: 'spec.finalizers[]', required: false, desc: '终结器，一般保持默认 kubernetes' },
  ],
  serviceaccount: [
    ...COMMON,
    { path: 'imagePullSecrets[]', required: false, desc: '拉取私有镜像时使用的密钥列表', example: 'name: my-registry-secret' },
    { path: 'automountServiceAccountToken', required: false, desc: '是否自动挂载 Token 到 Pod，默认 true', example: 'false' },
    { path: 'secrets[]', required: false, desc: '关联的 Secret 列表（K8s 1.24 后一般由系统管理）' },
  ],
  networkpolicy: [
    ...COMMON,
    { path: 'spec.podSelector', required: true, desc: '策略作用的 Pod 范围；空 {} 表示本命名空间全部 Pod', example: 'matchLabels:\n  app: my-app' },
    { path: 'spec.policyTypes[]', required: true, desc: '策略类型：Ingress（入站）/ Egress（出站）', example: '[Ingress, Egress]' },
    { path: 'spec.ingress[]', required: false, desc: '允许的入站规则：from（来源）+ ports（端口）' },
    { path: 'spec.egress[]', required: false, desc: '允许的出站规则：to（目的）+ ports（端口）' },
    { path: 'spec.ingress[].from[].podSelector', required: false, desc: '允许哪些 Pod 访问；空 {} 表示任意 Pod' },
    { path: 'spec.ingress[].ports[].port', required: false, desc: '允许访问的端口', example: '6379' },
  ],
  hpa: [
    { path: 'apiVersion', required: true, desc: '推荐 autoscaling/v2（支持多指标）', example: 'autoscaling/v2' },
    { path: 'kind', required: true, desc: '资源类型', example: 'HorizontalPodAutoscaler' },
    { path: 'metadata.name', required: true, desc: 'HPA 名称' },
    { path: 'spec.scaleTargetRef', required: true, desc: '扩缩容目标：apiVersion / kind / name', example: 'kind: Deployment, name: my-app' },
    { path: 'spec.minReplicas', required: false, desc: '最小副本数，默认 1', example: '2' },
    { path: 'spec.maxReplicas', required: true, desc: '最大副本数', example: '10' },
    { path: 'spec.metrics[]', required: false, desc: '扩缩容指标：Resource / Pods / Object / External', example: 'cpu 利用率 70%' },
    { path: 'spec.behavior', required: false, desc: '扩缩容行为控制：scaleUp / scaleDown 的稳定窗口与步长' },
  ],
}

// 通用兜底：未知类型给出最小必要字段说明
const FALLBACK = COMMON

export function fieldHelpFor(kind) {
  const k = String(kind || '').toLowerCase().replace(/s$/, '')
  return HELP[k] || HELP[kind] || FALLBACK
}
