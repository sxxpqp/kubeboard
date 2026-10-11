import { parseFormYaml, buildFormYaml, baseForm } from '../src/utils/simpleYaml.js'
let pass = 0, fail = 0
const eq=(n,g,w)=>{const ok=JSON.stringify(g)===JSON.stringify(w);if(ok){pass++;console.log('  ✅ '+n)}else{fail++;console.log('  ❌ '+n+' got='+JSON.stringify(g)+' want='+JSON.stringify(w))}}

console.log('\n=== Service ===')
{
  const f = baseForm('service','default')
  f.name='my-svc'; f.type='NodePort'; f.selector=[{key:'app',value:'web'}]
  f.ports=[{name:'http',port:80,targetPort:8080,protocol:'TCP',nodePort:30080}]
  const y = buildFormYaml('service', f)
  const f2 = parseFormYaml('service', y)
  eq('name', f2.name, 'my-svc')
  eq('type', f2.type, 'NodePort')
  eq('selector', f2.selector, [{key:'app',value:'web'}])
  eq('port', f2.ports[0].port, 80)
  eq('targetPort', f2.ports[0].targetPort, 8080)
  eq('nodePort', f2.ports[0].nodePort, 30080)
}
{
  const real=`apiVersion: v1
kind: Service
metadata:
  name: hello-service
  namespace: default
  labels:
    app: hello
spec:
  type: ClusterIP
  selector:
    app: hello
  ports:
  - port: 80
    targetPort: 80
    protocol: TCP
    name: http
`
  const f=parseFormYaml('service',real)
  eq('真实清单 name', f.name, 'hello-service')
  eq('真实清单 ports', f.ports.length, 1)
  eq('真实清单 targetPort', f.ports[0].targetPort, 80)
}

console.log('\n=== ConfigMap ===')
{
  const f=baseForm('configmap','default')
  f.name='my-cm'
  f.data=[{key:'a.txt',value:'hello'},{key:'b.ini',value:'x=1'}]
  const y=buildFormYaml('configmap',f)
  const f2=parseFormYaml('configmap',y)
  eq('name',f2.name,'my-cm')
  eq('data 条数',f2.data.length,2)
  eq('data 键',f2.data.map(d=>d.key).sort(),['a.txt','b.ini'])
}
{
  const real=`apiVersion: v1
kind: ConfigMap
metadata:
  name: kube-root-ca.crt
  namespace: default
  annotations:
    kubernetes.io/description: Contains a CA bundle
data:
  ca.crt: |
    -----BEGIN CERTIFICATE-----
    MIIB
    -----END CERTIFICATE-----
`
  const f=parseFormYaml('configmap',real)
  eq('真实 CM name',f.name,'kube-root-ca.crt')
  eq('真实 CM 有 ca.crt',f.data.some(d=>d.key==='ca.crt'),true)
  eq('多行内容保留换行', f.data.find(d=>d.key==='ca.crt').value.includes('\n'), true)
}

console.log('\n=== Secret ===')
{
  const f=baseForm('secret','default')
  f.name='my-secret'; f.type='kubernetes.io/basic-auth'
  f.data=[{key:'username',value:'YWRtaW4='},{key:'password',value:'cGFzcw=='}]
  const y=buildFormYaml('secret',f)
  const f2=parseFormYaml('secret',y)
  eq('name',f2.name,'my-secret')
  eq('type',f2.type,'kubernetes.io/basic-auth')
  eq('data',f2.data.length,2)
}
{
  const real=`apiVersion: v1
kind: Secret
metadata:
  name: tls-secret
  namespace: default
type: kubernetes.io/tls
data:
  tls.crt: LS0tLS1CRUdJTi
  tls.key: LS0tLS1CRUdJTi
`
  const f=parseFormYaml('secret',real)
  eq('真实 Secret name',f.name,'tls-secret')
  eq('真实 Secret type',f.type,'kubernetes.io/tls')
  eq('真实 Secret 键',f.data.map(d=>d.key).sort(),['tls.crt','tls.key'])
}

console.log('\n=== 边界 ===')
eq('Service 解析非 Service', parseFormYaml('service','kind: ConfigMap\nmetadata:\n  name: x'), null)
eq('ConfigMap 解析非 CM', parseFormYaml('configmap','kind: Deployment\nmetadata:\n  name: x'), null)
eq('build(service,null)', buildFormYaml('service',null), '')
eq('build(configmap,null)', buildFormYaml('configmap',null), '')
eq('build(secret,null)', buildFormYaml('secret',null), '')

console.log('\n'+'='.repeat(46))
console.log(`结果: ${pass} 通过 / ${fail} 失败`)
process.exit(fail?1:0)
