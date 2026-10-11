import { ref } from 'vue'
import http from '../api/http'

export function useNodeMetrics() {
  const items = ref([])
  const available = ref(null)
  const loading = ref(false)

  async function load() {
    loading.value = true
    try {
      const av = await http.get('/metrics/available')
      available.value = av.available
      if (av.available) {
        const data = await http.get('/metrics/nodes')
        items.value = data.items || []
      }
    } catch {
      available.value = false
    } finally {
      loading.value = false
    }
  }
  return { items, available, loading, load }
}

export function usePodMetrics() {
  const items = ref([])
  const available = ref(null)
  const loading = ref(false)

  async function load(namespace) {
    loading.value = true
    try {
      const av = await http.get('/metrics/available')
      available.value = av.available
      if (av.available) {
        const params = namespace ? { namespace } : {}
        const data = await http.get('/metrics/pods', { params })
        items.value = data.items || []
      }
    } catch {
      available.value = false
    } finally {
      loading.value = false
    }
  }
  return { items, available, loading, load }
}
