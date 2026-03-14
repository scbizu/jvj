# Devlog 文档按组件重组设计

## 1. 背景

当前 `docs/plans/` 同时承载了多个组件的设计文档与实现计划，文件名主要依赖日期前缀区分。  
随着 `connectrpc`、`message-bus`、`tape-service`、`executor`、`agent-skills` 等主题增加，目录已经更像“时间线快照”，而不是“组件视角的工程记录”。

这次整理的目标不是简单搬运文件，而是把这些文档按组件 domain 重新编排，并统一语言风格后归档到：

```text
docs/devlog/{component}/
```

## 2. 设计目标

1. 按组件归档历史设计和实现文档。
2. 用语义化文件名替代日期前缀文件名。
3. 允许根据组件 domain 改写原文标题、摘要和段落组织。
4. 保持文档原有意图，不额外扩张功能范围。
5. 迁移后让 `docs/devlog/{component}` 读起来像连续的组件开发记录。

## 3. 组件划分

本次按现有文档主题划分为 6 个组件目录：

```text
docs/devlog/
├── cobra-cli/
├── connectrpc/
├── message-bus/
├── tape-service/
├── executor/
└── agent-skills/
```

归属规则：

- `cobra-cli-*` -> `cobra-cli`
- `connectrpc-*` -> `connectrpc`
- `message-bus-*` -> `message-bus`
- `tape-service-*` -> `tape-service`
- `executor-script-planner-*` -> `executor`
- `agent-skills-handoff-*` -> `agent-skills`

## 4. 命名规则

迁移后不保留日期前缀，改用“组件目录 + 语义化文件名”。

建议映射如下：

| 旧文件 | 新文件 |
| --- | --- |
| `2026-02-24-cobra-cli-design.md` | `docs/devlog/cobra-cli/design.md` |
| `2026-02-24-cobra-cli-implementation.md` | `docs/devlog/cobra-cli/implementation.md` |
| `2026-02-24-connectrpc-design.md` | `docs/devlog/connectrpc/design.md` |
| `2026-02-24-connectrpc-architecture-implementation.md` | `docs/devlog/connectrpc/architecture-implementation.md` |
| `2026-02-24-message-bus-bot-design.md` | `docs/devlog/message-bus/bot-design.md` |
| `2026-02-24-message-bus-refinement-design.md` | `docs/devlog/message-bus/refinement-design.md` |
| `2026-02-24-message-bus-runtime-implementation.md` | `docs/devlog/message-bus/runtime-implementation.md` |
| `2026-02-24-message-bus-discord-telegram-implementation.md` | `docs/devlog/message-bus/discord-telegram-implementation.md` |
| `2026-03-14-tape-service-design-refinement.md` | `docs/devlog/tape-service/design-refinement.md` |
| `2026-03-14-tape-service-implementation.md` | `docs/devlog/tape-service/implementation.md` |
| `2026-03-14-executor-script-planner-design.md` | `docs/devlog/executor/script-planner-design.md` |
| `2026-03-14-executor-script-planner-implementation.md` | `docs/devlog/executor/script-planner-implementation.md` |
| `2026-03-14-agent-skills-handoff-design.md` | `docs/devlog/agent-skills/handoff-design.md` |
| `2026-03-14-agent-skills-handoff-implementation.md` | `docs/devlog/agent-skills/handoff-implementation.md` |

## 5. 内容重组规则

本次允许改写原文内容，但改写目标是**按组件 domain 对齐语言与结构**，而不是新增需求。

### 5.1 允许调整

- 标题改写
- 段落顺序调整
- 摘要与背景重写
- 重复内容合并
- 组件术语统一
- 对同组件多篇文档增加承接关系

### 5.2 不做的事

- 不凭空新增未讨论的功能设计
- 不改变既有设计结论的核心语义
- 不把一个组件拆成更多新组件

## 6. 组件语言规范

迁移时按组件统一术语：

- `cobra-cli`：命令入口、配置、运行方式
- `connectrpc`：传输层、协议、服务边界
- `message-bus`：入口、路由、运行时、平台集成
- `tape-service`：事实、anchor、handoff、view
- `executor`：planner、script builder、executor
- `agent-skills`：agent native skills、built-in handoff、skill bundle、preload

## 7. 引用与兼容处理

迁移不只是文件移动，还需要同步更新引用：

1. `docs/architecture.md` 中引用的设计文档路径
2. `docs/devlog/*` 文档内部引用的旧 `docs/plans/...` 路径
3. `.agents/skills/*` 中写死的 `docs/plans/...` 默认输出路径

兼容策略：

- 保留一个薄的 `docs/plans/README.md`
- README 只说明文档已迁移到 `docs/devlog/{component}`
- 不保留大量重定向文件

## 8. 执行顺序

建议迁移顺序：

1. 新建 `docs/devlog/{component}` 目录
2. 按组件迁移并改写文档
3. 批量更新引用
4. 添加 `docs/plans/README.md`
5. 校验旧路径残留

## 9. 校验方式

迁移完成后执行：

- `git diff --check`
- `rg -n "docs/plans/" docs .agents`
- `go test ./...`

成功标准：

- 组件文档都落到 `docs/devlog/{component}`
- 旧路径引用只保留 README 等允许位置
- 文档内容按组件 domain 语言统一
- 仓库测试继续通过

## 10. 结论

这次整理应被视为一次**组件化文档归档与语言重组**，而不是单纯文件搬家。  
目标是把当前按日期堆积的 `docs/plans/`，重构为按组件组织、按领域语言表达的 `docs/devlog/{component}`。
