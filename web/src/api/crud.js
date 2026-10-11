// 资源 CRUD 组合式函数：各页面共享创建/编辑/删除逻辑
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import http from '../api/http'

export function useResourceCRUD(kind, reloadFn) {
  const dialogVisible = ref(false)
  const dialogMode = ref('create')
  const currentName = ref('')
  const currentNamespace = ref('')
  const currentTemplate = ref('')

  function openCreate(template = '') {
    dialogMode.value = 'create'
    currentTemplate.value = template
    dialogVisible.value = true
  }

  function openEdit(name, namespace = '') {
    dialogMode.value = 'edit'
    currentName.value = name
    currentNamespace.value = namespace
    dialogVisible.value = true
  }

  async function remove(name, namespace = '') {
    try {
      await ElMessageBox.confirm(
        `确定删除 ${name} 吗？此操作不可恢复。`,
        '删除确认',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
    } catch {
      return
    }
    try {
      const params = { kind, name }
      if (namespace) params.namespace = namespace
      await http.delete('/resources', { params })
      ElMessage.success(`已删除 ${name}`)
      reloadFn && reloadFn()
    } catch (e) {
      ElMessage.error(e.response?.data?.error || '删除失败')
    }
  }

  return {
    dialogVisible, dialogMode, currentName, currentNamespace, currentTemplate,
    openCreate, openEdit, remove,
  }
}

export function onDialogSaved(reloadFn) {
  return () => { ElMessage.success('操作成功'); reloadFn && reloadFn() }
}
