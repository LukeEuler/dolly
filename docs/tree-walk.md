# 树形业务流程：Walk、Node 与 Context

## 设计目的

将业务执行顺序与节点实现分离，使流程结构可直接阅读、步骤可独立维护。

区块处理包含回滚检查、区块与交易写入、流水计算、余额更新等步骤。不同链可复用通用步骤，通过替换节点或子树表达 account、UTXO 及链特性差异，避免将差异集中为公共 struct 中不断增长的字段和分支。

| 组成 | 职责 |
| --- | --- |
| 树的构建代码 | 定义业务步骤、顺序和层次，相当于流程目录 |
| Node | 实现一个业务步骤 |
| Walk | 按深度优先顺序串行执行节点 |
| Context | 保存必要的跨节点共享状态 |

流程示意，具体顺序由业务确定：

```text
处理区块
├── 回滚检查
├── 写入区块
├── 处理交易
│   ├── 写入交易
│   ├── 生成 transfer 或 vin/vout
│   └── 处理资产变动
│       ├── 计算流水
│       ├── 更新余额
│       └── 记录充值事件
└── 完成区块
```

## 执行规则

```go
type Node interface {
    GetChildren() []Node
    Enter(*Context) error
    Exit(*Context) error
    WrapErr(*Context, error) error
}
```

每个节点依次执行 `Enter → 按切片顺序遍历子树 → Exit`。例如，A 包含子节点 B、C 时：

```text
A.Enter → B.Enter → B.Exit → C.Enter → C.Exit → A.Exit
```

`DefaultNode` 提供空的 `Enter`、`Exit`、原样返回错误的 `WrapErr`，以及保存子节点的 `Children` 字段。业务节点可嵌入它并覆盖所需方法。

`Walk` 在 `Enter` 成功后读取子节点，因此允许在 `Enter` 中按区块数据构建交易或 transfer 子树。复用节点实例时，应重建动态子节点，避免残留数据。

## 使用示例

以下为可运行的最小流程示例。`step` 用于演示执行轨迹，业务中可按职责定义不同节点类型。`buildBlockFlow` 是流程目录。

```go
package main

import (
    "fmt"

    "github.com/LukeEuler/dolly/common/tree"
)

const blockHeightKey = "block.height"

type step struct {
    tree.DefaultNode
    name string
}

func (s *step) Enter(ctx *tree.Context) error {
    height, err := ctx.Get[int64](blockHeightKey)
    if err != nil {
        return err
    }
    fmt.Printf("进入 %s：区块 %d\n", s.name, height)
    return nil
}

func (s *step) Exit(_ *tree.Context) error {
    fmt.Printf("退出 %s\n", s.name)
    return nil
}

func (s *step) WrapErr(_ *tree.Context, err error) error {
    return fmt.Errorf("%s: %w", s.name, err)
}

func newStep(name string, children ...tree.Node) *step {
    return &step{
        DefaultNode: tree.DefaultNode{Children: children},
        name:        name,
    }
}

func buildBlockFlow() tree.Node {
    return newStep("处理区块",
        newStep("回滚检查"),
        newStep("写入区块"),
        newStep("处理交易",
            newStep("写入交易"),
            newStep("处理 transfer"),
        ),
        newStep("完成区块"),
    )
}

func main() {
    ctx := tree.NewContext()
    ctx.Set(blockHeightKey, int64(100))

    if err := tree.Walk(buildBlockFlow(), ctx); err != nil {
        panic(err)
    }
}
```

## Context

通过 `NewContext()` 初始化，使用 `Set` 写入、`Get[T]` 读取：

```go
ctx := tree.NewContext()
ctx.Set("block.height", int64(100))
height, err := ctx.Get[int64]("block.height")
```

- `Get[T]` 执行类型断言，不做类型转换；`int64` 不能按 `int` 读取。
- 键不存在或类型不匹配时，返回 `T` 的零值和错误。
- 所有节点共享同一个 Context；没有节点作用域或自动状态恢复。
- `Clear()` 清空全部键，影响整次遍历；通常由调用方管理。
- 零值 Context 的 map 未初始化，不能直接 `Set`；Context 不提供并发保护、取消或超时。

Context 仅用于必要的跨节点数据，少用、精用。固定服务和配置优先保存在节点字段中。

## 错误与执行边界

| 失败位置 | 执行结果 |
| --- | --- |
| 当前节点 Enter | 跳过其子节点和 Exit，包装错误后返回 |
| 子节点 | 错误向上传播时停止后续兄弟节点，跳过祖先 Exit |
| 当前节点 Exit | 包装错误后返回 |

错误通过失败节点及各层祖先的 `WrapErr` 逐层包装。建议保留错误链，并保持非 nil 输入对应非 nil 输出；返回 nil 可能使上层继续执行。

**Exit 仅在 Enter 和全部子节点成功后执行，不保证用于资源清理。** Walk 不捕获 panic，不自动回滚，不处理数据库与消息一致性。必要的资源释放和事务收尾应由其所属调用边界保证。

## 编写约定

- 业务顺序集中在树的构建位置；节点内部避免调度其他业务节点。
- 节点名称表达业务动作，节点实现保持单一职责。
- 复用通用步骤；差异使用不同节点或子树表达。
- 保持树结构无环、串行执行；不并发修改共享节点或 Context。
