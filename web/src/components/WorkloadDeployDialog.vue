<template>
  <el-dialog v-model="visible" :title="title" width="720px" top="6vh" @closed="reset">
    <el-form :model="form" label-width="120px" label-position="right">
      <el-divider content-position="left">基本信息</el-divider>
      <el-form-item label="类型">
        <el-radio-group v-model="form.kind">
          <el-radio-button value="deployment">Deployment</el-radio-button>
          <el-radio-button value="statefulset">StatefulSet</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="项目" required>
        <el-select v-model="form.namespace" filterable style="width: 240px">
          <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
        </el-select>
      </el-form-item>
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="my-app" style="width: 240px" />
      </el-form-item>

      <el-divider content-position="left">容器</el-divider>
      <div v-for="(ctr, idx) in form.containers" :key="idx" class="ctr-block">
        <el-form-item label="容器名" :required="idx === 0">
          <el-input v-model="ctr.name" placeholder="app" style="width: 200px" />
        </el-form-item>
        <el-form-item label="镜像" required>
          <el-input v-model="ctr.image" placeholder="nginx:1.25-alpine" style="width: 320px" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="ctr.port" :min="0" :max="65535" controls-position="right" />
          <span class="hint">0 表示不暴露端口</span>
        </el-form-item>
        <el-form-item label="CPU/内存请求">
          <el-input v-model="ctr.cpu" placeholder="100m" style="width: 110px" />
          <el-input v-model="ctr.memory" placeholder="128Mi" style="width: 110px; margin-left: 8px" />
        </el-form-item>
        <el-button v-if="form.containers.length > 1" link type="danger" size="small" @click="form.containers.splice(idx, 1)">移除该容器</el-button>
      </div>
      <el-button link type="primary" :icon="'Plus'" @click="form.containers.push(emptyCtr())">添加容器</el-button>

      <el-divider content-position="left">副本与更新</el-divider>
      <el-form-item label="副本数" v-if="form.kind === 'deployment'">
        <el-input-number v-model="form.replicas" :min="0" :max="100" controls-position="right" />
      </el-form-item>
      <el-form-item label="服务名(headless)" v-else>
        <el-input v-model="form.serviceName" placeholder="留空自动生成" style="width: 240px" />
      </el-form-item>
      <el-form-item label="持久卷声明">
        <el-input v-model="form.storage" placeholder="如 my-pvc（可选）" style="width: 240px" />
        <span class="hint">StatefulSet 会挂载到 {{ form.mountPath }}</span>
      </el-form-item>
      <el-form-item label="挂载路径">
        <el-input v-model="form.mountPath" placeholder="/data" style="width: 200px" />
      </el-form-item>
    </el-form>

    <el-collapse style="margin-top: 8px">
      <el-collapse-item title="预览生成的 YAML">
        <pre class="yaml-preview">{{ generatedYaml }}</pre>
      </el-collapse-item>
    </el-collapse>

    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" @click="submit">部署</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const props = defineProps({
  modelValue: Boolean,
  defaultKind: { type: String, default: 'deployment' },
  defaultNamespace: { type: String, default: 'default' },
  namespaces: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue', 'deployed'])

const visible = ref(props.modelValue)
watch(() => props.modelValue, (v) => (visible.value = v))
watch(visible, (v) => emit('update:modelValue', v))

const emptyCtr = () => ({ name: 'app', image: '', port: 80, cpu: '', memory: '' })

const form = ref({
  kind: props.defaultKind,
  namespace: props.defaultNamespace,
  name: '',
  replicas: 2,
  serviceName: '',
  storage: '',
  mountPath: '/data',
  containers: [emptyCtr()],
})
const saving = ref(false)

const title = computed(() => (form.value.kind === 'statefulset' ? '部署 StatefulSet' : '部署 Deployment'))

function reset() {
  form.value = {
    kind: props.defaultKind, namespace: props.defaultNamespace, name: '',
    replicas: 2, serviceName: '', storage: '', mountPath: '/data',
    containers: [emptyCtr()],
  }
}

const generatedYaml = computed(() => {
  const f = form.value
  const kindName = f.kind === 'statefulset' ? 'StatefulSet' : 'Deployment'
  const ctrs = f.containers.map((c) => {
    const lines = [
      `        - name: ${c.name || 'app'}`,
      `          image: ${c.image || '<image>'}`,
    ]
    if (c.port) lines.push(`          ports:\n            - containerPort: ${c.port}`)
    if (c.cpu || c.memory) {
      lines.push(`          resources:\n            requests:\n              cpu: ${c.cpu || '100m'}\n              memory: ${c.memory || '128Mi'}`)
    }
    return lines.join('\n')
  }).join('\n')

  let spec = `  replicas: ${f.replicas}\n  selector:\n    matchLabels:\n      app: ${f.name || '<name>'}\n`
  spec += `  template:\n    metadata:\n      labels:\n        app: ${f.name || '<name>'}\n    spec:\n      containers:\n${ctrs}\n`
  if (f.kind === 'statefulset') {
    spec = `  serviceName: ${f.serviceName || f.name || '<name>'}\n` + spec
    if (f.storage) {
      spec += `  volumeClaimTemplates:\n    - metadata:\n        name: data\n      spec:\n        accessModes: [ReadWriteOnce]\n        resources:\n          requests:\n            storage: ${f.storage}\n`
    }
  }
  return `apiVersion: apps/v1\nkind: ${kindName}\nmetadata:\n  name: ${f.name || '<name>'}\n  namespace: ${f.namespace}\nspec:\n${spec}`
})

async function submit() {
  const f = form.value
  if (!f.name) return ElMessage.warning('请填写名称')
  const bad = f.containers.find((c) => !c.image)
  if (bad) return ElMessage.warning('每个容器都需要镜像')
  saving.value = true
  try {
    await http.post('/resources', { kind: f.kind, yaml: generatedYaml.value })
    ElMessage.success('部署成功')
    visible.value = false
    emit('deployed')
  } catch (e) {
    ElMessage.error('部署失败: ' + (e?.message || '未知错误'))
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.ctr-block { border: 1px solid var(--ks-border, #e4e8ef); border-radius: 6px; padding: 12px 0 4px; margin-bottom: 10px; }
.hint { color: var(--ks-text-secondary); font-size: 12px; margin-left: 8px; }
.yaml-preview { background: #f6f8fa; padding: 12px; border-radius: 6px; font-size: 12px; max-height: 300px; overflow: auto; }
</style>
