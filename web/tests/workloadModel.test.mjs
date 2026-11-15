import assert from 'node:assert/strict'
import {initial,setKind,simplify,validate} from '../src/utils/workloadModel.js'
import {containerFields,probeFields,volumeFields} from '../src/utils/workloadFields.js'
import {formatShell,splitShell} from '../src/utils/workloadYaml.js'
import {compareRevisions,revisionTemplate} from '../src/utils/revisionComparison.js'
const make=()=>initial('default')
for(const kind of ['Deployment','StatefulSet','DaemonSet'])assert.deepEqual(validate(setKind(make(),kind)),[],kind)
assert.equal(setKind(make(),'DaemonSet').spec.replicas,undefined)
assert.equal(setKind(setKind(make(),'StatefulSet'),'Deployment').spec.serviceName,undefined)
const original=make();original.metadata.resourceVersion='123';original.status={readyReplicas:2};original.spec.template.spec.affinity={podAffinity:{preferredDuringSchedulingIgnoredDuringExecution:[]}}
const cleaned=simplify(original)
assert.equal(cleaned.metadata.resourceVersion,undefined)
assert.equal(original.metadata.resourceVersion,'123')
assert.equal(cleaned.status,undefined)
assert.deepEqual(cleaned.spec.template.spec.affinity,original.spec.template.spec.affinity)
const commandFields=containerFields['端口与命令'].filter(f=>['command','args'].includes(f.key))
assert.deepEqual(commandFields.map(f=>f.type),['argv','argv'])
for(const argv of [['/bin/sh','-c'],['echo hello; exec app --message "a b"'],['','line\nbreak',"a'b",'C:\\path']])assert.deepEqual(splitShell(formatShell(argv)),argv)
const faulty=make();faulty.spec.template.spec.containers[0].command='sh -c'
assert(validate(faulty).some(e=>e.path.endsWith('.command')))
const refs=make();refs.spec.template.spec.containers[0].env=[{name:'TOKEN',value:'x',valueFrom:{secretKeyRef:{name:'token',key:'value'}}}]
assert(validate(refs).some(e=>e.path.includes('.env.')))
const unknown=make();unknown.spec.template.spec.volumes=[{name:'iscsi',iscsi:{targetPortal:'10.0.0.1:3260',iqn:'iqn.test',lun:0}}]
assert.deepEqual(validate(unknown),[],'Unlisted legal YAML volume source must survive the form')
unknown.spec.template.spec.volumes[0].emptyDir={}
assert(validate(unknown).some(e=>e.path==='volumes[0]'))
const requests=make();requests.spec.template.spec.containers[0].resources.requests.cpu='800m'
assert(validate(requests).some(e=>e.path.endsWith('requests.cpu')))
const mounts=make();mounts.spec.template.spec.containers[0].volumeMounts=[{name:'missing',mountPath:'/data'}]
assert(validate(mounts).some(e=>e.path.endsWith('.volumeMounts[0].name')))
const probes=make();probes.spec.template.spec.containers[0].readinessProbe={exec:{command:['true']},tcpSocket:{port:80}}
assert(validate(probes).some(e=>e.path.endsWith('.readinessProbe')))
for(const schema of [probeFields,volumeFields]) {
 const variants=schema.filter(f=>f.optional && f.exclude)
 assert(variants.length>=3)
 for(const variant of variants)assert(variants.filter(v=>v!==variant).every(v=>variant.exclude.includes(v.key)))
}
const a={metadata:{labels:{app:'a','pod-template-hash':'x'}},spec:{containers:[{name:'a',image:'nginx:1.27.5',args:['a b']}]}}
const b=structuredClone(a);b.metadata.labels['pod-template-hash']='y';b.spec.containers[0].image='nginx:1.28.0'
assert.deepEqual(compareRevisions(revisionTemplate(a),revisionTemplate(b)),[{path:'spec.containers[0].image',before:'nginx:1.27.5',after:'nginx:1.28.0',type:'修改'}])
assert.equal(a.metadata.labels['pod-template-hash'],'x')
assert.equal(compareRevisions({}, {x:0})[0].type,'新增')
assert.equal(compareRevisions({x:false},{})[0].type,'删除')
assert.deepEqual(compareRevisions({x:{a:1,b:2}},{x:{b:2,a:1}}),[])
console.log('工作负载完整模型、互斥来源、命令数组与版本对比检查通过')

const multiline=["echo hello\nexec app --name 'a b'",'tab\tvalue']
const oneLine=formatShell(multiline)
assert.equal(oneLine.includes('\n'),false,'Multiline scripts must display in one physical input line')
assert.deepEqual(splitShell(oneLine),multiline)
