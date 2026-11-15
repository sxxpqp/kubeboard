import assert from 'node:assert/strict'
import { projectPath, resourcePath, projectResources } from '../src/router/projectScope.js'

for (const resource of projectResources) {
  assert.equal(projectPath('/' + resource, 'kube-node-lease'), `/projects/kube-node-lease/${resource}`)
  assert.equal(projectPath(`/projects/old/${resource}`, 'default'), `/projects/default/${resource}`)
  assert.equal(projectPath(`/projects/old/${resource}`, ''), '/' + resource)
}
assert.equal(projectPath('/nodes', 'default'), '/nodes')
assert.equal(projectPath('/pvs', 'default'), '/pvs')
assert.equal(resourcePath('/projects/default/pvcs'), '/pvcs')
assert.equal(resourcePath('/workloads/deployment/default/nginx'), '/workloads/deployment/default/nginx')
console.log('项目路由转换检查通过')
