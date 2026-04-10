---
name: delivering-go-task-end-to-end
description: Use when an approved go-symphony design or task exists, the next step is to claim one real task, and coordination is needed across workspace artifacts, OpenSpec state, code changes, and review gates.
---

# 端到端交付一个 Go 任务

## 概述

把一个已经认领的 go-symphony 任务从调研推进到关闭，同时避免 task 状态、spec 状态、代码状态和 review 状态彼此漂移。

**核心原则：** 只有当仓库产物、验证证据、OpenSpec 状态和任务板状态全部一致时，这个任务才算真正完成。

**隔离规则：** 从任务启动开始，后续 research、spec、实现、验证和 review 都必须在隔离的 git worktree 中进行，避免与同仓其他进行中的任务互相污染。

**REQUIRED SUB-SKILLS:** `using-git-worktrees`, `deriving-task-board-from-design`, `openspec-explore`, `openspec-ff-change`, `openspec-apply-change`, `openspec-sync-specs`, `openspec-archive-change`, `test-driven-development`, `requesting-code-review`, `receiving-code-review`, `verification-before-completion`

每个阶段切换前，都先更新 task 文档，再进入下一阶段。

## 何时使用

在这些场景使用：

- 已存在批准通过的 `docs/plans/*-design.md`
- 需要认领并执行一个具体任务
- 这个任务需要经历调研、OpenSpec、实现、校验、review 和最终关闭
- 可以用多个 subagent 协作，但仍然需要一个明确 owner 统筹

这些场景不要使用：

- design task board 不存在或已经过时，此时先用 `deriving-task-board-from-design`
- 工作还处于宽泛设计阶段，此时先用 `compatibility-first-planning`

## 每个阶段的统一循环

这里的“大阶段”包括：方案澄清、spec 与测试策略、代码实现、代码校验/review、最终对比与遗留风险梳理。

每个大阶段都必须按这个顺序执行：

1. 先明确本阶段目标是什么
2. 执行本阶段任务
3. 审核或校验本阶段结果
4. 修复发现的问题
5. 重新审核或复验，直到达到该阶段验收阈值

不要只做“执行”，不做阶段性审核；也不要为了测试而测试，或者因为时间压力跳过复验。

## 每个任务必须产出的文件

对于一个已认领任务，其 `workspace/<task-id>/` 至少应包含：

- `original_impl.md`
- `new_impl.md`
- `final_impl_v1.md`
- `final_impl.md`
- `test_strategy.md`
- `todo.md`，当 review、对比或 parity 仍有遗留时

如果这些文件不存在，就不要宣称对应阶段已经完成。

## 流程

### 1. 认领任务并启动

阶段目标：

- 选出一个真正可做的任务
- 先建立隔离 worktree，再把 task 文档、workspace 和 owner 关系固定下来

执行：

1. 读取批准通过的设计文档。
2. 读取对应的 `*-design-task.md`。
3. 如果 task 文档不存在，停止当前流程，改用 `deriving-task-board-from-design`。
4. 选择一个可认领任务：`status=todo` 且所有硬依赖都已经 `done`。
5. 在修改 task 文档、创建 `workspace/<task-id>/`、开始 research、创建 OpenSpec change 或写任何实现代码之前，必须先调用 `using-git-worktrees` 创建隔离 worktree。
6. 只有当 worktree 已创建、已完成该 skill 要求的基线校验、并且后续操作将继续在该 worktree 中进行时，才允许继续。
7. 在该隔离 worktree 中，先把 task 文档更新为 `claimed`。
8. 在该隔离 worktree 中，创建 `workspace/<task-id>/`。
9. 只有当上述动作都已经在该隔离 worktree 中完成，且正式开始调研时，才把任务切到 `research`。

校验：

- worktree 已创建，且后续任务执行上下文已经切换到该隔离 worktree
- task 文档已更新
- `workspace/<task-id>/` 真实存在
- 认领信息、`Owner`、`Claimed At` 已落盘

阶段验收阈值：

- worktree 已经按 `using-git-worktrees` 建好并完成基线校验
- 在共享主工作区里不再继续该任务的 research / spec / 实现 / 验证 / review
- 只有满足这两条，任务才允许进入 `research`

修复与复验：

- 如果认领记录缺失、workspace 未创建，或 worktree 尚未就绪，就先补齐，不要继续后续阶段

默认规则：

- 一个已认领任务默认对应一个 OpenSpec change，除非 task 文档明确写了例外

在每次状态切换时，都要往 `Change Log` 里追加一条简短记录，保证历史不依赖聊天上下文。

### 2. 方案澄清与实现方案定稿

阶段目标：

- 明确旧系统是怎么做的
- 明确如果单独在 go-symphony 里实现，自己会怎么做
- 再基于“尽可能贴近 Symphony，但更 Go-native，且不过度设计”的原则，定出最终方案

执行：

- `subagent 1`：使用 `openspec-explore`，查看 `/Users/lihui/Documents/GitHub/symphony`，输出到 `workspace/<task-id>/original_impl.md`
- `subagent 2`：使用 `openspec-explore`，查看当前 `go-symphony` 现状，输出到 `workspace/<task-id>/new_impl.md`
- `subagent 3`：结合任务目标和前两份结果，整合为 `workspace/<task-id>/final_impl_v1.md`

随后，至少再启动 `2+` 个 review subagent 评审 `final_impl_v1.md`。

审核评分规则：

| 维度 | 分值 |
| --- | --- |
| Symphony 对齐度与来源忠实度 | 30 |
| Go-native 简洁性与可维护性 | 20 |
| 不过度设计 / 边界是否干净 | 20 |
| 实现清晰度与可测试性 | 15 |
| 验证覆盖与落地安全性 | 15 |

阶段验收阈值：

- 平均分 `>= 80`
- 没有 reviewer 报告高严重度问题

修复与复验：

- 低于 `80` 分，或存在高严重度问题，就修改 `final_impl_v1.md` 并重新评审
- 达到阈值后，产出 `workspace/<task-id>/final_impl.md`

当 `final_impl.md` 被接受后，把任务状态更新为 `spec`。

### 3. Spec 与测试策略

阶段目标：

- 把 `final_impl.md` 变成可执行的 OpenSpec change
- 明确“要测试什么、为什么这样测试、哪些测试能证明功能正常运转”，而不是为了测试而测试或过度测试

执行：

1. 使用一个 subagent 配合 `openspec-ff-change`，根据 `final_impl.md` 创建该任务的 change 和 spec 产物。
2. 把 change 名写回 task 文档。
3. 再启动一个 subagent，单独产出 `workspace/<task-id>/test_strategy.md`，把功能目标映射到编译、lint、单测、e2e、关键集成验证。
4. 再启动一个独立 review subagent，结合任务目标、`final_impl.md`、新 spec 产物和 `test_strategy.md` 做 spec review。

高严重度 spec 问题的判定：

- spec 实现的行为与任务目标或 `final_impl.md` 不一致
- 违反任务要求的 Symphony parity 或边界约束
- 漏掉了必须存在的验证或 rollout 约束
- 让 task 文档、spec 产物和 change 范围彼此不一致
- `test_strategy.md` 无法证明关键功能能正常运转，只是在堆测试数量

阶段验收阈值：

- 没有高严重度 spec 问题
- 没有未解决的 scope mismatch
- `test_strategy.md` 能解释每类测试在证明什么，而不是机械罗列

修复与复验：

- 发现 spec 或测试策略问题，就先修 spec / `test_strategy.md`
- 在 spec review 通过前，不要开始实现

不要让 `final_impl.md`、spec 产物、`test_strategy.md` 和 task 文档互相打架。

当实现真正开始时，把任务状态更新为 `implementing`。

### 4. 按 TDD 实现代码

阶段目标：

- 以最小、可验证的增量实现已批准的行为
- 让测试真正证明功能是否正常，而不是事后补测试

执行规则：

- 使用 `openspec-apply-change`
- 遵循 `test-driven-development`
- 只有在写集完全不重叠时才并行
- 始终保留一个 owner 负责集成

不要把多个并行 agent 扔到同一组文件上。

### 5. 代码校验与验证

阶段目标：

- 证明实现不仅“看起来对”，而且在工程层面能跑、能编、能过核心测试

执行：

- 在进入验证前，把任务状态更新为 `verifying`
- 运行本任务要求的验证：
  - lint
  - build / compile
  - unit tests
  - 当任务触达 e2e 面时，运行 e2e tests

审核：

- 使用 `verification-before-completion`
- 对照 `test_strategy.md` 检查：当前验证是否真的覆盖了任务要证明的功能

阶段验收阈值：

- 规定的验证项全部通过
- 或者被显式跳过的项，已经在 `workspace/<task-id>/todo.md` 和 task 文档中写清理由

修复与复验：

- 任何编译、lint、单测、e2e 问题都要修复并重跑
- 如果验证结果表明测试设计无效，先修 `test_strategy.md` 和测试本身，再继续

### 6. 代码 Review 与问题分流

阶段目标：

- 在关闭前发现明显 bug、回归和低级错误
- 把“大架构讨论”和“本次必须修”的问题分开

执行：

- 进入 review 循环前，把任务状态更新为 `review`
- 使用 `requesting-code-review` 运行 review
- 再用 `receiving-code-review` 评估 review 建议

问题分流规则：

- 明确的 bug、回归、低级正确性问题：现在修
- 更大的架构争议、可接受的延后项：记录到 `workspace/<task-id>/todo.md`

阶段验收阈值：

- 没有未处理的明确 bug / regression 级问题
- review 结论与 task 文档、spec、实现状态一致

修复与复验：

- 修完后，重跑受影响的验证和 review gate

### 7. 最终对比、风险梳理与关闭

阶段目标：

- 回到原始目标和 Symphony 存量实现，确认当前方案没有明显严重偏差
- 把真正的遗留事项和风险落盘
- 完成 spec sync / archive，并把任务状态收口

执行：

在标记任务为 `done` 之前，必须依次完成：

1. 再做一次与原始 Symphony 实现及任务目标的对比。
2. 把真实遗留项写入 `workspace/<task-id>/todo.md`。
3. 运行 `openspec-sync-specs`。
4. 运行 `openspec-archive-change`。
5. 把 task 文档更新为 `done`。

审核：

- 对照任务目标、spec、实现、旧系统行为，确认没有未记录的高严重度风险

阶段验收阈值：

- task board、workspace 产物、spec、代码状态全部一致
- 所有剩余风险都已经被显式记录

修复与复验：

- 如果 final compare 暴露了高严重度问题，就回到对应阶段继续修，不要硬关单

只要 task board、workspace 产物、spec 和代码里还有任一处不一致，这个任务就不算完成。

## 快速参考

| 阶段 | 最小证据 |
| --- | --- |
| `claimed` | worktree 已就绪，task 文档已更新，且 `workspace/<task-id>/` 已存在 |
| `research` | `original_impl.md` + `new_impl.md`，且这些产物生成于该任务的隔离 worktree |
| `spec` | `final_impl.md` + OpenSpec change + `test_strategy.md` |
| `implementing` | 代码实现已在对应 change 下展开 |
| `verifying` | lint/build/tests 已执行，并能解释它们证明了什么 |
| `review` | review 发现已记录并完成分流 |
| `done` | final compare 完成，sync + archive 完成，task 文档已更新 |

## 红旗信号

如果出现这些情况，就暂停并纠正流程：

- 任务还没认领就开始写代码
- 还没建立隔离 worktree，就在共享主工作区里直接开始 research、spec、实现或验证
- 结论留在聊天里，没有落到 `workspace/<task-id>/`
- `final_impl.md` 没经过带 rubric 的评审就被接受
- `test_strategy.md` 只是列测试，不解释这些测试证明什么
- review 反馈被忽略，没有做分流
- tests 通过了，就想在 sync/archive 之前直接标 `done`
- 没检查仓库真实状态，就声称文件或 change 已存在

## 常见错误

- 把 `80` 分门槛用成主观印象，而不是明确评分
- 认领任务后才想起建立隔离 worktree，导致 claim / workspace / 初始调研已经污染共享主工作区
- 不小心让一个 OpenSpec change 覆盖多个任务
- 阶段切换时忘记回写 task 文档
- 虽然开了 worktree，但后续又回到共享主工作区继续推进同一个任务
- 对同一写集乱开并行 agent
- 因为测试已经过了，就跳过最终的源实现对比
