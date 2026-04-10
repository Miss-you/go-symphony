---
name: deriving-task-board-from-design
description: Use when an approved go-symphony design exists but the design-task tracking doc is missing, stale, or too coarse to support safe task claiming and parallel work.
---

# 从设计推导任务板

## 概述

把已批准的设计文档整理成可执行的任务板，而不是再写一份“第二设计文档”。

**核心原则：** 任务 ID、依赖边和状态历史是执行期真相源，不能依赖聊天记忆。

这个 skill 负责创建或刷新 `docs/plans/*-design-task.md`，不负责真正实现某个任务。

## 何时使用

在这些场景使用：

- `docs/plans/*-design.md` 已经批准，准备进入实现
- 对应的 `docs/plans/*-design-task.md` 还不存在
- task 文档已经存在，但过时、依赖缺失，或描述得太粗，无法安全认领
- 可能会有多个 agent 并行工作，需要一个统一执行真相源

这些场景不要使用：

- 设计本身还在变化，此时先用 `compatibility-first-planning`
- 你已经认领了某个任务并进入执行，此时改用 `delivering-go-task-end-to-end`

## 必做检查

在落笔之前，先做这些检查：

1. 确认源设计文件真实存在。
2. 用同一个 stem 推导任务文档路径：`*-design.md` -> `*-design-task.md`。
3. 检查任务文档是否已经存在。
4. 如果已存在，除非有证据反驳，否则保留稳定 ID 以及已有的完成/认领历史。

不要凭记忆认领任务、重排任务 ID，或推进任务状态。

## 输出形状

任务文档要短、清晰、偏执行。固定包含这些 section：

1. `Source Design`
2. `Status Legend`
3. `Dependency Rules`
4. `Task Table`
5. `Claiming Rules`
6. `Change Log`

`Task Table` 保持最小化：

| ID | Title | Goal | Depends On | Parallel | Status | Owner | Claimed At | Workspace | Change | Done When | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |

## 流程

### 1. 读取源设计

读取已批准的设计文档，并抽出这些内容：

- 目标与非目标
- 阶段划分和闭环
- 已经存在的粗粒度任务和稳定 ID
- core 与 compatibility shell 的边界约束

如果设计里已经有 `T01` 到 `T18` 这类 ID，优先复用，不要随意重排。

### 2. 初始化或刷新任务文档

如果任务文档不存在：

- 创建它
- 总任务数尽量控制在 `10-20`
- 按“可交付行为闭环”或“验证闭环”拆，不按目录名拆

如果任务文档已存在：

- 刷新，而不是整份重写
- 保留已经完成的任务、已认领的任务和历史备注
- 如果要合并或拆分任务，必须在 `Change Log` 里写明原因

### 3. 定义依赖与并行组

每个任务都要有：

- 明确的硬依赖
- 可选的并行组标记
- 一个具体的 `Done When`
- 在 `Notes` 里写出第一个预期验证命令或 gate

“ready” 应该是推导结果，而不是单独状态：

- `ready` 的含义是 `status=todo` 且所有硬依赖都已 `done`

不要再引入单独的 `ready` 状态。

### 4. 使用显式状态流转

状态集固定为：

- `todo`
- `claimed`
- `research`
- `spec`
- `implementing`
- `verifying`
- `review`
- `blocked`
- `done`

`blocked` 可以打断任何活动状态。解除阻塞后，要回到之前的工作状态。
把这个原状态记录在 `Notes` 中，例如 `resume_to=implementing`。

### 5. 定义认领规则

认领必须遵守这些规则：

- 只能认领 `todo` 且所有硬依赖都已 `done` 的任务
- 先更新 task 文档
- 记录 `Owner` 和 `Claimed At`
- 然后再创建 `workspace/<task-id>/`
- 把 workspace 路径写回任务表
- 在 `Change Log` 中追加认领记录
- 一个已认领任务只对应一个 workspace 目录

如果要选“下一个任务”，优先选未阻塞、能解锁更多后续工作的高杠杆任务。

### 6. 用真实产物回收状态

刷新已有 task 文档时，要以证据为准：

- 源设计中的任务 ID 和依赖意图
- 现存的 `workspace/<task-id>/` 文件
- 活跃的 OpenSpec change
- 已归档的 OpenSpec change
- 仓库里已经落下的验证输出

当证据冲突时，按这个优先级判断：

1. 源设计及其稳定任务 ID
2. 已归档或活跃的 OpenSpec 状态
3. workspace 产物和已记录的验证输出
4. 当前 task 文档文本

如果任务被标成 `claimed` 或 `done`，但证据缺失，不要默认沿用旧状态。保留历史，但要在 `Notes` 或 `Change Log` 里显式说明回退或阻塞。

如果一个被标成 `done` 的任务被证伪，就回退到“仍然有证据支持的最高状态”：

- `review`
- `verifying`
- `implementing`
- `spec`
- `research`
- `claimed`
- 否则回到 `todo`

不要因为聊天里说“做完了”，就把任务标成 `done` 或推进状态。

## 快速参考

| 场景 | 动作 |
| --- | --- |
| task 文档不存在 | 从已批准设计创建 `*-design-task.md` |
| task 文档存在但过时 | 刷新它，且不要改动稳定任务 ID |
| 任务依赖都完成且状态是 `todo` | 它是可认领的 |
| 任务已经被认领 | `workspace/<task-id>/` 必须真实存在 |
| 设计中的粗任务已经够好 | 直接复用，不要再发明新 ID |

## 常见错误

- 把 task 文档写成第二份设计说明
- 重排稳定任务 ID，导致历史失真
- 忘了写 `Done When`，让完成条件变主观
- 把依赖和并行关系藏在 prose 里，不落在表格里
- 还没更新 task 文档就先认领或先开工
- 把聊天历史当真相，而不是仓库里的真实产物

## 交接

当 task 文档已经初始化或刷新完成后，使用 `delivering-go-task-end-to-end` 去认领并执行某一个任务。
