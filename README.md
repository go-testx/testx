# testx

一个偏 DX（Developer Experience）的 Go testing 工具层：**不替代 `testing`，而是允许你自行选择抽象程度。**

```text
Level 1  标准库 testing
Level 2  testing + Case / Assert / Require
Level 3  Run / RunErr / Preset / Contract
```

高层 API 只是低层 primitive 的编排。任何时候测试复杂了，都可以退回标准 `testing.T`。

## 快速选择

| 你的测试场景 | 推荐入口 |
| --- | --- |
| 需要完整控制 `t.Run` 结构，或希望 IDE 原生识别每一行 table case | `Case` + `Assert` / `Require` |
| 测试 `func(I) O` | `Run` 或 `Func(...).Run` |
| 测试 `func(I) (O, error)` | `RunErr` 或 `FuncErr(...).Run` |
| 测试 `func(context.Context, I) (O, error)` | `ContextFuncErr(...).Run` |
| 多个 interface 实现共享同一套行为规范 | `Contract` |
| 等待异步状态最终满足 | `Eventually` / `EventuallyValue` |
| 测试 `http.Handler` | `HTTP(...).Run` |
| 测试命令行程序 | `CLI().Run` |
| 比较结构化 JSON | `JSONEqual` |
| 校验文本或文件快照 | `Golden` / `Snapshot` |

不确定时，从 `Assert` / `Require` 和手写 `t.Run` 开始；重复样板代码明显后，再升级到 `Run`、`Preset` 或 `Contract`。

## 安装

```bash
go get github.com/go-testx/testx
```

要求 Go 1.23 或更高版本。项目 CI 会在 Linux、macOS 和 Windows 上运行普通测试、vet 与 race 检查。

当前只依赖 `github.com/google/go-cmp`，`Equal` 和 declarative runner 的结构比较都由 go-cmp 完成。

## 给 AI / LLM 使用

仓库提供专门面向代码生成工具的资料，避免模型因为训练数据中没有 testx 而猜测 API：

- [`llms.txt`](llms.txt)：短入口和最重要的生成规则，适合文档抓取器首先读取。
- [`llms-full.txt`](llms-full.txt)：合并后的完整上下文，适合一次性加入模型上下文。
- [`docs/ai`](docs/ai)：按概览、API、配方、约束拆分的权威源文档。
- [`skills/testx`](skills/testx)：可分发的 Codex/Agent Skill，按任务渐进加载上述资料。

给普通 AI 编码工具的最小提示可以写成：

```text
Read llms.txt and the linked testx AI documentation before writing tests.
Do not invent testx APIs, and remember that assertions report failures themselves.
```

Skill 目录可以直接随仓库交给支持 Skill 的 Agent，或安装到其个人 Skill 目录。`docs/ai` 是唯一人工维护的机器文档源；运行 `go generate ./internal/aidocs` 会更新 `llms-full.txt` 和 Skill references，`go test ./...` 会检查生成物及公开 API 覆盖是否过期。

## 5 分钟快速开始

下面是一个可以直接放进 `_test.go` 文件的完整示例：

```go
package example_test

import (
    "strconv"
    "testing"

    "github.com/go-testx/testx"
)

func TestAtoi(t *testing.T) {
    testx.RunErr(t, strconv.Atoi,
        testx.C("decimal", "42", 42),
        testx.C("invalid", "not-a-number", 0).
            WithError(testx.AnyError()),
    )
}
```

运行全部测试、整个测试函数或单个 Case：

```bash
go test ./...
go test -run '^TestAtoi$'
go test -run '^TestAtoi$/^invalid$'
```

`RunErr` 默认要求函数不返回错误。只有预期失败的 Case 才需要 `.WithError(...)`。

## Level 1：纯 testing

完全不使用 testx：

```go
func TestParse(t *testing.T) {
    got, err := Parse("INFO hello")
    if err != nil {
        t.Fatal(err)
    }
    if got.Message != "hello" {
        t.Fatalf("want hello, got %s", got.Message)
    }
}
```

## Level 2：DX primitives，保留测试结构控制权

适合希望 IDE 仍然直接看到 `t.Run` / table-driven test 的场景。

```go
cases := []testx.Case[string, Result]{
    {
        Name:   "info",
        Input:  "INFO hello",
        Expect: Result{Level: "INFO", Message: "hello"},
    },
    {
        Name:   "debug",
        Input:  "DEBUG access",
        Expect: Result{Level: "DEBUG", Message: "access"},
    },
}

for _, c := range cases {
    t.Run(c.Name, func(t *testing.T) {
        got, err := Parse(c.Input)

        testx.Require(t, err).NoError()
        testx.Assert(t, got).Equal(c.Expect)
    })
}
```

`Case[I, O]` 是普通泛型数据结构，不要求必须交给 testx 执行。Level 2 推荐直接写 struct literal，以尽量保留 IDE 对 table-driven case 的静态识别；`testx.C(...)` 更适合追求简洁的 Level 3。

### Assert / Require

```go
testx.Assert(t, got).Equal(want)
testx.Assert(t, value).NotEqual(other)
testx.Assert(t, ptr).Nil()
testx.Assert(t, ptr).NotNil()
testx.Assert(t, ok).True()
testx.Assert(t, ok).False()
testx.Assert(t, xs).Len(3)
testx.Assert(t, xs).Empty()
testx.Assert(t, xs).Contains(item)
testx.Assert(t, text).Contains("hello")
testx.Assert(t, text).MatchRegexp(`^INFO`)

testx.Require(t, err).NoError()
testx.Require(t, err).Error()
testx.Require(t, err).ErrorIs(io.EOF)
testx.Require(t, err).ErrorContains("invalid")
```

- `Assert` 失败：`t.Errorf`，继续当前测试。
- `Require` 失败：`t.Fatalf`，立即结束当前测试。
- 会报告测试失败的 helper 都调用 `t.Helper()`，失败位置尽量指向业务测试代码。
- assertion 方法返回 `bool`，但通常不需要手动检查；测试失败已经通过 `testing.TB` 报告。

同一个测试中有多条断言时，可以先绑定 `testing.TB`，避免重复传入 `t`：

```go
check := testx.New(t)
check.Assert(got).Equal(want)
check.Assert(err).NoError()
check.Require(response).NotNil()
```

`New(t)` 的绑定方法接受不同类型的实际值，适合追求调用简洁；它会把失败继续转发到同一个 `t`。如果希望让 `actual` 和 `expected` 保持编译期同类型约束，继续使用泛型入口 `testx.Assert(t, got).Equal(want)`。

给失败信息增加业务上下文：

```go
testx.Assert(t, got).
    Because("loading user %q", userID).
    Equal(want)
```

错误类型断言与标准库 `errors.As` 使用相同的 target 规则：

```go
var pathErr *os.PathError
testx.Require(t, err).ErrorAs(&pathErr)
```

`Contains` / `NotContains` 支持字符串、数组、slice 和 map key；`ContainsAll` / `ElementsMatch` 只接受数组或 slice。`ElementsMatch` 会考虑重复元素，但忽略顺序。

如果函数返回一个字段很多的结构体，但当前只关心其中几个字段，可以直接对字段断言，不必比较整个结构体：

```go
func assertParseFields(
    t *testing.T,
    parser *StringLineLogParser,
    raw source.RawRecord,
    want logentry.LogEntry,
) {
    got, err := parser.Parse(raw)
    testx.Require(t, err).NoError()
    testx.Assert(t, got.Level).Equal(want.Level)
    testx.Assert(t, got.Message).Equal(want.Message)
}
```

在具体测试中传入你的 `RawRecord` fixture 和期望的 `LogEntry` 即可。这种写法不会因为 `LogEntry` 新增时间戳、来源、扩展字段而让测试无关地失败；如果要批量测试多条输入，也可以把 `Parse` 放进 `RunErr`，在 subject 中投影出 `Level` 和 `Message` 两个字段。

### go-cmp options

```go
testx.Assert(t, got).Equal(want,
    cmpopts.IgnoreFields(User{}, "UpdatedAt"),
)
```

Case 自身也可以携带比较规则：

```go
testx.C("empty", input, want).
    WithCmp(cmpopts.EquateEmpty())
```

## Level 3：隐藏样板代码

### `func(I) O`

```go
testx.Run(t, strings.ToUpper,
    testx.C("a", "hello", "HELLO"),
    testx.C("b", "world", "WORLD"),
)
```

内部仍然使用真正的 `t.Run` 创建子测试，但源码层面的 IDE case 静态识别可能不如手写 Level 2。

### `func(I) (O, error)`

```go
testx.RunErr(t, Parse,
    testx.C("info", "INFO hello", Result{
        Level: "INFO",
        Message: "hello",
    }),

    testx.C("invalid", "BAD hello", Result{}).
        WithError(testx.ErrorContains("invalid level")),
)
```

默认 error expectation 是 **NoError**。错误 Case 默认只验证错误，不比较返回值；需要验证部分返回值时使用 `.CompareOutputOnError()`。

Error expectation 的选择规则：

| API | 适用场景 |
| --- | --- |
| 默认值 | 必须没有错误 |
| `AnyError()` | 只关心是否发生错误 |
| `ErrorIs(target)` | 错误链中应包含已知 sentinel error |
| `ErrorContains(text)` | 只能通过稳定的错误文本片段识别 |
| `ErrorMatch(description, predicate)` | 需要自定义错误分类逻辑 |

完整示例：

```go
testx.RunErr(t, Decode,
    testx.C("eof", emptyInput, Result{}).
        WithError(testx.ErrorIs(io.EOF)),
    testx.C("partial result", badInput, partialResult).
        WithError(testx.ErrorMatch("validation error", func(err error) bool {
            return errors.Is(err, ErrValidation)
        })).
        CompareOutputOnError(),
)
```

`CompareOutputOnError` 只影响该 Case：先验证错误 expectation，随后继续使用 `go-cmp` 比较返回值。`ErrorMatch` 的 predicate 如果 panic，会转换成测试失败，不会让整个测试进程失控。

### Cases 复用

```go
cases := testx.Cases(
    testx.C("a", " hello ", "hello"),
    testx.C("b", " world ", "world"),
)

cases.Run(t, strings.TrimSpace)
// 或
cases.RunErr(t, someFuncReturningError)
```

### Preset 写法

如果你希望把“如何调用 subject”显式表示出来：

```go
testx.Func(strings.ToUpper).Run(t, cases...)

testx.FuncErr(Parse).Run(t,
    testx.C("info", "INFO hello", want),
)
```

Context subject 使用：

```go
func LoadUser(ctx context.Context, id string) (User, error) {
    // ...
}

testx.ContextFuncErr(LoadUser).Run(t,
    testx.C("found", userID, wantUser),
)
```

每个 Case 使用独立的可取消 context，Case 结束时自动取消。

## Interface 行为契约：Contract

编译期的“是否实现 interface”仍然应该直接写：

```go
var _ Store = (*RedisStore)(nil)
```

Contract 解决的是另一件事：**多个实现是否遵守相同的行为语义。**

```go
var StoreContract = testx.NewContract[Store]("store",
    testx.S("set then get", func(t *testing.T, store Store) {
        store.Set("foo", "bar")

        got, ok := store.Get("foo")
        testx.Require(t, ok).True()
        testx.Assert(t, got).Equal("bar")
    }),

    testx.S("missing key", func(t *testing.T, store Store) {
        _, ok := store.Get("missing")
        testx.Assert(t, ok).False()
    }),
)
```

然后：

```go
func TestStoreContract(t *testing.T) {
    StoreContract.VerifyAll(t,
        testx.Impl("memory", func(t *testing.T) Store {
            return NewMemoryStore()
        }),
        testx.Impl("redis", func(t *testing.T) Store {
            return NewTestRedisStore(t)
        }),
    )
}
```

会产生类似：

```text
TestStoreContract
├── memory
│   ├── set_then_get
│   └── missing_key
└── redis
    ├── set_then_get
    └── missing_key
```

**每一个 Spec 都重新调用 Factory**，避免有状态实现之间互相污染；Factory 收到真实 `*testing.T`，可以继续使用 `t.Cleanup()`、`t.TempDir()` 等标准能力。

Contract 同样保留 Level 2 能力：`Contract.Specs` 是公开数据，可以自己循环和组织 `t.Run`。

三种验证入口的区别：

| API | 产生的测试层级 |
| --- | --- |
| `contract.Verify(t, factory)` | 直接在当前测试下运行每个 Spec |
| `contract.VerifyAs(t, "memory", factory)` | 先创建 implementation 子测试，再运行每个 Spec |
| `contract.VerifyAll(t, implementations...)` | 对多个命名 implementation 生成完整测试树 |

Factory 或 Spec test 为 nil 时会得到带 Contract 名称的测试失败。即使 Contract 暂时没有 Spec，nil Factory 也会被拒绝。

## Case 控制

Level 3 Case 支持：

```go
testx.C("parallel", in, out).Parallel()
testx.C("todo", in, out).Skip("not implemented")
testx.C("custom compare", in, out).WithCmp(...)
testx.C("error", in, zero).WithError(testx.ErrorIs(ErrInvalid))
```

- `.Skip(reason)` 总会跳过 Case，即使 reason 是空字符串。
- `.Parallel()` 使用标准 `t.Parallel()`；subject、fixture 和共享数据仍需自行保证并发安全。
- 空 Case 名称会回退为 `case`。建议名称在同一测试内保持唯一，便于 `go test -run` 和 IDE 精确定位。
- Case 名称可以动态生成并正常运行，但 IDE 插件通常只能静态识别字符串字面量。

## Eventually

```go
testx.Eventually(t, func() bool {
    return queue.Len() == 1
}).Every(20 * time.Millisecond).Within(time.Second)
```

适合异步后端测试，避免手写 polling loop。

需要返回轮询结果时：

```go
value, ok := testx.EventuallyValue(t, func() (Item, bool) {
    item, found := cache.Get("key")
    return item, found
}).Every(20 * time.Millisecond).Within(time.Second)
```

`Every` 和 `Within` 都要求正数；轮询等待不会超过剩余 deadline。

`Within` 返回是否在 deadline 内满足条件，同时在失败时通过 `testing.TB` 报告错误。不要把 `Every` 设置成忙循环级别的极小间隔；它应该反映被测系统合理的可观察频率。

## HTTP preset

```go
testx.HTTP(handler).Run(t,
    testx.C("create",
        testx.HTTPRequest{
            Method: http.MethodPost,
            Target: "/items",
            Body:   []byte(`{"name":"book"}`),
        },
        testx.HTTPResponse{
            Status: http.StatusCreated,
            Body:   `{"id":"1"}`,
            Header: http.Header{"Content-Type": {"application/json"}},
        },
    ),
)
```

响应 Body 精确比较；期望 Header 是子集，只比较列出的 header。

HTTP 默认值和比较规则：

- 空 `HTTPRequest.Method` 使用 `GET`。
- 空 `HTTPRequest.Target` 使用 `/`。
- `HTTPResponse.Status == 0` 表示期望 `200 OK`。
- 请求 `Header` 会复制到 `http.Request`。
- 响应 Body 精确比较，不自动做 JSON 语义比较。
- 期望响应 Header 是子集；没有列出的实际 Header 不影响测试。
- handler 为 nil 或请求无法构造时，当前 Case 会立即失败。

## JSON、Panic 与 collection assertion

```go
testx.Assert(t, actualJSON).JSONEqual(expectedJSON)
testx.Assert(t, fn).Panics()
testx.Assert(t, fn).PanicsWith("boom")
testx.Assert(t, fn).NotPanics()

testx.Assert(t, values).NotContains(item)
testx.Assert(t, values).ContainsAll(subset)
testx.Assert(t, values).ElementsMatch(unordered)
testx.Assert(t, value).Because("user %s", id).Equal(want)
```

`JSONEqual` 忽略空白、对象 key 顺序和数字的等价表示，例如 `1` 与 `1.0` 相等。

输入可以是 `string`、`[]byte`、`json.RawMessage`，也可以是能被 `encoding/json` 编码的普通 Go 值。输入必须恰好包含一个合法 JSON 值；无效 JSON 或连续多个 JSON 值会报告测试失败。数组顺序仍然参与比较。

Panic assertion 只接受无参数、非 nil 函数；返回值可以忽略。`PanicsWith` 使用 `go-cmp` 比较 recover 到的值：

```go
testx.Assert(t, func() { panic(MyError{Code: 42}) }).
    PanicsWith(MyError{Code: 42})
```

## Golden / snapshot

```go
testx.GoldenString(t, "testdata/output.golden", output)
testx.SnapshotString(t, "response", output)
```

默认只读比较。设置 `TESTX_UPDATE_GOLDEN=1` 后，Golden 使用同目录临时文件替换目标；Snapshot 自动写入 `testdata/snapshots`。

常用工作流：

```powershell
$env:TESTX_UPDATE_GOLDEN = '1'
go test ./...
Remove-Item Env:TESTX_UPDATE_GOLDEN
go test ./...
```

```bash
TESTX_UPDATE_GOLDEN=1 go test ./...
go test ./...
```

- 默认模式下文件不存在会失败，并提示使用更新环境变量创建。
- `Snapshot(t, "response", ...)` 的路径由 `t.Name()` 和 snapshot 名称生成，位于 `testdata/snapshots`。
- 更新使用同目录临时文件替换，减少进程中断留下半写文件的风险。
- CI 应保持默认只读模式，并提交审核后的 golden 文件。

## CLI preset

```go
testx.CLI().Run(t,
    testx.C("version",
        testx.CLIRequest{Path: "go", Args: []string{"version"}},
        testx.CLIResult{ExitCode: 0, Stdout: "...\n"},
    ),
)
```

CLI 通过 `os/exec` 直接调用程序，不经过 shell。非零退出码保存在 `ExitCode`，超时保存在 `TimedOut`；启动失败才让 Case 失败。

完整请求还可以设置工作目录、环境变量、标准输入和 timeout：

```go
testx.CLI().Run(t,
    testx.C("stdin",
        testx.CLIRequest{
            Path:    executable,
            Args:    []string{"read"},
            Dir:     t.TempDir(),
            Env:     []string{"APP_ENV=test"},
            Stdin:   "hello\n",
            Timeout: 2 * time.Second,
        },
        testx.CLIResult{ExitCode: 0, Stdout: "hello\n"},
    ),
)
```

`Env` 会追加到当前进程环境；同名变量由后追加的值覆盖。timeout 后 `TimedOut` 为 true，输出仍包含进程终止前已捕获的内容。`Stdout` 和 `Stderr` 都是精确比较，可以通过 `.WithCmp(...)` 自定义规则。

## Benchmark / fuzz seeds

```go
func BenchmarkParse(b *testing.B) {
    testx.Benchmark(b, func(input string) { Parse(input) },
        testx.B("short", "INFO ok"),
        testx.B("long", strings.Repeat("x", 1024)),
    )
}

func FuzzParse(f *testing.F) {
    testx.FuzzSeed(f, "INFO ok")
    testx.FuzzSeeds(f, []any{"DEBUG x"}, []any{""})
    f.Fuzz(func(t *testing.T, input string) { Parse(input) })
}
```

多参数 fuzz target 的每个 seed 必须按参数顺序提供完整值：

```go
func FuzzLookup(f *testing.F) {
    testx.FuzzSeed(f, "users", int64(42))
    testx.FuzzSeeds(f,
        []any{"users", int64(1)},
        []any{"orders", int64(2)},
    )
    f.Fuzz(func(t *testing.T, table string, id int64) { /* ... */ })
}
```

`Benchmark` 会为每个 `B(name, input)` 建立子 benchmark、重置计时器并启用 allocation 统计。benchmark subject 为 nil、fuzz seed 为空时会直接失败。

## VS Code

[`go-testx/testx-vscode`](https://github.com/go-testx/testx-vscode) 提供 `Run case` / `Debug case` CodeLens。第一版识别标准 `func TestXxx(t *testing.T)` 中名称为字符串 literal 的直接 `testx.C(...)` 调用；动态名称和构造器别名无法静态识别。安装与开发见该仓库。

## GoLand

[`go-testx/testx-goland`](https://github.com/go-testx/testx-goland) 提供针对 GoLand 2026.2（build `262`）开发和验证的第一版插件。它识别标准测试函数中的直接字符串 Case：

```go
func TestParse(t *testing.T) {
    testx.RunErr(t, Parse,
        testx.C("valid", "INFO hello", want),
        testx.C("invalid", "BAD hello", Result{}),
    )
}
```

插件能力：

- 在 `testx.C("name", ...)` 的 Case 名称旁显示 GoLand 原生 Run / Debug 图标。
- 监听 GoLand 测试事件，把 Case gutter 图标更新为运行中、通过、失败或忽略状态；状态保存在当前项目会话内，下一次测试运行开始时重新收集。
- 使用 GoLand 原生 Go Test runner 执行 `TestParse/invalid`，保留测试树、失败重跑、Debug 和 Coverage。
- 支持默认导入名 `testx` 和显式 alias，例如 `tx "github.com/go-testx/testx"`。
- 正确转义 Case 名称中的正则字符，并保留 `/` 作为 Go 子测试层级分隔符。
- 对动态 Case 名称和重复 Case 名称显示弱警告 Inspection。
- 对 `Contract.VerifyAll` 中内联的 `testx.Impl("memory", ...)` 提供 implementation 级运行入口。
- 在 Golden / Snapshot 调用的编辑器菜单中提供带 `TESTX_UPDATE_GOLDEN=1` 的更新动作；如果调用位于静态 `testx.C("case", ...)` 内，会只更新当前 Case，否则更新整个测试函数。
- 把光标放在 `testx.Run`、别名 `x.Run`、`testx.RunErr` 或 `testx.Func(...).Run` 上，使用编辑器菜单的 **Generate testx.C cases...**，输入 `a,b,c` 就会把三个 Case 追加到剩余参数中。
- 生成器会解析 subject 的函数签名，默认填入可编译的空初始化，例如 `source.RawRecord{}` 和 `logentry.LogEntry{}`；输入模板和期望模板可以直接改成真实 fixture，也支持 `${name}` / `{{name}}` 占位符。
- 如果调用中已经有 Case，可以勾选克隆首个 Case 的输入、期望和 modifier，再批量替换名称。对 `RunErr` / `FuncErr`，错误期望下拉框提供“不要求错误”、“要求任意错误”、“错误消息包含文本”和 `errors.Is` 表达式。
- 在 Go 文件、函数/方法、类型、接口、测试文件或选区上使用 **Prepare testx test prompt...**，可以自动收集最小相关源码、已有测试和版本匹配的 testx 文档；用户可以填写要求/草稿、编辑预览并复制给任意 AI。该功能不联网、不调用模型，也不修改源码。
- 提供 `txcase`、`txrun`、`txrunerr`、`txcontract` Live Template。

当前插件支持字符串字面量、编译期 `const` 引用和简单字符串拼接，也支持测试函数内的 `Cases(...)` 以及同一 `_test.go` 文件中只被一个 `TestXxx` 使用的包级 CaseSet。普通变量、运行时拼接、自定义包装函数、跨文件共享或被多个测试使用的 CaseSet、dot import 仍暂不显示精确 Case gutter；这不影响测试实际运行。

构建和安装见 [`go-testx/testx-goland`](https://github.com/go-testx/testx-goland) 的 README。

生成器示例：

```go
func (p *StringLineLogParser) Parse(raw source.RawRecord) (logentry.LogEntry, error) {
    // ...
}

func TestParse(t *testing.T) {
    p := &StringLineLogParser{}
    testx.RunErr(t, p.Parse)
}
```

把光标放在 `RunErr` 行，选择 **Generate testx.C cases...**，输入 `valid,invalid` 后，默认会生成类似：

```go
testx.RunErr(t, p.Parse,
    testx.C("valid", source.RawRecord{}, logentry.LogEntry{}),
    testx.C("invalid", source.RawRecord{}, logentry.LogEntry{}),
)
```

这里的“错误期望”只控制 `RunErr` 收到的 `error`：默认“不要求错误”表示 `err == nil`；选择“任意错误”会生成 `.WithError(testx.AnyError())`，用于验证该 Case 必须失败。它不会替你猜测业务错误内容，`ErrorContains` 和 `errors.Is` 选项需要在对话框中提供具体文本或表达式。

## 设计原则

1. **testing 永远是底座。** testx 不重新实现 test runner。
2. **Primitive first, orchestration second.** `Case`、assertion、contract 都能脱离 Level 3 runner 使用。
3. **渐进式抽象。** 同一个项目甚至同一个 `_test.go` 可以同时使用三个 Level。
4. **复杂测试不强塞 DSL。** 一旦 declarative API 不合适，就回到 `t.Run`，仍然可以继续使用 testx 的 assertion。
5. **尽量编译期类型安全。** Runner / Case / Preset 使用泛型，避免通过反射识别函数签名。

## 当前 API

```text
Case[I, O]
C(name, input, expect)
Cases(...)
Case.WithCmp(...)
Case.WithError(...)
Case.CompareOutputOnError()
Case.Parallel()
Case.Skip(...)

Assert(t, actual)
Require(t, actual)
  .Because(...)
  .Equal / .NotEqual
  .Nil / .NotNil
  .True / .False
  .Len / .Empty
  .Contains / .NotContains / .ContainsAll / .ElementsMatch
  .MatchRegexp
  .NoError / .Error / .ErrorIs / .ErrorAs / .ErrorContains
  .JSONEqual
  .Panics / .PanicsWith / .NotPanics

Run(t, func(I) O, cases...)
RunErr(t, func(I) (O, error), cases...)

Func(fn).Run(...)
FuncErr(fn).Run(...)
ContextFuncErr(fn).Run(...)
HTTP(handler).Run(...)
CLI().Run(...)

NewContract[T](...)
S(name, spec)
Impl(name, factory)
Contract.Verify(...)
Contract.VerifyAs(...)
Contract.VerifyAll(...)

Eventually(...).Every(...).Within(...)
EventuallyValue(...).Every(...).Within(...)

Golden / GoldenString / Snapshot / SnapshotString
Benchmark / B
FuzzSeed / FuzzSeeds
```
