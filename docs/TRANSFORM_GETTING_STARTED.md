# Transform Feature - 开发入门指南

## 🎯 当前状态：TDD Red Phase ✅

### 测试运行结果总结

```
Total Tests: 24
- ✅ PASS: 6 tests (错误处理测试按预期通过)
- ❌ FAIL: 15 tests (预期失败 - "not implemented")
- ⏭️  SKIP: 3 tests (高级功能，稍后实现)

覆盖率: 0% (骨架代码)
```

### 测试分类

#### ✅ 通过的测试 (6)
这些测试通过是因为它们测试的是"应该返回错误"的场景：
- TestPythonExecutor_ErrorHandling (4 sub-tests)
- TestPythonExecutor_ValidateExpression (2 sub-tests)
- TestPythonExecutor_Timeout

#### ❌ 失败的测试 (15) - **这是正确的！**
这些测试失败是因为功能还未实现（TDD Red Phase）：
- TestPythonExecutor_SimpleExpression (4 sub-tests)
- TestPythonExecutor_JSONParsing (3 sub-tests)
- TestPythonExecutor_FilterExpression (3 sub-tests)
- TestTransformProcessor_DryRun
- TestTransformProcessor_BasicTransform
- TestTransformProcessor_WithFilter (3 sub-tests)
- TestTransformProcessor_BatchProcessing
- TestTransformProcessor_ErrorRecovery
- TestTransformProcessor_Limit
- TestTransformProcessor_KeyTransform
- TestTransformProcessor_Statistics

#### ⏭️ 跳过的测试 (3)
这些是高级功能，等基础实现完成后再做：
- TestPythonExecutor_ScriptFile
- TestPythonExecutor_MemoryLimit
- TestTransformProcessor_ProgressCallback
- TestTransformProcessor_ConcurrentSafety
- TestTransformProcessor_MemoryUsage

## 🚀 下一步：进入 TDD Green Phase

### Phase 1: 实现 Python Executor (优先级最高)

#### 1.1 技术选型决策
需要选择 Python 集成方案：

**方案 A: 外部 Python 进程（推荐 MVP）**
```go
// 优点：简单、安全、跨平台
// 缺点：性能开销、需要系统有 Python

cmd := exec.Command("python3", "-c", expr)
// 通过 stdin/stdout 传递数据
```

**方案 B: Starlark（推荐生产）**
```go
// 优点：纯 Go、安全沙箱、无需外部依赖
// 缺点：不是完整 Python、需要学习新 API

import "go.starlark.net/starlark"
```

#### 1.2 实现步骤
1. **简单表达式执行** (让前 4 个测试通过)
   ```bash
   # 目标：TestPythonExecutor_SimpleExpression 全部通过
   go test -v -run TestPythonExecutor_SimpleExpression
   ```

2. **JSON 处理** (让 JSON 测试通过)
   ```bash
   # 目标：TestPythonExecutor_JSONParsing 全部通过
   go test -v -run TestPythonExecutor_JSONParsing
   ```

3. **过滤表达式** (让过滤测试通过)
   ```bash
   # 目标：TestPythonExecutor_FilterExpression 全部通过
   go test -v -run TestPythonExecutor_FilterExpression
   ```

4. **表达式验证** (让验证测试通过)
   ```bash
   # 目标：TestPythonExecutor_ValidateExpression 剩余测试通过
   go test -v -run TestPythonExecutor_ValidateExpression
   ```

### Phase 2: 实现 Transform Processor

#### 2.1 Dry-Run 模式
```go
// 实现目标：
// 1. 不修改数据库
// 2. 返回预览数据
// 3. 统计信息准确

go test -v -run TestTransformProcessor_DryRun
```

#### 2.2 基础转换
```go
// 实现目标：
// 1. 遍历 column family
// 2. 应用转换表达式
// 3. 写回数据库

go test -v -run TestTransformProcessor_BasicTransform
```

#### 2.3 过滤和限制
```go
// 实现目标：
// 1. 支持过滤表达式
// 2. 支持 limit 参数

go test -v -run TestTransformProcessor_WithFilter
go test -v -run TestTransformProcessor_Limit
```

## 📝 开发工作流

### 每次开发循环

```bash
# 1. 选择一个失败的测试
go test -v -run TestPythonExecutor_SimpleExpression

# 2. 编写最小实现让测试通过
# 编辑 internal/transform/python_executor.go

# 3. 运行测试验证
go test -v -run TestPythonExecutor_SimpleExpression

# 4. 如果通过，重构代码（保持测试通过）

# 5. 提交代码
git add internal/transform/
git commit -m "feat: implement simple expression execution"

# 6. 重复下一个测试
```

### 持续验证

```bash
# 运行所有测试
go test ./internal/transform/... -v

# 查看覆盖率
go test ./internal/transform/... -cover

# 生成覆盖率报告
go test ./internal/transform/... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

## 🏗️ 实现建议

### Python Executor 实现模板（外部进程）

```go
func (e *pythonExecutor) ExecuteExpression(expr string, context map[string]interface{}) (interface{}, error) {
    // 1. 准备 Python 脚本
    script := e.buildScript(expr, context)
    
    // 2. 创建命令
    ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
    defer cancel()
    
    cmd := exec.CommandContext(ctx, "python3", "-c", script)
    
    // 3. 执行并获取输出
    output, err := cmd.Output()
    if err != nil {
        return nil, e.parseError(err)
    }
    
    // 4. 解析结果
    return e.parseOutput(output), nil
}

func (e *pythonExecutor) buildScript(expr string, context map[string]interface{}) string {
    // 构建完整的 Python 脚本
    // 1. 导入必要的模块
    // 2. 设置上下文变量
    // 3. 执行表达式
    // 4. 打印结果
}
```

### Transform Processor 实现模板

```go
func (p *transformProcessor) Process(cf string, opts TransformOptions) (*TransformResult, error) {
    // 1. 验证选项
    if err := validateOptions(opts); err != nil {
        return nil, err
    }
    
    // 2. 初始化结果
    result := &TransformResult{
        StartTime: time.Now(),
    }
    
    // 3. 获取迭代器
    iterator := p.db.NewIterator(cf)
    defer iterator.Close()
    
    // 4. 遍历并处理
    for iterator.SeekToFirst(); iterator.Valid(); iterator.Next() {
        key := iterator.Key()
        value := iterator.Value()
        
        // 4.1 应用过滤
        if !p.shouldProcess(key, value, opts) {
            result.Skipped++
            continue
        }
        
        // 4.2 应用转换
        newKey, newValue, err := p.transform(key, value, opts)
        if err != nil {
            result.Errors = append(result.Errors, TransformError{...})
            continue
        }
        
        // 4.3 写入（如果不是 dry-run）
        if !opts.DryRun {
            p.db.Put(cf, newKey, newValue)
            result.Modified++
        } else {
            result.DryRunData = append(result.DryRunData, DryRunEntry{...})
        }
        
        result.Processed++
        
        // 4.4 检查 limit
        if opts.Limit > 0 && result.Processed >= opts.Limit {
            break
        }
    }
    
    // 5. 完成统计
    result.EndTime = time.Now()
    result.Duration = result.EndTime.Sub(result.StartTime)
    
    return result, nil
}
```

## 📊 进度跟踪

### Milestone 1: Python Executor (目标：3天)
- [ ] Day 1: SimpleExpression + JSONParsing
- [ ] Day 2: FilterExpression + ValidateExpression
- [ ] Day 3: 错误处理优化 + 单元测试全部通过

### Milestone 2: Transform Processor (目标：2天)
- [ ] Day 4: DryRun + BasicTransform
- [ ] Day 5: WithFilter + Limit + Statistics

### Milestone 3: 集成和命令行 (目标：2天)
- [ ] Day 6: 命令行参数解析 + Help
- [ ] Day 7: 集成测试 + 文档

### Milestone 4: 高级功能 (目标：1天)
- [ ] Day 8: ScriptFile + 性能优化

## 🎓 TDD 最佳实践提醒

1. **一次只做一件事**
   - 选择一个失败的测试
   - 只写能让它通过的代码
   - 不要提前实现还没测试的功能

2. **保持测试快速**
   - 单元测试应该秒级完成
   - 使用 mock 避免真实 I/O
   - 需要数据库的测试使用内存数据库

3. **频繁运行测试**
   - 每次修改后立即运行
   - 使用 `-run` 只运行相关测试
   - 定期运行完整测试套件

4. **先让测试通过，再优化**
   - Red → Green → Refactor
   - 不要在 Red 阶段优化
   - 重构时保持测试绿色

## 🤝 协作建议

### Git Workflow
```bash
# 每个功能一个分支
git checkout -b feature/transform-python-executor

# 频繁提交
git commit -m "test: add simple expression tests"
git commit -m "feat: implement simple expression execution"
git commit -m "refactor: improve error handling"

# 合并前确保所有测试通过
go test ./internal/transform/... -v
git push origin feature/transform-python-executor
```

### Code Review Checklist
- [ ] 所有新代码都有测试
- [ ] 所有测试都通过
- [ ] 测试覆盖率 > 80%
- [ ] 代码有适当的注释
- [ ] 错误处理完善
- [ ] 没有硬编码的测试数据路径

## 📚 参考资源

### Go 测试
- [Go Testing Documentation](https://pkg.go.dev/testing)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)

### Python 集成
- [exec.Command Documentation](https://pkg.go.dev/os/exec)
- [Starlark Go](https://github.com/google/starlark-go)

### TDD
- [Test-Driven Development](https://martinfowler.com/bliki/TestDrivenDevelopment.html)
- [The Three Rules of TDD](http://butunclebob.com/ArticleS.UncleBob.TheThreeRulesOfTdd)

## 🎯 成功的标志

当你看到这个输出时，Phase 1 就成功了：

```bash
$ go test ./internal/transform/... -v
...
ok      rocksdb-cli/internal/transform  0.5s    coverage: 85.2% of statements
```

祝开发顺利！🚀
