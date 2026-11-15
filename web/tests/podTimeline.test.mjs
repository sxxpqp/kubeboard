import assert from 'node:assert/strict'
import { podTimeline } from '../src/utils/podTimeline.js'
const rows = podTimeline({ creationTime: 10, containers: [
  { name: 'worker', startedAt: 20, finishedAt: 30, reason: 'Error', exitCode: 1,
    lastTerminated: { finishedAt: 15, reason: 'OOMKilled', exitCode: 137 } },
  { name: 'setup', finishedAt: 25, reason: 'Completed', exitCode: 0 },
] })
assert.deepEqual(rows.map(row => row.time), [30, 25, 20, 15, 10])
assert.equal(rows[0].title, '容器 worker 终止')
assert.equal(rows[0].message, 'Error · exit 1')
assert.equal(rows[0].type, 'warning')
assert.equal(rows[1].type, 'success')
assert.equal(rows[3].title, '容器 worker 上次终止')
assert.deepEqual(podTimeline({ containers: [{ finishedAt: 0 }] }), [])
console.log('Pod 终止时间线检查通过')
