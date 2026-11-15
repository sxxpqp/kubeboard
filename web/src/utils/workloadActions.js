export function copyWorkload(source, name) {
  const object = structuredClone(source)
  if (!/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(name) || name.length > 63) throw new Error('新名称需为小写字母、数字或连字符，最长 63 位')
  if (name === source.metadata.name) throw new Error('请使用不同的新名称')
  delete object.status
  for (const key of ['uid', 'resourceVersion', 'generation', 'creationTimestamp', 'managedFields', 'selfLink', 'ownerReferences', 'deletionTimestamp', 'deletionGracePeriodSeconds', 'finalizers', 'generateName']) delete object.metadata[key]
  object.metadata.name = name
  if (object.metadata.annotations) delete object.metadata.annotations['kubectl.kubernetes.io/last-applied-configuration']
  object.spec.selector.matchLabels ||= {}
  object.spec.selector.matchLabels['kubeboard.io/instance'] = name
  object.spec.template.metadata.labels ||= {}
  object.spec.template.metadata.labels['kubeboard.io/instance'] = name
  return object
}
export function changeContainerImage(source, name, image) {
  const object = structuredClone(source)
  const pod = object.spec.template.spec
  const container = [...pod.containers,...(pod.initContainers || [])].find(c => c.name === name)
  if (!container) throw new Error('目标容器不存在，请刷新后重试')
  if (!image.trim()) throw new Error('镜像不能为空')
  container.image = image.trim()
  return object
}

// Apply only fields actually changed in the form, retaining fields the form cannot represent.
export function mergeFormChanges(original, baseline, edited) {
  const clone = value => value === undefined ? undefined : JSON.parse(JSON.stringify(value))
  function merge(target, before, after) {
    if (JSON.stringify(before) === JSON.stringify(after)) return clone(target)
    if (after === undefined) return undefined
    if (Array.isArray(after)) {
      if (!Array.isArray(before)) return clone(after)
      const identity = item => item && typeof item === 'object' ? item.name ?? item.key : undefined
      return after.map((item, index) => {
        const id = identity(item)
        const prev = id === undefined ? before[index] : before.find(entry => identity(entry) === id)
        const old = id === undefined ? target?.[index] : target?.find(entry => identity(entry) === id)
        return merge(old, prev, item)
      })
    }
    if (after && typeof after === 'object') {
      const result = clone(target) || {}
      for (const key of new Set([...Object.keys(before || {}), ...Object.keys(after)])) {
        if (JSON.stringify(before?.[key]) === JSON.stringify(after[key])) continue
        const value = merge(target?.[key], before?.[key], after[key])
        if (value === undefined) delete result[key]
        else result[key] = value
      }
      if ('value' in after && before?.value !== after.value) delete result.valueFrom
      return result
    }
    return clone(after)
  }
  return merge(original, baseline, edited)
}
