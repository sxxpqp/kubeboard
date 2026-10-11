<template>
  <div class="login-page">
    <div class="login-box">
      <div class="login-title">
        <span class="logo-icon">K</span>
        <h2>Kubeboard</h2>
        <p>Kubernetes 多维度可视化管理控制台</p>
      </div>
      <el-form :model="form" @submit.prevent="doLogin">
        <el-form-item>
          <el-input v-model="form.username" placeholder="用户名" size="large">
            <template #prefix><el-icon><User /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-form-item>
          <el-input v-model="form.password" type="password" placeholder="密码" size="large" show-password @keyup.enter="doLogin">
            <template #prefix><el-icon><Lock /></el-icon></template>
          </el-input>
        </el-form-item>
        <el-button type="primary" size="large" style="width: 100%" :loading="loading" native-type="submit">
          登 录
        </el-button>
      </el-form>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import http from '../api/http'

const router = useRouter()
const form = reactive({ username: '', password: '' })
const loading = ref(false)

async function doLogin() {
  if (!form.username || !form.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }
  loading.value = true
  try {
    const data = await http.post('/login', form)
    localStorage.setItem('kubeboard_token', data.token)
    localStorage.setItem('kubeboard_username', data.username)
    router.push('/overview')
  } catch (e) {
    ElMessage.error(e.response?.data?.error || '登录失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1b2a3d 0%, #0c3a5e 100%);
}
.login-box {
  width: 380px;
  background: #fff;
  border-radius: 8px;
  padding: 40px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
}
.login-title { text-align: center; margin-bottom: 28px; }
.login-title h2 { margin: 12px 0 4px; color: var(--ks-text-primary); }
.login-title p { margin: 0; color: var(--ks-text-secondary); font-size: 12px; }
.logo-icon {
  width: 48px; height: 48px;
  background: var(--ks-primary);
  color: #fff;
  border-radius: 10px;
  display: inline-flex; align-items: center; justify-content: center;
  font-size: 24px; font-weight: 700;
}
</style>
