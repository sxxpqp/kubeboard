# Kubeboard

Kubernetes 多维度可视化管理控制台，前端 KubeSphere 3.x 风格，后端 Go + Gin。

## 功能

- **集群概览**：版本、节点、命名空间、Pod 状态、部署、服务、近期事件
- **节点管理**：状态、角色、容量、Kubelet 版本、操作系统
- **应用负载**：Deployment / StatefulSet / DaemonSet（伸缩、镜像更新、滚动重启）
- **容器组**：状态、重启次数、节点、Pod IP、日志、终端
- **存储管理**：PVC / PV / StorageClass
- **配置与密钥**：ConfigMap / Secret / ServiceAccount
- **网络**：Service / Ingress
- **任务**：Job / CronJob
- **事件**：按命名空间、级别过滤

## 架构

```
cmd/kubeboard/         入口
internal/config/       启动参数与配置
internal/auth/         JWT 鉴权 + 登录限流
internal/k8s/          client-go 封装
internal/api/          HTTP/WebSocket 接口
web/                   Vue 3 + Element Plus + Vite 前端
deploy/                Kubernetes 部署清单（最小权限 RBAC）
```

## 快速开始

### 本地运行（对默认 kubeconfig 指向的集群）

```bash
# 构建前端
cd web && npm install && npm run build && cd ..

# 构建并运行后端
go build -o kubeboard ./cmd/kubeboard
./kubeboard --static ./web/dist --listen :9090
```

浏览器打开 `http://localhost:9090`，默认账号 `admin` / `kubeboard@2024`。

### 启动参数

| 参数 | 环境变量 | 默认值 | 说明 |
|---|---|---|---|
| `--listen` | — | `:9090` | 监听地址 |
| `--kubeconfig` | — | 空（in-cluster / `~/.kube/config`） | kubeconfig 路径 |
| `--jwt-secret` | `KUBEBOARD_JWT_SECRET` | `kubeboard-dev-secret` | JWT 签名密钥 |
| `--admin-user` | — | `admin` | 管理员用户名 |
| `--admin-password` | `KUBEBOARD_ADMIN_PASSWORD` | `kubeboard@2024` | 管理员密码 |
| `--static` | — | 空（不托管前端） | 前端构建产物目录 |
| `--enable-exec` | `KUBEBOARD_ENABLE_EXEC` | `false` | 开启 Pod 终端（需 `pods/exec` RBAC） |

> ⚠️ **安全**：默认凭据仅用于本地开发。生产部署务必通过环境变量或 Secret 覆盖
> `KUBEBOARD_JWT_SECRET` 和 `KUBEBOARD_ADMIN_PASSWORD`。

## 部署到 Kubernetes（最小权限）

`deploy/` 下提供最小权限模型：

- `kubeboard-reader`：只读所有资源（**默认授予**）
- `kubeboard-operator`：写操作（伸缩/重启/删除等），**不含** `pods/exec`
- `kubeboard-exec`：仅 `pods/exec`，配合 `--enable-exec` 使用

```bash
kubectl apply -f deploy/rbac.yaml
# 编辑 deploy/deployment.yaml 替换 jwt-secret / admin-password 后：
kubectl apply -f deploy/deployment.yaml
```

按需放开权限：取消 `deploy/rbac.yaml` 中对应 ClusterRoleBinding 的注释。

## 安全说明

- 登录限流：按用户名（5 次 / 5 分钟）与来源 IP（20 次 / 5 分钟）双维度独立累计
- 终端 shell 白名单：`/bin/sh`、`/bin/bash`、`/bin/ash`、`/bin/zsh`
- 终端默认关闭，需显式 `--enable-exec`
- 所有登录尝试与 exec 操作均有审计日志

## License

MIT
