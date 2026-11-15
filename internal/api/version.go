package api

// APIVersion 标识后端 API 契约版本：前端依赖的接口集合发生变化（新增路由、
// 响应结构变更）时必须递增。前端启动时经 /api/version 读取并与自身构建时
// 注入的期望值比对，不匹配则展示告警横幅（P0-⑤，防旧二进制静默 404）。
//
// 构建期注入（覆盖默认值）：
//
//	go build -ldflags "-X github.com/sxxpqp/kubeboard/internal/api.APIVersion=v2"
var APIVersion = "v2"

// ConsoleVer 前端控制台版本展示用。
var ConsoleVer = "v3.2.0"
