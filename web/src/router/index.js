import { createRouter, createWebHistory } from 'vue-router'
import { projectPath, projectResources } from './projectScope'

const routes = [
  { path: '/login', component: () => import('../views/Login.vue') },
  {
    path: '/',
    component: () => import('../components/Layout.vue'),
    children: [
      { path: '', redirect: '/overview' },
      // 集群管理
      { path: 'overview', component: () => import('../views/Overview.vue'), meta: { title: '集群概览' } },
      { path: 'nodes', component: () => import('../views/Nodes.vue'), meta: { title: '节点' } },
      { path: 'namespaces', component: () => import('../views/Namespaces.vue'), meta: { title: '项目' } },
      { path: 'namespaces/:namespace', component: () => import('../views/NamespaceDetail.vue'), meta: { title: '项目详情' } },
      { path: 'workloads/create', component: () => import('../views/WorkloadEditor.vue'), meta: { title: '创建工作负载', editor: true } },
      { path: 'projects/:namespace/workloads/create', component: () => import('../views/WorkloadEditor.vue'), meta: { title: '创建工作负载', editor: true } },
      { path: 'workloads/:kind/:namespace/:name/edit', component: () => import('../views/WorkloadEditor.vue'), meta: { title: '编辑工作负载', editor: true } },
      { path: 'workloads/:kind/:namespace/:name', component: () => import('../views/WorkloadDetail.vue'), meta: { title: '工作负载详情' } },
      { path: 'detail/:kind/:namespace/:name', component: () => import('../views/GenericDetail.vue'), meta: { title: '资源详情' } },
      { path: 'events', component: () => import('../views/Events.vue'), meta: { title: '事件' } },
      // 应用负载
      { path: 'workloads', component: () => import('../views/Workloads.vue'), meta: { title: '工作负载' } },
      { path: 'jobs', component: () => import('../views/Jobs.vue'), meta: { title: '任务' } },
      { path: 'cronjobs', component: () => import('../views/CronJobs.vue'), meta: { title: '定时任务' } },
      { path: 'pods', component: () => import('../views/Pods.vue'), meta: { title: '容器组' } },
      { path: 'services', component: () => import('../views/Services.vue'), meta: { title: '服务' } },
      { path: 'ingresses', component: () => import('../views/Ingresses.vue'), meta: { title: '应用路由' } },
      { path: 'hpas', component: () => import('../views/HPAs.vue'), meta: { title: '弹性伸缩' } },
      { path: 'networkpolicies', component: () => import('../views/NetworkPolicies.vue'), meta: { title: '网络策略' } },
      // 存储管理
      { path: 'pvcs', component: () => import('../views/PVCs.vue'), meta: { title: '持久卷声明' } },
      { path: 'pvs', component: () => import('../views/PVs.vue'), meta: { title: '持久卷' } },
      { path: 'storageclasses', component: () => import('../views/StorageClasses.vue'), meta: { title: '存储类' } },
      // 配置与密钥
      { path: 'configmaps', component: () => import('../views/ConfigMaps.vue'), meta: { title: '配置字典' } },
      { path: 'secrets', component: () => import('../views/Secrets.vue'), meta: { title: '保密字典' } },
      { path: 'serviceaccounts', component: () => import('../views/ServiceAccounts.vue'), meta: { title: '服务账号' } },
    ],
  },
]

// 复用相同页面组件，项目范围由路径参数提供。
const projectRoutes = routes[1].children.filter(route => projectResources.has(route.path))
routes[1].children.push(...projectRoutes.map(route => ({ ...route, path: `projects/:namespace/${route.path}` })))

const router = createRouter({ history: createWebHistory(), routes })

router.beforeEach((to) => {
  if (to.path !== '/login' && !localStorage.getItem('kubeboard_token')) {
    return '/login'
  }
  if (typeof to.query.namespace === 'string') {
    const path = projectPath(to.path, to.query.namespace)
    if (path !== to.path || to.params.namespace) {
      const { namespace, ...query } = to.query
      return { path, query, hash: to.hash, replace: true }
    }
  }
})

export default router
