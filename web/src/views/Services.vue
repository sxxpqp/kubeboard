<template>
 <ResourceListView kind="service" endpoint="/services" :columns="columns" :template="template" :ops-width="230">
  <template #col-ports="{row}"><el-tag v-for="(port,index) in row.ports" :key="index" size="small">{{port.port}}/{{port.protocol}}<template v-if="port.nodePort"> → {{port.nodePort}}</template></el-tag></template>
  <template #ops="{row,edit,remove}"><el-button link type="primary" @click.stop="testConnectivity(row)">后端检查</el-button><el-button link type="primary" @click.stop="edit(row.name,row.namespace)">编辑</el-button><el-button link type="danger" @click.stop="remove(row.name,row.namespace)">删除</el-button></template>
  <template #extra>
    <!-- 连通性测试结果 -->
    <el-dialog v-model="connVisible" :title="`后端检查: ${connTarget.namespace}/${connTarget.name}`" width="560px">
      <div v-loading="connLoading" style="min-height: 160px">
        <template v-if="conn">
          <div class="conn-head">
            <el-tag :type="conn.healthy ? 'success' : 'danger'" effect="dark" size="large">
              {{ conn.healthy ? '健康：有就绪后端' : '异常：无就绪后端' }}
            </el-tag>
            <span class="conn-type">{{ conn.type }}</span>
          </div>
          <el-descriptions :column="2" border size="small" style="margin-bottom: 16px">
            <el-descriptions-item label="Cluster IP">{{ conn.clusterIP || '-' }}</el-descriptions-item>
            <el-descriptions-item label="External Name">{{ conn.externalName || '-' }}</el-descriptions-item>
            <el-descriptions-item label="后端 Pod">{{ conn.backendPods }}</el-descriptions-item>
            <el-descriptions-item label="就绪 Pod">{{ conn.readyPods }}</el-descriptions-item>
            <el-descriptions-item label="Endpoints">{{ conn.endpointAddrs }}</el-descriptions-item>
            <el-descriptions-item label="选择器">
              <span v-if="Object.keys(conn.selector || {}).length">
                <el-tag v-for="(v, k) in conn.selector" :key="k" size="small" style="margin: 0 4px 2px 0">{{ k }}={{ v }}</el-tag>
              </span>
              <span v-else>-</span>
            </el-descriptions-item>
          </el-descriptions>
          <div class="conn-title">端口</div>
          <el-table :data="conn.ports || []" size="small">
            <el-table-column prop="name" label="名称" width="100" />
            <el-table-column prop="port" label="端口" width="80" />
            <el-table-column prop="targetPort" label="目标端口" width="100" />
            <el-table-column prop="protocol" label="协议" width="80" />
            <el-table-column prop="nodePort" label="NodePort" width="100">
              <template #default="{ row }">{{ row.nodePort || '-' }}</template>
            </el-table-column>
          </el-table>
          <div class="conn-hint">
            {{ conn.readyPods > 0
              ? `✓ ${conn.readyPods} 个就绪后端，尚未验证实际网络连通性`
              : '✗ 无就绪后端，请检查 Pod、选择器或外部后端配置' }}
          </div>
        </template>
      </div>
      <template #footer>
        <el-button @click="loadConn" :loading="connLoading">重新测试</el-button>
        <el-button type="primary" @click="connVisible = false">关闭</el-button>
      </template>
    </el-dialog>

  </template>
 </ResourceListView>
</template>
<script setup>
import {ref} from 'vue'
import {ElMessage} from 'element-plus'
import ResourceListView from '../components/ResourceListView.vue'
import http from '../api/http'
const connVisible=ref(false),connLoading=ref(false),conn=ref(null),connTarget=ref({namespace:'',name:''})
const columns=[{prop:'name',label:'名称',primary:true,minWidth:220,sortable:true},{prop:'namespace',label:'项目',width:150},{prop:'type',label:'类型',type:'tag',width:140},{prop:'clusterIP',label:'集群 IP',width:150},{prop:'externalIP',label:'外部 IP',width:150},{prop:'ports',label:'端口映射',minWidth:230}]
const template = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  selector:
    app: my-app
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
  type: ClusterIP
`

function testConnectivity(row) {
  connTarget.value = { namespace: row.namespace, name: row.name }
  conn.value = null
  connVisible.value = true
  loadConn()
}
async function loadConn() {
  connLoading.value = true
  try {
    conn.value = await http.get('/services/connectivity', {
      params: { namespace: connTarget.value.namespace, name: connTarget.value.name },
    })
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '连通性测试失败')
  } finally {
    connLoading.value = false
  }
}

</script>
<style scoped>.el-tag{margin:0 6px 4px 0}.conn-head{display:flex;gap:12px;align-items:center;margin-bottom:18px}.conn-hint{font-size:13px;margin-top:16px}.conn-title{margin-bottom:10px;font-weight:600}</style>
