import assert from 'node:assert/strict'
import { copyWorkload, changeContainerImage } from '../src/utils/workloadActions.js'
import { splitShell, formatShell } from '../src/utils/workloadYaml.js'
const source={metadata:{name:'app',namespace:'default',uid:'old',ownerReferences:[{}],annotations:{custom:'preserve'}},status:{replicas:2},spec:{selector:{matchLabels:{app:'app'}},template:{metadata:{labels:{app:'app'}},spec:{containers:[{name:'web',image:'nginx:1.27.5'},{name:'sidecar',image:'busybox:1.37.0'}],affinity:{custom:true}}}}}
const changed=changeContainerImage(source,'web','nginx:1.28.0')
assert.equal(changed.spec.template.spec.containers[1].image,'busybox:1.37.0')
assert.equal(source.spec.template.spec.containers[0].image,'nginx:1.27.5')
assert.deepEqual(changed.spec.template.spec.affinity,source.spec.template.spec.affinity)
const copied=copyWorkload(source,'app-copy')
assert.equal(copied.metadata.uid,undefined)
assert.equal(copied.status,undefined)
assert.equal(copied.metadata.ownerReferences,undefined)
assert.equal(copied.metadata.annotations.custom,'preserve')
assert.equal(copied.spec.selector.matchLabels['kubeboard.io/instance'],'app-copy')
assert.equal(copied.spec.template.metadata.labels['kubeboard.io/instance'],'app-copy')
assert.throws(()=>copyWorkload(source,'app'))
for(const args of [['/bin/sh','-c'],['echo hi'],['','a b',"it's fine",'a"b','C:\\dir'],['echo','hello world']])assert.deepEqual(splitShell(formatShell(args)),args)
assert.throws(()=>splitShell("'unclosed"))
console.log('工作负载操作与命令往返检查通过')
const { mergeFormChanges } = await import('../src/utils/workloadActions.js')
const original={metadata:{resourceVersion:'123'},spec:{template:{spec:{affinity:{untouched:true},initContainers:[{name:'init',image:'busybox'}],containers:[{name:'app',image:'old',env:[{name:'TOKEN',valueFrom:{secretKeyRef:{name:'secret',key:'token'}}}],securityContext:{runAsNonRoot:true}}]}}}}
const baseline={spec:{template:{spec:{containers:[{name:'app',image:'old',env:[{name:'TOKEN',value:''}]}]}}}}
const edited=structuredClone(baseline);edited.spec.template.spec.containers[0].image='new'
const merged=mergeFormChanges(original,baseline,edited)
assert.equal(merged.metadata.resourceVersion,'123')
assert.deepEqual(merged.spec.template.spec.affinity,{untouched:true})
assert.equal(merged.spec.template.spec.initContainers.length,1)
assert.deepEqual(merged.spec.template.spec.containers[0].env,original.spec.template.spec.containers[0].env)
assert.equal(merged.spec.template.spec.containers[0].securityContext.runAsNonRoot,true)
assert.equal(merged.spec.template.spec.containers[0].image,'new')
const { podTimeline } = await import('../src/utils/podTimeline.js')
assert.deepEqual(podTimeline({creationTime:1,conditions:[{type:'Ready',status:'False',lastTransitionTime:4}]},[{reason:'Pulling',lastSeen:3},{reason:'Unknown',lastSeen:-100}]).map(r=>r.time),[4,3,1])
assert.equal(podTimeline({conditions:[{type:'Ready',status:'False',lastTransitionTime:4}]})[0].title,'Ready = False')

const initSource=structuredClone(source)
initSource.spec.template.spec.initContainers=[{name:'prepare',image:'busybox:1.37.0'}]
const initChanged=changeContainerImage(initSource,'prepare','busybox:1.36.1')
assert.equal(initChanged.spec.template.spec.initContainers[0].image,'busybox:1.36.1')
assert.deepEqual(initChanged.spec.template.spec.containers,source.spec.template.spec.containers)
