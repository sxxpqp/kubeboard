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
              :class="{ active: route.path === item.path }"
              @click="router.push(item.path)"
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
        <div class="crumb">{{ route.meta.title || 'Kubeboard' }}</div>
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
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import http from '../api/http'
import { useAutoRefreshSettings } from '../composables/useAutoRefresh'

const route = useRoute()
const router = useRouter()
const clusterVersion = ref('')
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
      { path: '/networkpolicies', label: '网络策略', icon: 'Share' },
    ],
  },
  {
    title: '应用负载',
    items: [
      { path: '/workloads', label: '工作负载', icon: 'Box' },
      { path: '/jobs', label: '任务', icon: 'Clock' },
      { path: '/cronjobs', label: '定时任务', icon: 'Timer' },
      { path: '/pods', label: '容器组', icon: 'Grid' },
      { path: '/services', label: '服务', icon: 'Connection' },
      { path: '/ingresses', label: '应用路由', icon: 'Guide' },
      { path: '/hpas', label: '弹性伸缩', icon: 'TrendCharts' },
    ],
  },
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

function onCommand(cmd) {
  if (cmd === 'logout') {
    localStorage.removeItem('kubeboard_token')
    localStorage.removeItem('kubeboard_username')
    router.push('/login')
  }
}

onMounted(async () => {
  try {
    const v = await http.get('/version')
    clusterVersion.value = v.version || ''
  } catch { /* ignore */ }
})
</script>

<style scoped>
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
