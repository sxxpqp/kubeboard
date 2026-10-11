<template>
  <!-- Service 表单 -->
  <div v-if="kind === 'service'">
    <el-collapse v-model="open">
      <el-collapse-item name="basic">
        <template #title><span class="sec-title">基础信息</span></template>
        <el-form-item label="项目">
          <el-select v-model="form.namespace" filterable style="width: 260px">
            <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" style="width: 260px" /></el-form-item>
        <el-form-item label="类型">
          <el-radio-group v-model="form.type">
            <el-radio-button value="ClusterIP">ClusterIP</el-radio-button>
            <el-radio-button value="NodePort">NodePort</el-radio-button>
            <el-radio-button value="LoadBalancer">LoadBalancer</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" /></el-form-item>
        <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
        <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
      </el-collapse-item>

      <el-collapse-item name="ports">
        <template #title><span class="sec-title">端口与后端</span></template>
        <el-form-item label="选择器（后端 Pod 标签）">
          <KeyValueEditor v-model="form.selector" key-placeholder="标签键 如 app" value-placeholder="值 如 web" />
        </el-form-item>
        <el-form-item label="端口">
          <div class="port-list">
            <div v-for="(p, pi) in form.ports" :key="pi" class="port-row">
              <el-input v-model="p.name" placeholder="名称" style="width: 110px" />
              <el-input-number v-model="p.port" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 120px" />
              <span class="arrow">→</span>
              <el-input-number v-model="p.targetPort" :min="1" :max="65535" controls-position="right" placeholder="目标" style="width: 120px" />
              <el-select v-model="p.protocol" style="width: 95px">
                <el-option label="TCP" value="TCP" /><el-option label="UDP" value="UDP" />
              </el-select>
              <el-input-number v-if="form.type === 'NodePort'" v-model="p.nodePort" :min="30000" :max="32767" controls-position="right" placeholder="nodePort" style="width: 130px" />
              <el-button link type="danger" size="small" @click="form.ports.splice(pi, 1)">删除</el-button>
            </div>
            <el-button link type="primary" size="small" @click="form.ports.push({ name: '', port: 80, targetPort: 80, protocol: 'TCP', nodePort: null })">添加端口</el-button>
          </div>
        </el-form-item>
        <el-form-item label="会话保持">
          <el-select v-model="form.sessionAffinity" style="width: 200px">
            <el-option label="None（不保持）" value="None" />
            <el-option label="ClientIP（按来源 IP）" value="ClientIP" />
          </el-select>
        </el-form-item>
        <el-form-item label="外部流量策略" v-if="form.type !== 'ClusterIP'">
          <el-select v-model="form.externalTrafficPolicy" style="width: 200px">
            <el-option label="Cluster（集群级）" value="Cluster" />
            <el-option label="Local（仅本节点）" value="Local" />
          </el-select>
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
  </div>

  <!-- ConfigMap 表单 -->
  <div v-else-if="kind === 'configmap'">
    <el-collapse v-model="open">
      <el-collapse-item name="basic">
        <template #title><span class="sec-title">基础信息</span></template>
        <el-form-item label="项目">
          <el-select v-model="form.namespace" filterable style="width: 260px">
            <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" style="width: 260px" /></el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" /></el-form-item>
        <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
        <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
      </el-collapse-item>
      <el-collapse-item name="data">
        <template #title><span class="sec-title">数据（{{ form.data.length }} 项）</span></template>
        <el-form-item label="键值对">
          <div class="data-list">
            <div v-for="(d, di) in form.data" :key="di" class="data-row">
              <el-input v-model="d.key" placeholder="键名 如 app.properties" style="width: 220px" />
              <el-button v-if="isMultiline(d.value)" link type="primary" size="small" @click="toggleMulti(di)">收起</el-button>
              <el-input v-else v-model="d.value" placeholder="值" style="width: 380px" />
              <el-button v-if="!isMultiline(d.value)" link size="small" @click="expandMulti(di)">多行</el-button>
              <el-button link type="danger" size="small" @click="form.data.splice(di, 1)">删除</el-button>
            </div>
            <el-input
              v-if="multiIdx >= 0"
              v-model="form.data[multiIdx].value"
              type="textarea" :rows="6"
              class="multi-input"
              placeholder="多行内容，如配置文件"
            />
            <el-button link type="primary" size="small" @click="form.data.push({ key: '', value: '' })">添加数据项</el-button>
          </div>
        </el-form-item>
        <el-form-item label="不可变">
          <el-switch v-model="form.immutable" />
          <span class="hint">设为不可变后，只能删除重建</span>
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
  </div>

  <!-- Secret 表单 -->
  <div v-else-if="kind === 'secret'">
    <el-collapse v-model="open">
      <el-collapse-item name="basic">
        <template #title><span class="sec-title">基础信息</span></template>
        <el-form-item label="项目">
          <el-select v-model="form.namespace" filterable style="width: 260px">
            <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" style="width: 260px" /></el-form-item>
        <el-form-item label="类型">
          <el-select v-model="form.type" style="width: 300px">
            <el-option v-for="t in secretTypes" :key="t.value" :label="t.label" :value="t.value" />
          </el-select>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" /></el-form-item>
        <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
        <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
      </el-collapse-item>
      <el-collapse-item name="data">
        <template #title><span class="sec-title">数据（{{ form.data.length }} 项，值为 base64）</span></template>
        <el-alert type="info" :closable="false" class="b64-tip">
          <template #title>
            <span>data 需 base64 编码。填明文后点「编码」自动转换，或直接填已编码值。</span>
          </template>
        </el-alert>
        <el-form-item label="键值对">
          <div class="data-list">
            <div v-for="(d, di) in form.data" :key="di" class="data-row">
              <el-input v-model="d.key" placeholder="键名 如 password" style="width: 200px" />
              <el-input v-model="d.value" placeholder="值（base64 或明文）" style="width: 360px" />
              <el-button link type="primary" size="small" @click="b64encode(di)">编码</el-button>
              <el-button link size="small" @click="b64decode(di)">解码</el-button>
              <el-button link type="danger" size="small" @click="form.data.splice(di, 1)">删除</el-button>
            </div>
            <el-button link type="primary" size="small" @click="form.data.push({ key: '', value: '' })">添加数据项</el-button>
          </div>
        </el-form-item>
        <el-form-item label="不可变">
          <el-switch v-model="form.immutable" />
          <span class="hint">设为不可变后，只能删除重建</span>
        </el-form-item>
      </el-collapse-item>
    </el-collapse>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import KeyValueEditor from './KeyValueEditor.vue'
import { SECRET_TYPES } from '../utils/simpleYaml'

const props = defineProps({
  kind: { type: String, required: true },
  modelValue: { type: Object, required: true },
  namespaces: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:modelValue'])

const open = ref(['basic', 'ports', 'data'])
const secretTypes = SECRET_TYPES
const multiIdx = ref(-1)

const form = computed({
  get: () => props.modelValue,
  set: (v) => emit('update:modelValue', v),
})

function isMultiline(v) { return typeof v === 'string' && v.includes('\n') }
function toggleMulti(i) { multiIdx.value = multiIdx.value === i ? -1 : i }
function expandMulti(i) { multiIdx.value = i }
function b64encode(i) {
  try {
    form.value.data[i].value = btoa(unescape(encodeURIComponent(form.value.data[i].value)))
    ElMessage.success('已编码')
  } catch { ElMessage.error('编码失败') }
}
function b64decode(i) {
  try {
    form.value.data[i].value = decodeURIComponent(escape(atob(form.value.data[i].value)))
    ElMessage.success('已解码')
  } catch { ElMessage.error('不是合法的 base64') }
}
</script>

<style scoped>
.sec-title { font-weight: 600; font-size: 14px; }
.port-row, .data-row { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.arrow { color: var(--ks-text-secondary, #86909c); }
.data-list { width: 100%; }
.multi-input { margin: 6px 0 10px; }
.hint { color: var(--ks-text-secondary, #86909c); font-size: 12px; margin-left: 8px; }
.b64-tip { margin-bottom: 10px; }
</style>
