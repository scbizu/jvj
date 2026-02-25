# Cobra CLI 封装设计

## 目标

将 `cmd/agent-runtime` 入口改为 Cobra 风格，支持 `root + run + version(占位)`，并保持“缺少 config 报错”的现有语义。

## 设计

1. `main.go` 改为 `Execute()` 驱动。
2. `run` 子命令负责实际执行：
   - 支持位置参数 `run <config-path>`
   - 支持 `--config` flag
   - 两者同时存在时优先使用 `--config`
3. `version` 子命令先输出固定版本占位值。

## 错误处理

- 无位置参数且未提供 `--config` 时返回 `config path is required`。
- 不做静默默认值。

## 测试

- 新增 `newRunCmd()` 行为测试：
  - 无参数失败
  - 传 `--config` 成功
