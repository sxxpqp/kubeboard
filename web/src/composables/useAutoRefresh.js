import { onUnmounted, ref } from 'vue'

// 全局自动刷新设置（跨页面共享，持久化到 localStorage）
const enabled = ref(localStorage.getItem('kubeboard_autorefresh') === 'true')
const interval = ref(Number(localStorage.getItem('kubeboard_autorefresh_interval')) || 15000)

export function useAutoRefreshSettings() {
  function setEnabled(v) {
    enabled.value = v
    localStorage.setItem('kubeboard_autorefresh', String(v))
  }
  function setIntervalMs(ms) {
    interval.value = ms
    localStorage.setItem('kubeboard_autorefresh_interval', String(ms))
  }
  return { enabled, interval, setEnabled, setIntervalMs }
}

// 页面级自动刷新：组件挂载时按全局设置轮询，卸载时清理
export function useAutoRefresh(loadFn) {
  let timer = null

  function stop() {
    if (timer) { clearInterval(timer); timer = null }
  }

  function start() {
    stop()
    if (!enabled.value) return
    timer = setInterval(() => {
      // 避免页面隐藏时无谓请求
      if (document.visibilityState === 'visible') loadFn()
    }, interval.value)
  }

  function setEnabled(v) {
    enabled.value = v
    localStorage.setItem('kubeboard_autorefresh', String(v))
    v ? start() : stop()
  }

  onUnmounted(stop)

  return { start, stop, setEnabled }
}
