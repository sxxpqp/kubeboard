// I9 双向转换严格测试：解析/生成/往返/边界
import { parseWorkloadYaml, buildWorkloadYaml, baseWorkloadForm, emptyCtr, emptyVol, kvToObj, objToKv } from '../src/utils/workloadYaml.js'
import { fieldHelpFor } from '../src/utils/fieldHelp.js'

let pass = 0, fail = 0
const failures = []
function ok(name, cond, extra = '') {
  if (cond) { pass++; console.log('  ✅ ' + name) }
  else { fail++; failures.push(name + (extra ? '  ' + extra : '')); console.log('  ❌ ' + name + (extra ? '  ' + extra : '')) }
}
function eq(name, got, want) { ok(name, JSON.stringify(got) === JSON.stringify(want), `got=${JSON.stringify(got)} want=${JSON.stringify(want)}`) }

console.log('\n=== 1. 解析：kubectl 风格（序列与 key 同缩进）===')
{
  const y = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: d1
  namespace: ns1
spec:
  replicas: 5
  selector:
    matchLabels:
      app: d1
  template:
    metadata:
      labels:
        app: d1
    spec:
      containers:
      - name: c1
        image: nginx:1.25
        env:
        - name: A
          value: "1"
        ports:
        - containerPort: 8080
          protocol: TCP
`
  const f = parseWorkloadYaml(y)
  ok('解析成功', !!f)
  eq('kind', f.kind, 'deployment')
  eq('name', f.name, 'd1')
  eq('namespace', f.namespace, 'ns1')
  eq('replicas', f.replicas, 5)
  eq('标签回退 selector.matchLabels', f.labels, [{ key: 'app', value: 'd1' }])
  eq('容器数', f.containers.length, 1)
  eq('镜像', f.containers[0].image, 'nginx:1.25')
  eq('端口', f.containers[0].ports, [{ name: '', containerPort: 8080, protocol: 'TCP' }])
  eq('环境变量', f.containers[0].env, [{ key: 'A', value: '1' }])
}

console.log('\n=== 2. 解析：列表缩进 2 格风格 ===')
{
  const y = `kind: Deployment
metadata:
  name: d2
spec:
  replicas: 1
  template:
    spec:
      containers:
        - name: c1
          image: alpine
`
  const f = parseWorkloadYaml(y)
  eq('name', f.name, 'd2')
  eq('镜像', f.containers[0].image, 'alpine')
}

console.log('\n=== 3. 解析：StatefulSet + volumeClaimTemplates ===')
{
  const y = `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: ss1
  namespace: default
spec:
  serviceName: ss1-headless
  replicas: 3
  template:
    spec:
      containers:
      - name: app
        image: redis:7
        volumeMounts:
        - name: data
          mountPath: /data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: [ReadWriteOnce]
      resources:
        requests:
          storage: 10Gi
      storageClassName: standard
`
  const f = parseWorkloadYaml(y)
  eq('kind', f.kind, 'statefulset')
  eq('serviceName', f.serviceName, 'ss1-headless')
  eq('storageSize', f.storageSize, '10Gi')
  eq('storageClass', f.storageClass, 'standard')
  eq('挂载路径', f.volumes[0]?.mountPath, '/data')
}

console.log('\n=== 4. 解析：多容器 / 多卷 / 探针 / 资源 ===')
{
  const y = `kind: Deployment
metadata:
  name: multi
  namespace: default
spec:
  replicas: 2
  template:
    spec:
      imagePullSecrets:
      - name: regcred
      serviceAccountName: mysa
      restartPolicy: OnFailure
      nodeSelector:
        disktype: ssd
      tolerations:
      - key: dedicated
        operator: Equal
        value: gpu
        effect: NoSchedule
      containers:
      - name: a
        image: app-a:1
        command: ["/bin/sh", "-c"]
        args: ["echo hi"]
        resources:
          requests: {cpu: 100m, memory: 64Mi}
          limits: {cpu: 500m, memory: 256Mi}
        livenessProbe:
          httpGet: {path: /live, port: 8080}
          initialDelaySeconds: 10
          periodSeconds: 20
          timeoutSeconds: 3
          failureThreshold: 5
      - name: b
        image: app-b:2
        readinessProbe:
          tcpSocket: {port: 9000}
      volumes:
      - name: v1
        persistentVolumeClaim: {claimName: my-pvc}
      - name: v2
        secret: {secretName: my-secret}
`
  const f = parseWorkloadYaml(y)
  eq('容器数=2', f.containers.length, 2)
  eq('镜像 a', f.containers[0].image, 'app-a:1')
  eq('镜像 b', f.containers[1].image, 'app-b:2')
  eq('command', f.containers[0].command, '/bin/sh -c')
  eq('args', f.containers[0].args, 'echo hi')
  eq('cpuReq', f.containers[0].cpuReq, '100m')
  eq('memLim', f.containers[0].memLim, '256Mi')
  eq('liveness 启用', f.containers[0].probes.liveness.enabled, true)
  eq('liveness 路径', f.containers[0].probes.liveness.path, '/live')
  eq('liveness 延迟', f.containers[0].probes.liveness.initialDelaySeconds, 10)
  eq('liveness 阈值', f.containers[0].probes.liveness.failureThreshold, 5)
  eq('readiness TCP 端口', f.containers[1].probes.readiness.port, 9000)
  eq('imagePullSecrets', f.imagePullSecrets, 'regcred')
  eq('serviceAccount', f.serviceAccount, 'mysa')
  eq('restartPolicy', f.restartPolicy, 'OnFailure')
  eq('nodeSelector', f.nodeSelector, [{ key: 'disktype', value: 'ssd' }])
  eq('tolerations', f.tolerations, [{ key: 'dedicated', value: 'gpu' }])
  eq('卷数=2', f.volumes.length, 2)
  eq('卷1 类型', f.volumes[0].type, 'pvc')
  eq('卷1 来源', f.volumes[0].source, 'my-pvc')
  eq('卷2 类型', f.volumes[1].type, 'secret')
  eq('卷2 来源', f.volumes[1].source, 'my-secret')
}

console.log('\n=== 5. 生成：空表单不崩溃 ===')
{
  const f = baseWorkloadForm('deployment', 'default')
  f.name = 'x'
  const out = buildWorkloadYaml(f)
  ok('生成含 kind', out.includes('kind: Deployment'))
  ok('生成含 replicas', out.includes('replicas: 2'))
}

console.log('\n=== 6. 往返一致性：生成 -> 解析 -> 生成 ===')
{
  const f = baseWorkloadForm('deployment', 'default')
  f.name = 'rt'
  f.replicas = 3
  f.labels = [{ key: 'app', value: 'rt' }]
  f.containers = [emptyCtr()]
  f.containers[0].name = 'web'
  f.containers[0].image = 'nginx:1.27'
  f.containers[0].env = [{ key: 'MODE', value: 'prod' }]
  f.containers[0].cpuLim = '1'
  f.containers[0].probes.readiness.enabled = true
  f.containers[0].probes.readiness.path = '/ready'
  f.volumes = [emptyVol()]
  f.volumes[0].type = 'configMap'
  f.volumes[0].source = 'cfg'
  f.volumes[0].mountPath = '/etc/cfg'
  const y1 = buildWorkloadYaml(f)
  const f2 = parseWorkloadYaml(y1)
  const y2 = buildWorkloadYaml(f2)
  ok('往返后镜像一致', y2.includes('nginx:1.27'))
  ok('往返后副本一致', f2.replicas === 3)
  ok('往返后卷一致', f2.volumes[0]?.source === 'cfg')
  ok('往返后探针一致', f2.containers[0].probes.readiness.path === '/ready')
}

console.log('\n=== 7. 边界：空/非法/非工作负载输入 ===')
{
  eq('空串 -> null', parseWorkloadYaml(''), null)
  eq('纯空白 -> null', parseWorkloadYaml('   \n  '), null)
  eq('非法 YAML -> null', parseWorkloadYaml('{{{ not yaml'), null)
  eq('Service 不是工作负载 -> null', parseWorkloadYaml('kind: Service\nmetadata:\n  name: s'), null)
  eq('ConfigMap -> null', parseWorkloadYaml('kind: ConfigMap\nmetadata:\n  name: c'), null)
}

console.log('\n=== 8. 边界：含块标量的清单（| 多行）===')
{
  const y = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: blob
  annotations:
    note: |
      line1
      line2
spec:
  replicas: 1
  template:
    spec:
      containers:
      - name: c
        image: busybox
`
  const f = parseWorkloadYaml(y)
  ok('含块标量仍能解析', !!f && f.name === 'blob')
  ok('块标量内容保留', typeof f.annotations.find(a => a.key === 'note')?.value === 'string')
}

console.log('\n=== 9. 边界：特殊字符 / 数字 / 布尔 ===')
{
  const y = `kind: Deployment
metadata:
  name: tricky
spec:
  replicas: 0
  template:
    spec:
      containers:
      - name: c
        image: nginx
        env:
        - name: BOOL
          value: "true"
        - name: NUM
          value: "8080"
        - name: JSON
          value: '{"a":1}'
`
  const f = parseWorkloadYaml(y)
  eq('replicas=0 正确解析', f.replicas, 0)
  eq('布尔字符串保留', f.containers[0].env[0].value, 'true')
  eq('数字字符串保留', f.containers[0].env[1].value, '8080')
  eq('JSON 字符串保留', f.containers[0].env[2].value, '{"a":1}')
}

console.log('\n=== 10. 字段说明覆盖度 ===')
{
  const kinds = ['deployment','statefulset','service','configmap','secret','ingress','pvc','job','cronjob','daemonset','namespace','serviceaccount','networkpolicy','hpa']
  let covered = 0
  for (const k of kinds) {
    const h = fieldHelpFor(k)
    if (h && h.length >= 4) covered++
    else failures.push('字段说明不足: ' + k)
  }
  ok(`14 类资源字段说明齐全 (${covered}/14)`, covered === 14)
  const dep = fieldHelpFor('deployment')
  ok('deployment 说明含 replicas', dep.some(x => x.path === 'spec.replicas'))
  ok('deployment 说明含 image', dep.some(x => x.path.includes('image')))
  ok('必填字段有标记', dep.some(x => x.required === true))
  ok('未知类型有兜底', fieldHelpFor('unknownkind').length > 0)
}

console.log('\n=== 11. 工具函数 ===')
{
  eq('kvToObj', kvToObj([{ key: 'a', value: '1' }, { key: 'b', value: '2' }]), { a: '1', b: '2' })
  eq('kvToObj 忽略空 key', kvToObj([{ key: '', value: 'x' }]), {})
  eq('objToKv', objToKv({ a: '1' }), [{ key: 'a', value: '1' }])
  eq('objToKv 空', objToKv(null), [])
}

console.log('\n' + '='.repeat(50))
console.log(`结果: ${pass} 通过 / ${fail} 失败`)
if (failures.length) { console.log('\n失败项:'); failures.forEach(f => console.log('  - ' + f)) }
process.exit(fail ? 1 : 0)
