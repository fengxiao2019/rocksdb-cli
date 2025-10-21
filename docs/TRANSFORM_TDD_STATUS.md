# Transform Feature - TDD Status

## 📊 当前进度

### ✅ 已完成

1. **开发计划** (`docs/TRANSFORM_FEATURE_PLAN.md`)
   - 完整的功能需求和设计
   - 8天开发里程碑
   - 技术选型和架构设计
   - 安全考虑和文档结构

2. **测试框架搭建**
   - `internal/transform/` 目录结构
   - 类型定义 (`types.go`)
   - 测试文件（failing tests）
   - 骨架实现（stub implementations）

3. **测试用例编写**
   - ✅ `python_executor_test.go` - 13个测试用例
   - ✅ `transform_test.go` - 11个测试用例
   - 总计：24个测试用例（目前全部 FAIL）

### 📝 测试用例清单

#### Python Executor Tests (13)
- [x] TestPythonExecutor_SimpleExpression (4 sub-tests)
  - uppercase string
  - lowercase string
  - string concatenation
  - numeric operation
- [x] TestPythonExecutor_JSONParsing (3 sub-tests)
  - parse and modify JSON field
  - add new JSON field
  - filter JSON by field
- [x] TestPythonExecutor_FilterExpression (3 sub-tests)
  - filter by key prefix
  - filter by value content
  - filter by JSON field
- [x] TestPythonExecutor_ErrorHandling (4 sub-tests)
  - syntax error
  - undefined variable
  - type error
  - import error
- [x] TestPythonExecutor_ValidateExpression (4 sub-tests)
  - valid expression
  - valid multiline expression
  - invalid syntax
  - empty expression
- [x] TestPythonExecutor_ScriptFile (skipped)
- [x] TestPythonExecutor_Timeout
- [x] TestPythonExecutor_MemoryLimit (skipped)

#### Transform Processor Tests (11)
- [x] TestTransformProcessor_DryRun
- [x] TestTransformProcessor_BasicTransform
- [x] TestTransformProcessor_WithFilter (3 sub-tests)
  - filter by key prefix
  - filter all
  - filter none
- [x] TestTransformProcessor_BatchProcessing
- [x] TestTransformProcessor_ErrorRecovery
- [x] TestTransformProcessor_Limit
- [x] TestTransformProcessor_KeyTransform
- [x] TestTransformProcessor_Statistics
- [x] TestTransformProcessor_ProgressCallback (skipped)
- [x] TestTransformProcessor_ConcurrentSafety (skipped)
- [x] TestTransformProcessor_MemoryUsage (skipped)

## 🔄 TDD 流程

### Phase 1: Red (当前阶段) ✅
- [x] 编写失败的测试用例
- [x] 创建类型定义和接口
- [x] 创建骨架实现（返回 "not implemented" 错误）
- [x] 验证测试可以编译并失败

### Phase 2: Green (下一步)
- [ ] 实现 PythonExecutor
  - [ ] 选择 Python 集成方案
  - [ ] 实现表达式执行
  - [ ] 实现错误处理
  - [ ] 让 python_executor_test.go 通过
- [ ] 实现 TransformProcessor
  - [ ] 实现基础转换逻辑
  - [ ] 实现 dry-run 模式
  - [ ] 实现批处理
  - [ ] 让 transform_test.go 通过

### Phase 3: Refactor
- [ ] 优化代码结构
- [ ] 改进性能
- [ ] 增强错误处理
- [ ] 添加文档注释

## 🎯 下一步行动

### 立即开始（今天）
1. **运行测试验证框架**
   ```bash
   cd internal/transform
   go test -v
   ```
   预期：所有测试失败，输出 "not implemented" 错误

2. **选择 Python 集成方案**
   - 推荐：外部 Python 进程（简单、安全）
   - 备选：Starlark（纯 Go、沙箱）
   
3. **实现 PythonExecutor 基础版本**
   - 实现 `ExecuteExpression` 使用 `exec.Command`
   - 处理标准输入/输出
   - 实现基础错误处理

### 本周内完成
4. **让前 5 个测试通过**
   - TestPythonExecutor_SimpleExpression
   - TestPythonExecutor_JSONParsing
   - TestPythonExecutor_FilterExpression
   - TestPythonExecutor_ErrorHandling
   - TestPythonExecutor_ValidateExpression

5. **实现 TransformProcessor MVP**
   - DryRun 模式
   - 基础转换
   - 简单统计

## 📈 测试覆盖率目标

### Milestone 1 (MVP)
- [ ] 核心功能测试覆盖率 > 80%
- [ ] 所有主要功能有至少 1 个测试

### Milestone 2 (完整功能)
- [ ] 整体测试覆盖率 > 85%
- [ ] 边界情况测试完善
- [ ] 性能测试基准

### Milestone 3 (生产就绪)
- [ ] 整体测试覆盖率 > 90%
- [ ] 集成测试完整
- [ ] 压力测试和并发测试

## 🔍 当前测试运行方法

```bash
# 运行所有 transform 测试
cd /Users/daoma/wkspace/rocksdb-cli
go test ./internal/transform/... -v

# 运行特定测试
go test ./internal/transform/... -v -run TestPythonExecutor_SimpleExpression

# 查看测试覆盖率
go test ./internal/transform/... -cover

# 生成覆盖率报告
go test ./internal/transform/... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## 📚 参考资料

### Python 集成方案
1. **外部进程方案**
   - `os/exec` 包文档
   - Python subprocess 最佳实践

2. **Starlark 方案**
   - https://github.com/google/starlark-go
   - Starlark 语言规范

### TDD 最佳实践
- 先写最简单的测试
- 一次只让一个测试通过
- 频繁重构
- 保持测试独立

## 🎉 成功标准

### MVP 成功标准（3天内）
- [ ] 至少 10 个测试通过
- [ ] 可以执行简单的 Python 表达式
- [ ] Dry-run 模式工作正常
- [ ] 基础错误处理完善

### 完整功能成功标准（7天内）
- [ ] 所有非性能测试通过
- [ ] 支持脚本文件
- [ ] 批处理和过滤工作正常
- [ ] 命令行集成完成

### 生产就绪标准（10天内）
- [ ] 所有测试通过
- [ ] 性能达标
- [ ] 文档完整
- [ ] 代码审查通过
