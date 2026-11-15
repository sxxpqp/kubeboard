<template>
  <div style="display: flex; height: 100vh">
    <aside class="ks-sidebar">
      <div class="ks-logo">
        <span class="logo-icon">K</span>
        Kubeboard
      </div>

      <nav class="ks-menu">
        <template v-for="group in menus" :key="group.title">
          <div class="ks-menu-group-title" @click="toggle(group.title)">
            <span>{{ group.title }}</span>
            <el-icon class="group-arrow" :class="{ open: !collapsed[group.title] }"><ArrowRight /></el-icon>
          </div>
          <div v-show="!collapsed[group.title]">
            <div
              v-for="item in group.items"
              :key="item.path"
              class="ks-menu-item"
              :class="{ active: resourcePath(route.path) === item.path || resourcePath(route.path).startsWith(item.path + '/') }"
              @click="navigate(item.path)"
            >
              <el-icon><component :is="item.icon" /></el-icon>
              {{ item.label }}
            </div>
          </div>
        </template>
      </nav>
      <div class="ks-sidebar-footer">
        <span class="cluster-dot"></span>
        <span>{{ clusterVersion }}</span>
      </div>
    </aside>
    <div style="flex: 1; display: flex; flex-direction: column; overflow: hidden">
      <header class="ks-header">
        <span class="cluster-context">当前集群</span>
        <el-select class="header-namespace" :model-value="currentNamespace" placeholder="全部命名空间" clearable filterable aria-label="切换命名空间" @change="changeScope"><el-option v-for="ns in scopeNamespaces" :key="ns" :label="ns" :value="ns" /></el-select>
        <div class="crumb"><span class="muted">集群管理</span><span class="crumb-divider">/</span>{{ route.meta.title || 'Kubeboard' }}</div>
        <el-tooltip :content="autoRefresh.enabled.value ? '自动刷新已开启' : '自动刷新已关闭'" placement="bottom">
          <el-switch
            v-model="autoRefreshSwitch"
            size="small"
            inline-prompt
            active-text="自动"
            inactive-text="手动"
            style="margin-right: 18px"
            @change="onAutoRefreshChange"
          />
        </el-tooltip>
        <el-dropdown @command="onCommand">
          <span style="cursor: pointer; display: flex; align-items: center; gap: 6px">
            <el-avatar :size="28" style="background: var(--ks-primary)">{{ avatarChar }}</el-avatar>
            <span>{{ username }}</span>
          </span>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="logout">退出登录</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </header>
      <main class="ks-main">
        <div v-if="apiMismatch" class="api-warn">
          ⚠️ 前后端版本不匹配（期望 API {{ expectedApiVersion }}，实际 {{ actualApiVersion }}）——后端可能是旧二进制，新功能将 404。请重启/重编译后端后刷新。
        </div>
        <div v-if="!route.params.name && !route.meta.editor" class="ks-page-heading"><div><div class="ks-eyebrow">KUBERNETES CONSOLE</div><h1>{{ route.meta.title }}</h1><p>查看资源运行状态，管理集群中的应用与服务</p></div><span class="ks-context">{{ currentNamespace || '全部项目' }}</span></div>
        <router-view :key="route.path" />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import { projectPath, resourcePath } from '../router/projectScope'
import { useAutoRefreshSettings } from '../composables/useAutoRefresh'

const route = useRoute()
const router = useRouter()
const clusterVersion = ref('')
const scopeNamespaces = ref([])
const clusterPaths = ['/overview', '/nodes', '/namespaces', '/pvs', '/storageclasses']
const currentNamespace = computed(() => String(route.params.namespace === '_' ? '' : route.params.namespace || ''))
function navigate(path) {
  router.push(projectPath(path, currentNamespace.value))
}
function changeScope(value) {
  const path = route.params.name || clusterPaths.includes(route.path) ? '/workloads' : resourcePath(route.path)
  router.push(projectPath(path, value))
}

const autoRefresh = useAutoRefreshSettings()
const autoRefreshSwitch = ref(autoRefresh.enabled.value)

function onAutoRefreshChange(v) {
  autoRefresh.setEnabled(v)
}

const collapsed = reactive({
  集群管理: false,
  应用负载: false,
  存储管理: false,
  配置与密钥: false,
})

const menus = [
  {
    title: '集群管理',
    items: [
      { path: '/overview', label: '集群概览', icon: 'Odometer' },
      { path: '/nodes', label: '节点', icon: 'Cpu' },
      { path: '/namespaces', label: '项目', icon: 'FolderOpened' },
      { path: '/events', label: '事件', icon: 'Bell' },

    ],
  },
  {
    title: '应用负载',
    items: [
      { path: '/workloads', label: '工作负载', icon: 'Box' },
      { path: '/jobs', label: '任务', icon: 'Clock' },
      { path: '/cronjobs', label: '定时任务', icon: 'Timer' },
      { path: '/pods', label: '容器组', icon: 'Grid' },


      { path: '/hpas', label: '弹性伸缩', icon: 'TrendCharts' },
    ],
  },
  { title: '网络', items: [
    { path: '/services', label: '服务', icon: 'Connection' },
    { path: '/ingresses', label: '应用路由', icon: 'Guide' },
    { path: '/networkpolicies', label: '网络策略', icon: 'Share' },
  ] },
  {
    title: '存储管理',
    items: [
      { path: '/pvcs', label: '持久卷声明', icon: 'Coin' },
      { path: '/pvs', label: '持久卷', icon: 'Files' },
      { path: '/storageclasses', label: '存储类', icon: 'Box' },
    ],
  },
  {
    title: '配置与密钥',
    items: [
      { path: '/configmaps', label: '配置字典', icon: 'Document' },
      { path: '/secrets', label: '保密字典', icon: 'Lock' },
      { path: '/serviceaccounts', label: '服务账号', icon: 'UserFilled' },
    ],
  },
]

function toggle(title) {
  collapsed[title] = !collapsed[title]
}

const username = computed(() => localStorage.getItem('kubeboard_username') || 'admin')
const avatarChar = computed(() => username.value.charAt(0).toUpperCase())

// P0-⑤：API 契约版本比对（旧后端 + 新前端 → 横幅告警而非静默 404）
// 注意：后端响应缺失 apiVersion 字段 = 后端是未注入版本的更旧二进制，同样视为不匹配。
const expectedApiVersion = typeof __EXPECTED_API_VERSION__ !== 'undefined' ? __EXPECTED_API_VERSION__ : 'v2'
const actualApiVersion = ref('')
const apiMismatch = computed(() => actualApiVersion.value !== expectedApiVersion)

function onCommand(cmd) {
  if (cmd === 'logout') {
    localStorage.removeItem('kubeboard_token')
    localStorage.removeItem('kubeboard_username')
    router.push('/login')
  }
}

onMounted(async () => {
  http.get('/namespaces').then(data => { scopeNamespaces.value = (data.items || []).map(ns => ns.name) }).catch(() => {})
  try {
    const v = await http.get('/version')
    clusterVersion.value = v.version || ''
    actualApiVersion.value = v.apiVersion || ''
  } catch { /* ignore */ }
})
</script>

<style scoped>
.cluster-context { margin-right: 16px; white-space: nowrap; font-size: 13px; }
.header-namespace { width: 200px; margin-right: 24px; }

.api-warn {
  background: var(--ks-danger, #f56c6c);
  color: #fff;
  padding: 8px 16px;
  font-size: 13px;
  border-radius: 4px;
  margin-bottom: 12px;
}
.ks-menu-group-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: pointer;
  user-select: none;
}
.ks-menu-group-title:hover { color: #9aa8bd; }
.group-arrow { transition: transform 0.2s; font-size: 12px; }
.group-arrow.open { transform: rotate(90deg); }
.ks-sidebar-footer {
  padding: 12px 18px;
  font-size: 11px;
  color: #6c7a90;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  display: flex;
  align-items: center;
  gap: 8px;
}
.cluster-dot { width: 7px; height: 7px; border-radius: 50%; background: var(--ks-success); }
</style>
