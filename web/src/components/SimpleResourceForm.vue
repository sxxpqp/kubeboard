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

  <!-- ==================== Ingress 表单 ==================== -->
  <div v-else-if="kind === 'ingress'">
    <el-collapse v-model="open">
      <el-collapse-item name="basic">
        <template #title><span class="sec-title">基础信息</span></template>
        <el-form-item label="项目">
          <el-select v-model="form.namespace" filterable style="width: 260px">
            <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" style="width: 260px" /></el-form-item>
        <el-form-item label="Ingress 类">
          <el-input v-model="form.ingressClassName" placeholder="如 nginx（留空用默认控制器）" style="width: 300px" />
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" /></el-form-item>
        <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
        <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
      </el-collapse-item>

      <el-collapse-item name="rules">
        <template #title><span class="sec-title">路由规则（{{ form.rules.length }} 个域名）</span></template>
        <div v-for="(rule, ri) in form.rules" :key="ri" class="rule-block">
          <div class="rule-head">
            <el-input v-model="rule.host" placeholder="域名，如 app.example.com（留空匹配所有）" style="width: 320px" />
            <el-button v-if="form.rules.length > 1" link type="danger" size="small" @click="form.rules.splice(ri, 1)">删除域名</el-button>
          </div>
          <div v-for="(p, pi) in rule.paths" :key="pi" class="path-row">
            <el-input v-model="p.path" placeholder="/api" style="width: 160px" />
            <el-select v-model="p.pathType" style="width: 130px">
              <el-option label="Prefix" value="Prefix" />
              <el-option label="Exact" value="Exact" />
              <el-option label="ImplementationSpecific" value="ImplementationSpecific" />
            </el-select>
            <span class="arrow">→</span>
            <el-input v-model="p.serviceName" placeholder="后端 Service 名" style="width: 180px" />
            <el-input-number v-model="p.servicePort" :min="1" :max="65535" controls-position="right" placeholder="端口" style="width: 120px" />
            <el-button v-if="rule.paths.length > 1" link type="danger" size="small" @click="rule.paths.splice(pi, 1)">删除</el-button>
          </div>
          <el-button link type="primary" size="small" @click="rule.paths.push({ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 })">添加路径</el-button>
        </div>
        <el-button link type="primary" @click="form.rules.push({ host: '', paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }] })">添加域名</el-button>
      </el-collapse-item>

      <el-collapse-item name="tls">
        <template #title><span class="sec-title">TLS 证书（{{ form.tls.length }} 项）</span></template>
        <div v-for="(t, ti) in form.tls" :key="ti" class="data-row">
          <el-input v-model="t.hosts" placeholder="域名，多个逗号分隔" style="width: 300px" />
          <el-input v-model="t.secretName" placeholder="TLS Secret 名称" style="width: 220px" />
          <el-button link type="danger" size="small" @click="form.tls.splice(ti, 1)">删除</el-button>
        </div>
        <el-button link type="primary" size="small" @click="form.tls.push({ hosts: '', secretName: '' })">添加 TLS</el-button>
      </el-collapse-item>
    </el-collapse>
  </div>

  <!-- ==================== Job / CronJob 共用容器表单 ==================== -->
  <div v-else-if="kind === 'job' || kind === 'cronjob'">
    <el-collapse v-model="open">
      <el-collapse-item name="basic">
        <template #title><span class="sec-title">基础信息</span></template>
        <el-form-item label="项目">
          <el-select v-model="form.namespace" filterable style="width: 260px">
            <el-option v-for="n in namespaces" :key="n" :label="n" :value="n" />
          </el-select>
        </el-form-item>
        <el-form-item label="名称" required><el-input v-model="form.name" style="width: 260px" /></el-form-item>
        <el-form-item v-if="kind === 'cronjob'" label="调度表达式" required>
          <el-input v-model="form.schedule" placeholder="0 2 * * *（分 时 日 月 周）" style="width: 300px" />
          <span class="hint">示例：0 2 * * * = 每天 2 点</span>
        </el-form-item>
        <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="2" style="width: 460px" /></el-form-item>
        <el-form-item label="标签"><KeyValueEditor v-model="form.labels" /></el-form-item>
        <el-form-item label="注解"><KeyValueEditor v-model="form.annotations" /></el-form-item>
      </el-collapse-item>

      <el-collapse-item name="job">
        <template #title><span class="sec-title">任务参数</span></template>
        <el-form-item v-if="kind === 'job'" label="完成次数">
          <el-input-number v-model="form.completions" :min="1" controls-position="right" />
          <span class="hint">需要成功完成的 Pod 总数</span>
        </el-form-item>
        <el-form-item v-if="kind === 'job'" label="并发数">
          <el-input-number v-model="form.parallelism" :min="1" controls-position="right" />
          <span class="hint">同时运行的 Pod 数</span>
        </el-form-item>
        <el-form-item v-if="kind === 'cronjob'" label="并发策略">
          <el-select v-model="form.concurrencyPolicy" style="width: 240px">
            <el-option label="Allow（允许并发）" value="Allow" />
            <el-option label="Forbid（禁止并发）" value="Forbid" />
            <el-option label="Replace（替换旧任务）" value="Replace" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="kind === 'cronjob'" label="保留成功/失败记录">
          <el-input-number v-model="form.successfulJobsHistoryLimit" :min="0" controls-position="right" style="width: 120px" />
          <el-input-number v-model="form.failedJobsHistoryLimit" :min="0" controls-position="right" style="width: 120px; margin-left: 8px" />
        </el-form-item>
        <el-form-item label="失败重试上限">
          <el-input-number v-model="form.backoffLimit" :min="0" controls-position="right" style="width: 140px" />
        </el-form-item>
        <el-form-item label="超时秒数">
          <el-input-number v-model="form.activeDeadlineSeconds" :min="0" controls-position="right" placeholder="不限" style="width: 160px" />
        </el-form-item>
        <el-form-item label="完成后清理(秒)">
          <el-input-number v-model="form.ttlSecondsAfterFinished" :min="0" controls-position="right" placeholder="不清理" style="width: 160px" />
        </el-form-item>
        <el-form-item label="重启策略">
          <el-select v-model="form.restartPolicy" style="width: 200px">
            <el-option label="OnFailure（失败重启）" value="OnFailure" />
            <el-option label="Never（不重启）" value="Never" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="kind === 'cronjob'" label="暂停调度">
          <el-switch v-model="form.suspend" />
        </el-form-item>
      </el-collapse-item>

      <el-collapse-item name="containers">
        <template #title><span class="sec-title">容器（{{ form.containers.length }}）</span></template>
        <div v-for="(c, ci) in form.containers" :key="ci" class="ctr-block">
          <el-form-item label="容器名"><el-input v-model="c.name" style="width: 200px" /></el-form-item>
          <el-form-item label="镜像" required><el-input v-model="c.image" placeholder="busybox:1.36" style="width: 340px" /></el-form-item>
          <el-form-item label="启动命令"><el-input v-model="c.command" placeholder="/bin/sh -c（多个用空格分隔）" style="width: 420px" /></el-form-item>
          <el-form-item label="参数"><el-input v-model="c.args" placeholder="如 echo hello" style="width: 420px" /></el-form-item>
          <el-form-item label="环境变量"><KeyValueEditor v-model="c.env" key-placeholder="变量名" /></el-form-item>
          <el-form-item label="资源配额">
            <div class="res-grid">
              <div class="res-col">
                <div class="res-sub">requests</div>
                <el-input v-model="c.cpuReq" placeholder="CPU" style="width: 140px" />
                <el-input v-model="c.memReq" placeholder="内存" style="width: 140px; margin-top: 6px" />
              </div>
              <div class="res-col">
                <div class="res-sub">limits</div>
                <el-input v-model="c.cpuLim" placeholder="CPU" style="width: 140px" />
                <el-input v-model="c.memLim" placeholder="内存" style="width: 140px; margin-top: 6px" />
              </div>
            </div>
          </el-form-item>
        </div>
      </el-collapse-item>
    </el-collapse>
  </div>

  <!-- ==================== DaemonSet 表单 ==================== -->
  <div v-else-if="kind === 'daemonset'">
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

      <el-collapse-item name="containers">
        <template #title><span class="sec-title">容器（{{ form.containers.length }}）</span></template>
        <div v-for="(c, ci) in form.containers" :key="ci" class="ctr-block">
          <el-form-item label="容器名"><el-input v-model="c.name" style="width: 200px" /></el-form-item>
          <el-form-item label="镜像" required><el-input v-model="c.image" placeholder="fluentd:1.16" style="width: 340px" /></el-form-item>
          <el-form-item label="拉取策略">
            <el-select v-model="c.imagePullPolicy" style="width: 180px">
              <el-option label="IfNotPresent" value="IfNotPresent" />
              <el-option label="Always" value="Always" />
              <el-option label="Never" value="Never" />
            </el-select>
          </el-form-item>
          <el-form-item label="启动命令"><el-input v-model="c.command" style="width: 420px" /></el-form-item>
          <el-form-item label="参数"><el-input v-model="c.args" style="width: 420px" /></el-form-item>
          <el-form-item label="端口">
            <div class="port-list">
              <div v-for="(p, pi) in c.ports" :key="pi" class="port-row">
                <el-input v-model="p.name" placeholder="名称" style="width: 110px" />
                <el-input-number v-model="p.containerPort" :min="1" :max="65535" controls-position="right" style="width: 120px" />
                <el-select v-model="p.protocol" style="width: 95px">
                  <el-option label="TCP" value="TCP" /><el-option label="UDP" value="UDP" />
                </el-select>
                <el-button link type="danger" size="small" @click="c.ports.splice(pi, 1)">删除</el-button>
              </div>
              <el-button link type="primary" size="small" @click="c.ports.push({ name: '', containerPort: 80, protocol: 'TCP' })">添加端口</el-button>
            </div>
          </el-form-item>
          <el-form-item label="环境变量"><KeyValueEditor v-model="c.env" key-placeholder="变量名" /></el-form-item>
          <el-form-item label="资源配额">
            <div class="res-grid">
              <div class="res-col">
                <div class="res-sub">requests</div>
                <el-input v-model="c.cpuReq" placeholder="CPU" style="width: 140px" />
                <el-input v-model="c.memReq" placeholder="内存" style="width: 140px; margin-top: 6px" />
              </div>
              <div class="res-col">
                <div class="res-sub">limits</div>
                <el-input v-model="c.cpuLim" placeholder="CPU" style="width: 140px" />
                <el-input v-model="c.memLim" placeholder="内存" style="width: 140px; margin-top: 6px" />
              </div>
            </div>
          </el-form-item>
          <el-form-item label="使用宿主网络">
            <el-switch v-model="c.hostNetwork" />
            <el-checkbox v-model="c.hostPID" style="margin-left: 16px">使用宿主 PID 空间</el-checkbox>
          </el-form-item>
        </div>
      </el-collapse-item>

      <el-collapse-item name="advanced">
        <template #title><span class="sec-title">高级设置</span></template>
        <el-form-item label="更新策略">
          <el-select v-model="form.updateStrategy" style="width: 200px">
            <el-option label="RollingUpdate（滚动）" value="RollingUpdate" />
            <el-option label="OnDelete（手动）" value="OnDelete" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="form.updateStrategy === 'RollingUpdate'" label="maxUnavailable">
          <el-input v-model="form.maxUnavailable" placeholder="1 或 25%" style="width: 150px" />
        </el-form-item>
        <el-form-item label="服务账号"><el-input v-model="form.serviceAccount" style="width: 240px" /></el-form-item>
        <el-form-item label="节点选择"><KeyValueEditor v-model="form.nodeSelector" key-placeholder="标签键" /></el-form-item>
        <el-form-item label="容忍（Tolerations）"><KeyValueEditor v-model="form.tolerations" key-placeholder="键" value-placeholder="值（可空）" /></el-form-item>
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
