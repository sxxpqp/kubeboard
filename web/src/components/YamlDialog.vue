<template>
  <el-dialog
    v-model="visible"
    :title="mode === 'create' ? '创建资源 (YAML)' : `编辑: ${name}`"
    width="760px"
    :close-on-click-modal="false"
    @closed="$emit('update:modelValue', false)"
  >
    <el-input
      v-model="yamlText"
      type="textarea"
      :rows="22"
      placeholder="在此粘贴或编写 Kubernetes YAML"
      class="yaml-editor"
      spellcheck="false"
    />
    <div v-if="errorMsg" class="yaml-error">
      <el-icon><WarningFilled /></el-icon> {{ errorMsg }}
    </div>
    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="saving" @click="save">
        {{ mode === 'create' ? '创建' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const props = defineProps({
  modelValue: Boolean,
  mode: { type: String, default: 'create' }, // create | edit
  kind: { type: String, required: true },    // deployment/service/configmap/...
  namespace: { type: String, default: '' },
  name: { type: String, default: '' },
  template: { type: String, default: '' },   // create 模式的初始 YAML
})
const emit = defineEmits(['update:modelValue', 'saved'])

const visible = ref(props.modelValue)
const yamlText = ref('')
const saving = ref(false)
const errorMsg = ref('')

watch(() => props.modelValue, async (v) => {
  visible.value = v
  errorMsg.value = ''
  if (v) {
    if (props.mode === 'edit') {
      try {
        const params = { kind: props.kind, name: props.name }
        if (props.namespace) params.namespace = props.namespace
        const resp = await fetch('/api/resource?' + new URLSearchParams(params), {
          headers: { Authorization: 'Bearer ' + localStorage.getItem('kubeboard_token') },
        })
        if (!resp.ok) throw new Error((await resp.json()).error || '加载失败')
        yamlText.value = await resp.text()
      } catch (e) {
        ElMessage.error(e.message)
        visible.value = false
        emit('update:modelValue', false)
      }
    } else {
      yamlText.value = props.template
    }
  }
})

watch(visible, (v) => { if (!v) emit('update:modelValue', false) })

async function save() {
  saving.value = true
  errorMsg.value = ''
  try {
    const url = props.mode === 'create' ? '/resources' : '/resources'
    await http({
      method: props.mode === 'create' ? 'post' : 'put',
      url,
      data: { kind: props.kind, yaml: yamlText.value },
    })
    ElMessage.success(props.mode === 'create' ? '创建成功' : '保存成功')
    visible.value = false
    emit('saved')
  } catch (e) {
    errorMsg.value = e.response?.data?.error || e.message
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.yaml-editor :deep(textarea) {
  font-family: 'SF Mono', Menlo, Monaco, Consolas, monospace;
  font-size: 12px;
  line-height: 1.6;
}
.yaml-error {
  margin-top: 8px;
  color: var(--ks-danger);
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 4px;
}
</style>
