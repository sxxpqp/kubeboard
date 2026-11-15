// 页面项目范围属于路由路径；API 请求仍使用 Kubernetes namespace 参数。
export const projectResources = new Set([
  'workloads', 'jobs', 'cronjobs', 'pods', 'services', 'ingresses',
  'hpas', 'networkpolicies', 'pvcs', 'configmaps', 'secrets', 'serviceaccounts', 'events',
])

export function resourcePath(path) {
  return path.replace(/^\/projects\/[^/]+/, '') || '/'
}

export function projectPath(path, namespace) {
  const base = resourcePath(path)
  return namespace && projectResources.has(base.slice(1))
    ? `/projects/${encodeURIComponent(namespace)}${base}`
    : base
}
