// Only controller-generated identity is excluded; user labels and configuration remain comparable.
export function revisionTemplate(template) {
  if (!template) return null
  const result=structuredClone(template)
  if(result.metadata?.labels) delete result.metadata.labels['pod-template-hash']
  return result
}
export function compareRevisions(before,after,path='') {
  if(Object.is(before,after))return []
  if(before && after && typeof before==='object' && typeof after==='object' && Array.isArray(before)===Array.isArray(after)) {
    const keys=[...new Set([...Object.keys(before),...Object.keys(after)])].sort()
    return keys.flatMap(key=>compareRevisions(before[key],after[key],Array.isArray(before)?`${path}[${key}]`:path?`${path}.${key}`:key))
  }
  return [{path:path||'配置',before,after,type:before===undefined?'新增':after===undefined?'删除':'修改'}]
}
