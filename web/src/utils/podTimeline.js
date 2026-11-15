export function podTimeline(detail, events = []) {
  const rows = []
  const add = (time, title, message='', type='primary') => { if(Number(time)>0) rows.push({ time:Number(time), title, message, type }) }
  add(detail.creationTime,'创建 Pod')
  for(const c of detail.conditions || []) add(c.lastTransitionTime,c.type+' = '+c.status,c.reason || c.message,c.status==='True'?'success':'warning')
  for(const c of detail.containers || []) { add(c.startedAt,'容器 '+c.name+' 启动','','success'); if(c.finishedAt)add(c.finishedAt,'容器 '+c.name+' 终止',`${c.reason || ''} · exit ${c.exitCode}`,c.exitCode === 0?'success':'warning'); if(c.lastTerminated)add(c.lastTerminated.finishedAt,'容器 '+c.name+' 上次终止',`${c.lastTerminated.reason} · exit ${c.lastTerminated.exitCode}`,'warning') }
  for(const e of events)add(e.lastSeen>0?e.lastSeen:e.firstSeen,e.reason,e.message,e.type==='Warning'?'warning':'primary')
  add(detail.deletionTime,'开始终止','','danger')
  return rows.sort((a,b)=>b.time-a.time)
}
