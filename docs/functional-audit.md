# 功能检查记录

检查日期：2026-11-15。范围为当前工作区（包含已有未提交修改）。

## 验证结果

- `npm test`：通过，覆盖 YAML 转换、项目路由、资源内容、工作负载操作、完整模型、版本差异、字段渲染和 Pod 时间线。
- `npm run build`：通过；Nodes 与主入口产物超过 500 kB，存在体积警告。
- `GOCACHE=/tmp/kubeboard-go-cache go test ./... -race`：通过。
- `GOCACHE=/tmp/kubeboard-go-cache go vet ./...`：通过。

## 功能范围与证据边界

| 范围 | 本轮检查 | 尚缺验证 |
|---|---|---|
| 登录、JWT、限流、启动配置 | 后端测试、路由和鉴权代码 | 浏览器登录与过期跳转 |
| 概览、节点、命名空间 | 路由和处理器、前端构建 | 实际数据、指标服务缺失场景、节点操作 |
| 工作负载创建、编辑、YAML | 模型、转换、命令与字段渲染测试 | 集群创建保存、准入校验 |
| 伸缩、镜像更新、重启、暂停 | 接口与前端操作逻辑检查 | 集群执行与收敛 |
| 发布历史、比较、回退 | 后端版本测试、前端差异测试 | StatefulSet 部署权限、真实回退 |
| Pod 生命周期 | 时间线测试、处理器检查 | 初始化容器诊断（见问题） |
| 日志、终端 | 流式接口、开关和 shell 白名单代码 | 实际流、断连释放、终端交互 |
| Service、Ingress、配置、Secret、任务 | 转换测试、路由、构建 | 集群 CRUD 和关联数据 |
| 存储 | 后端存储详情测试、前端构建 | 实际绑定、存储类及供应器行为 |
| HPA、网络策略、事件、配额 | 接口、页面和部署权限检查 | 实际数据和写操作权限 |

## 已确认问题

1. **StatefulSet 历史在默认部署下无权限**：`internal/api/rollout.go` 列出 `apps/controllerrevisions`，`deploy/rbac.yaml` 的 reader 未授予该资源权限。影响历史查看及回退前的版本读取。
2. **运维角色与页面能力不一致**：operator 缺少 HPA、NetworkPolicy 的写权限；页面提供创建/删除。节点封锁需要 nodes 更新权限，PV 页面操作亦须逐项核对。启用 operator 不能保证所有页面写操作成功。
3. **Pod 生命周期遗漏初始化容器**：`internal/api/pod_lifecycle.go` 仅遍历 `Spec.Containers` / `ContainerStatuses`，没有 InitContainers / InitContainerStatuses；初始化失败时诊断不完整，重启总数也只统计普通容器。
4. **伸缩请求缺少显式输入校验**：`scaleWorkload` 的 replicas 为非指针，缺失字段会按 0 执行；负数也未在接口层拒绝。应区分缺失与合法的 0，并验证非负。

## 验收限制

本轮没有执行真实集群写操作，也未完成浏览器逐页面交互验收。自动测试通过不等于所有功能已验收。DaemonSet 历史/回退属于当前已说明的能力限制。本文记录检查结果，未修改功能或部署权限。

## 后续修复

- 伸缩接口已要求 replicas 必填且非负，新增无集群访问的非法参数回归测试。
- 部署清单已补 ControllerRevision 读取、HPA/NetworkPolicy 写入和 nodes 更新权限；尚未应用到集群。
- Pod 生命周期已包含初始化容器的状态、当前与历史终止信息，重启总数包含初始化容器；新增内存 HTTP transport 回归测试。
