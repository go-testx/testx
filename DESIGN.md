# testx 设计说明

## 目标

testx 不是一个替代 `testing` 的 runner，而是一个可渐进采用的 Go testing DX toolkit。

核心目标：

- 简单测试尽可能只写「输入、期望、意图」。
- 不强迫所有测试进入 DSL。
- 保留标准 `*testing.T` 作为底座和 escape hatch。
- 能够在同一个项目中自由混用不同抽象级别。
- 核心 API 尽量泛型化，避免用反射识别函数签名。
- 高层能力建立在低层 primitive 上，而不是另起一套体系。

## 三个抽象 Level

### Level 1 — Standard testing

完全使用标准库。

testx 对这一层没有任何影响。

### Level 2 — DX primitives

由调用者自己控制：

- `t.Run`
- 循环
- setup / cleanup
- mock
- goroutine
- case 调度

而 testx 提供：

- `Case[I, O]`
- `Assert`
- `Require`
- go-cmp diff
- Error assertions
- Eventually 等 helper

这一层的重点是：**增强 testing，而不是接管 testing。**

如果 IDE 的 table-driven case 静态识别很重要，推荐：

```go
cases := []testx.Case[string, Result]{
    {Name: "a", Input: "a", Expect: wantA},
    {Name: "b", Input: "b", Expect: wantB},
}

for _, c := range cases {
    t.Run(c.Name, func(t *testing.T) {
        // ...
    })
}
```

### Level 3 — Declarative orchestration

允许 testx 隐藏重复控制流：

```go
testx.RunErr(t, Parse,
    testx.C("info", "INFO hello", wantInfo),
    testx.C("debug", "DEBUG hello", wantDebug),
)
```

这一层愿意用一部分 IDE 静态识别能力交换更少的样板代码。

运行时仍使用真实 `t.Run`，因此子测试、CI 输出、`go test -run` 等标准机制仍然存在。

## Primitive first, orchestration second

这是最重要的内部设计原则。

`Case` 首先是普通数据：

```go
type Case[I, O any] struct {
    Name   string
    Input  I
    Expect O
}
```

Level 2 可以自己消费它；Level 3 的 runner 也消费同一个类型。

因此从：

```go
testx.RunErr(t, fn, cases...)
```

降级到：

```go
for _, c := range cases {
    t.Run(c.Name, func(t *testing.T) {
        // 自定义复杂逻辑
    })
}
```

不需要重写测试数据。

## Preset

Preset 表示“如何调用 subject”，而不是“测试本身是什么”。

内置 preset：

- `Func`: `func(I) O`
- `FuncErr`: `func(I) (O, error)`
- `ContextFuncErr`: `func(context.Context, I) (O, error)`
- `HTTP`: handler request/response
- `CLI`: subprocess request/result

两种使用方式并存：

```go
testx.RunErr(t, fn, cases...)
```

和：

```go
testx.FuncErr(fn).Run(t, cases...)
```

这些 preset 不改变 `Case` 的基础模型。JSON、Golden、Panic、Eventually、benchmark 和 fuzz 保持为 assertion/helper，因为它们不表示统一的 input/output subject 调用约定。

## Error expectation

不使用 `WantErr bool`。

原因：`bool` 无法表达错误语义。

使用：

```go
case.WithError(testx.AnyError())
case.WithError(testx.ErrorIs(ErrNotFound))
case.WithError(testx.ErrorContains("invalid"))
case.WithError(testx.ErrorMatch("validation error", predicate))
```

`ErrorExpectation` 的零值代表 NoError，因此普通成功 case 不需要额外配置。

## Interface testing / Contract

Go 编译器已经能解决“某个类型有没有实现 interface”：

```go
var _ Store = (*RedisStore)(nil)
```

框架需要解决的是另一件事：

> 不同实现是否满足同一个行为契约？

因此提供：

- `Contract[T]`
- `Spec[T]`
- `Factory[T]`
- `Implementation[T]`
- `Verify / VerifyAll`

Factory 每个 Spec 都重新调用一次，从而让 stateful implementation 默认隔离。

Contract 依旧遵守两级使用方式：

- Level 2：直接读取 `Contract.Specs`，自己组织测试。
- Level 3：调用 `Verify/VerifyAll` 自动编排。

## Assertion

基本形态：

```go
testx.Assert(t, actual).Equal(expected)
testx.Require(t, err).NoError()
```

选择 actual 在前，是为了避免 `assert.Equal(t, expected, actual)` 的参数顺序记忆成本。

- `Assert` -> `Errorf`
- `Require` -> `Fatalf`
- helper 内部必须 `t.Helper()`

结构比较使用 go-cmp，从而获得语义比较和 diff。

## 不做的事情

至少早期版本不做：

- 替代 `go test`
- 自己实现 test discovery protocol
- 自己实现 coverage / benchmark / fuzz runner
- 强制 BDD DSL
- 为了统一 API 而用反射接受任意函数签名
- 要求复杂测试也必须改写成 testx DSL

## 已实现的辅助能力

除核心 Runner 外，当前实现还包含：

- `Eventually` 与 `EventuallyValue`
- Context、HTTP、CLI preset
- JSON 语义比较
- Golden / Snapshot 文件比较
- Panic assertion
- collection matcher
- benchmark 与 fuzz seed helper
- 可选 VS Code CodeLens 扩展

这些能力都建立在标准 `testing`、`net/http/httptest`、`os/exec` 和 `go-cmp` 之上，不改变 Go 测试运行协议。

## API 稳定性方向

优先保持稳定：

- `Case[I, O]`
- `Assert / Require`
- `Run / RunErr`
- `Contract`

更实验性的能力放在 preset/helper 层，以便后续演进。
