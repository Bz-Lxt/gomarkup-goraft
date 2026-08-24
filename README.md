# GoRaft

手写 Raft 的强一致性配置中心与集群声纳大屏。

## 1. 如何启动

```bash
docker compose up --build -d
```

浏览器打开 `http://127.0.0.1:28371`。5 个节点映射 `28372–28376`。默认 **Demo 模式**（选举超时 1500–3000ms）。所有容器 `TZ=Asia/Shanghai`。

## 2. 使用说明

大屏左侧是集群拓扑，右侧是 Raft 检视与写入瀑布流。底部可发起线性一致 KV 写入、注入 Kill/分区/延迟，并滚动全节点日志。顶部徽标标明 demo/production。观测通道（拓扑、检视）是本地脏读；KV 默认走 Leader ReadIndex。

## 3. 服务列表及API说明

| 服务 | URL |
|---|---|
| 大屏 | http://127.0.0.1:28371 |
| n1..n5 | http://127.0.0.1:28372 .. 28376 |
| 健康 | `GET /health` |
| 同源反代 | `/n1` .. `/n5` |

完整契约见 `docs/API.md`。

## 4. 测试账号

无登录。Chaos / KV 控制台对本地集群开放。

## 5. 题目内容

用 Go 手写 Raft（选举、心跳、日志复制、成员变更、WAL、快照），保证 Linearizable Read/Write，并用 React + D3 实时展示选举与复制全过程。

## 6. 项目结构

```
backend/           Go Raft 引擎与 HTTP/WS
frontend-user/     React + D3 大屏
docs/              Requirements / Roadmap / API / DesignSpec
tests/             Playwright + API smoke
docker-compose.yml
```

## 7. API 模拟与切换指南

本项目**没有外部计费 API，因此没有 Mock Provider，也没有 real/mock 开关**。共识、WAL、快照、RPC 全部是真实实现。网络分区与宕机由 RPC 层 Chaos 注入器模拟（应用层丢包，等价于分区，不使用 `NET_ADMIN`）。`RAFT_MODE=demo|production` 只改变超时，不切换假数据。
