import { parseExtraYaml, buildExtraYaml, baseExtraForm } from '../src/utils/extraYaml.js'
let pass = 0, fail = 0
const eq=(n,g,w)=>{const ok=JSON.stringify(g)===JSON.stringify(w);if(ok){pass++;console.log('  ✅ '+n)}else{fail++;console.log('  ❌ '+n+' got='+JSON.stringify(g)+' want='+JSON.stringify(w))}}

console.log('\n=== Ingress ===')
{
  const f=baseExtraForm('ingress','default')
  f.name='web-ing'; f.ingressClassName='nginx'
  f.rules=[{host:'app.example.com',paths:[{path:'/api',pathType:'Prefix',serviceName:'api-svc',servicePort:8080}]}]
  f.tls=[{hosts:'app.example.com',secretName:'app-tls'}]
  const y=buildExtraYaml('ingress',f)
  const f2=parseExtraYaml('ingress',y)
  eq('name',f2.name,'web-ing')
  eq('className',f2.ingressClassName,'nginx')
  eq('host',f2.rules[0].host,'app.example.com')
  eq('path',f2.rules[0].paths[0].path,'/api')
  eq('pathType',f2.rules[0].paths[0].pathType,'Prefix')
  eq('svcName',f2.rules[0].paths[0].serviceName,'api-svc')
  eq('svcPort',f2.rules[0].paths[0].servicePort,8080)
  eq('tls',f2.tls[0].secretName,'app-tls')
}
{
  const real=`apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: hello-ingress
  namespace: default
spec:
  ingressClassName: nginx
  rules:
  - host: hello.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: hello-service
            port:
              number: 80
`
  const f=parseExtraYaml('ingress',real)
  eq('真实 Ingress name',f.name,'hello-ingress')
  eq('真实 Ingress svc',f.rules[0].paths[0].serviceName,'hello-service')
  eq('真实 Ingress host',f.rules[0].host,'hello.local')
}

console.log('\n=== Job ===')
{
  const f=baseExtraForm('job','default')
  f.name='batch-job'; f.completions=3; f.parallelism=2; f.backoffLimit=4
  f.ttlSecondsAfterFinished=600
  f.containers[0].image='busybox'; f.containers[0].command='/bin/sh -c'; f.containers[0].args='echo done'
  const y=buildExtraYaml('job',f)
  const f2=parseExtraYaml('job',y)
  eq('name',f2.name,'batch-job')
  eq('completions',f2.completions,3)
  eq('parallelism',f2.parallelism,2)
  eq('backoffLimit',f2.backoffLimit,4)
  eq('ttl',f2.ttlSecondsAfterFinished,600)
  eq('image',f2.containers[0].image,'busybox')
  eq('command',f2.containers[0].command,'/bin/sh -c')
  eq('args',f2.containers[0].args,'echo done')
  eq('restartPolicy',f2.restartPolicy,'OnFailure')
}

console.log('\n=== CronJob ===')
{
  const f=baseExtraForm('cronjob','default')
  f.name='nightly'; f.schedule='0 2 * * *'; f.concurrencyPolicy='Forbid'
  f.successfulJobsHistoryLimit=5; f.failedJobsHistoryLimit=2
  f.containers[0].image='alpine'; f.containers[0].command='echo run'
  const y=buildExtraYaml('cronjob',f)
  const f2=parseExtraYaml('cronjob',y)
  eq('name',f2.name,'nightly')
  eq('schedule',f2.schedule,'0 2 * * *')
  eq('concurrencyPolicy',f2.concurrencyPolicy,'Forbid')
  eq('successHistory',f2.successfulJobsHistoryLimit,5)
  eq('failHistory',f2.failedJobsHistoryLimit,2)
  eq('image',f2.containers[0].image,'alpine')
  eq('suspend',f2.suspend,false)
}

console.log('\n=== DaemonSet ===')
{
  const f=baseExtraForm('daemonset','default')
  f.name='log-agent'; f.updateStrategy='RollingUpdate'; f.maxUnavailable='25%'
  f.labels=[{key:'app',value:'log-agent'}]
  f.containers[0].image='fluentd:1.16'; f.containers[0].cpuReq='100m'; f.containers[0].cpuLim='500m'
  f.containers[0].ports=[{name:'metrics',containerPort:24231,protocol:'TCP'}]
  f.nodeSelector=[{key:'kubernetes.io/os',value:'linux'}]
  f.tolerations=[{key:'node-role.kubernetes.io/control-plane',value:''}]
  const y=buildExtraYaml('daemonset',f)
  const f2=parseExtraYaml('daemonset',y)
  eq('name',f2.name,'log-agent')
  eq('updateStrategy',f2.updateStrategy,'RollingUpdate')
  eq('maxUnavailable',f2.maxUnavailable,'25%')
  eq('image',f2.containers[0].image,'fluentd:1.16')
  eq('cpuReq',f2.containers[0].cpuReq,'100m')
  eq('cpuLim',f2.containers[0].cpuLim,'500m')
  eq('port',f2.containers[0].ports[0].containerPort,24231)
  eq('nodeSelector',f2.nodeSelector,[{key:'kubernetes.io/os',value:'linux'}])
  eq('tolerations',f2.tolerations,[{key:'node-role.kubernetes.io/control-plane',value:''}])
}

console.log('\n=== 边界 ===')
eq('ingress 解析非 Ingress', parseExtraYaml('ingress','kind: Service\nmetadata:\n  name: x'), null)
eq('job 解析非 Job', parseExtraYaml('job','kind: Deployment\nmetadata:\n  name: x'), null)
eq('build(ingress,null)', buildExtraYaml('ingress',null), '')
eq('build(job,null)', buildExtraYaml('job',null), '')
eq('build(cronjob,null)', buildExtraYaml('cronjob',null), '')
eq('build(daemonset,null)', buildExtraYaml('daemonset',null), '')

console.log('\n'+'='.repeat(46))
console.log(`结果: ${pass} 通过 / ${fail} 失败`)
process.exit(fail?1:0)
