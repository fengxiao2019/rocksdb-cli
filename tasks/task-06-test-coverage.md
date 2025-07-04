# Task 06: 测试覆盖完善

**优先级**: 🟡 中优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 4-5小时  
**负责人**: 待分配  

## 问题描述

当前项目测试覆盖存在不足：

### 发现的问题
1. **缺乏集成测试**:
   - 只有单元测试，缺少端到端测试
   - 模块间交互未充分测试
   - 真实场景覆盖不足

2. **错误路径测试不全**:
   - 主要测试正常流程
   - 异常情况和边界条件测试不足
   - 错误恢复机制未测试

3. **性能测试缺失**:
   - 无基准测试
   - 内存使用量未监控
   - 并发性能未验证

4. **测试数据管理**:
   - 测试数据创建繁琐
   - 缺乏测试数据清理
   - 测试间数据污染

## 影响分析

- **代码质量**: 隐藏的bug可能在生产环境暴露
- **重构信心**: 缺乏完整测试覆盖导致重构风险高
- **性能回退**: 无法及时发现性能问题
- **维护成本**: 手动测试工作量大

## 解决方案

### 1. 完善测试体系
```
tests/                    # 测试目录重组
├── unit/                 # 单元测试
├── integration/          # 集成测试
├── performance/          # 性能测试
├── fixtures/             # 测试数据
├── helpers/              # 测试工具
└── scenarios/            # 场景测试
```

### 2. 测试类型分类
- **单元测试**: 函数级别的测试
- **集成测试**: 模块间交互测试
- **端到端测试**: 完整功能流程测试
- **性能测试**: 基准测试和压力测试
- **错误注入测试**: 异常场景测试

## 实施步骤

### Phase 1: 测试基础设施 (1.5小时)
1. 重构测试目录结构
2. 创建测试数据管理工具
3. 实现测试辅助函数
4. 设置测试环境隔离

### Phase 2: 集成测试 (2小时)
1. 数据库集成测试
2. 命令执行集成测试
3. REPL交互测试
4. 文件导入导出测试

### Phase 3: 性能和压力测试 (1.5小时)
1. 数据库操作基准测试
2. 大数据量性能测试
3. 并发访问测试
4. 内存使用量测试

## 验收标准

### 覆盖率要求
- [ ] 单元测试覆盖率 > 85%
- [ ] 集成测试覆盖主要功能流程
- [ ] 错误路径测试覆盖 > 70%
- [ ] 性能基准测试建立

### 质量要求
- [ ] 测试用例可读性和可维护性
- [ ] 测试数据管理规范
- [ ] 测试执行时间 < 2分钟
- [ ] 测试稳定性 > 99%

### 功能要求
- [ ] 自动化测试流程
- [ ] 测试报告生成
- [ ] 持续集成集成
- [ ] 测试数据隔离

## 测试计划

### 单元测试增强
```go
// 测试辅助函数
func setupTestDB(t *testing.T) (*db.DB, func()) {
    tmpDir := t.TempDir()
    testDB, err := db.Open(tmpDir)
    require.NoError(t, err)
    
    return testDB, func() {
        testDB.Close()
    }
}

// 错误注入测试
func TestDB_GetCF_Errors(t *testing.T) {
    tests := []struct {
        name        string
        cf          string
        key         string
        setup       func(*db.DB)
        expectedErr error
    }{
        {
            name:        "column family not found",
            cf:          "nonexistent",
            key:         "test",
            expectedErr: db.ErrColumnFamilyNotFound,
        },
        {
            name: "key not found",
            cf:   "default",
            key:  "nonexistent",
            setup: func(d *db.DB) {
                // 确保CF存在但key不存在
            },
            expectedErr: db.ErrKeyNotFound,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db, cleanup := setupTestDB(t)
            defer cleanup()
            
            if tt.setup != nil {
                tt.setup(db)
            }
            
            _, err := db.GetCF(tt.cf, tt.key)
            assert.ErrorIs(t, err, tt.expectedErr)
        })
    }
}
```

### 集成测试示例
```go
func TestEndToEnd_CRUD_Operations(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    handler := &command.Handler{DB: db, State: &command.ReplState{CurrentCF: "default"}}
    
    // 创建列族
    result := captureOutput(func() {
        handler.Execute("createcf users")
    })
    assert.Contains(t, result, "OK")
    
    // 切换列族
    handler.Execute("usecf users")
    
    // 插入数据
    result = captureOutput(func() {
        handler.Execute("put user:1 {\"name\":\"Alice\",\"age\":30}")
    })
    assert.Contains(t, result, "OK")
    
    // 查询数据
    result = captureOutput(func() {
        handler.Execute("get user:1")
    })
    assert.Contains(t, result, "Alice")
    
    // JSON查询
    result = captureOutput(func() {
        handler.Execute("jsonquery name Alice")
    })
    assert.Contains(t, result, "user:1")
    
    // 导出数据
    tmpFile := filepath.Join(t.TempDir(), "export.csv")
    handler.Execute(fmt.Sprintf("export %s", tmpFile))
    
    // 验证导出文件
    content, err := os.ReadFile(tmpFile)
    require.NoError(t, err)
    assert.Contains(t, string(content), "user:1")
}
```

### 性能测试示例
```go
func BenchmarkDB_GetCF(b *testing.B) {
    db, cleanup := setupBenchDB(b)
    defer cleanup()
    
    // 预填充数据
    for i := 0; i < 1000; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := fmt.Sprintf("value_%d", i)
        db.PutCF("default", key, value)
    }
    
    b.ResetTimer()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            key := fmt.Sprintf("key_%d", rand.Intn(1000))
            _, err := db.GetCF("default", key)
            if err != nil {
                b.Error(err)
            }
        }
    })
}

func TestDB_Memory_Usage(t *testing.T) {
    var m1, m2 runtime.MemStats
    
    runtime.GC()
    runtime.ReadMemStats(&m1)
    
    // 执行大量操作
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    for i := 0; i < 10000; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := strings.Repeat("x", 1000) // 1KB per value
        db.PutCF("default", key, value)
    }
    
    runtime.GC()
    runtime.ReadMemStats(&m2)
    
    memUsage := m2.Alloc - m1.Alloc
    t.Logf("Memory usage: %d bytes", memUsage)
    
    // 验证内存使用在合理范围内
    assert.Less(t, memUsage, uint64(50*1024*1024)) // < 50MB
}
```

### 错误注入测试
```go
func TestErrorRecovery(t *testing.T) {
    tests := []struct {
        name     string
        scenario func(*testing.T, *db.DB)
    }{
        {
            name: "database corruption recovery",
            scenario: func(t *testing.T, db *db.DB) {
                // 模拟数据库损坏
                db.Close()
                // 重新打开应该能恢复或给出清晰错误
            },
        },
        {
            name: "disk full simulation",
            scenario: func(t *testing.T, db *db.DB) {
                // 模拟磁盘空间不足
                // 验证错误处理
            },
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            db, cleanup := setupTestDB(t)
            defer cleanup()
            tt.scenario(t, db)
        })
    }
}
```

## 测试工具和辅助函数

### 测试数据生成器
```go
type TestDataGenerator struct {
    db *db.DB
}

func (g *TestDataGenerator) CreateUsers(count int) error {
    for i := 0; i < count; i++ {
        key := fmt.Sprintf("user:%d", i+1)
        value := fmt.Sprintf(`{"id":%d,"name":"User%d","age":%d}`, 
            i+1, i+1, 20+i%50)
        if err := g.db.PutCF("users", key, value); err != nil {
            return err
        }
    }
    return nil
}

func (g *TestDataGenerator) CreateTimeSeries(count int) error {
    now := time.Now()
    for i := 0; i < count; i++ {
        timestamp := now.Add(-time.Duration(i) * time.Minute)
        key := fmt.Sprintf("%d", timestamp.Unix())
        value := fmt.Sprintf(`{"timestamp":%d,"value":%f}`, 
            timestamp.Unix(), rand.Float64()*100)
        if err := g.db.PutCF("metrics", key, value); err != nil {
            return err
        }
    }
    return nil
}
```

### 测试断言扩展
```go
func AssertJSONEqual(t *testing.T, expected, actual string) {
    var expectedJSON, actualJSON interface{}
    
    err := json.Unmarshal([]byte(expected), &expectedJSON)
    require.NoError(t, err, "expected JSON is invalid")
    
    err = json.Unmarshal([]byte(actual), &actualJSON)
    require.NoError(t, err, "actual JSON is invalid")
    
    assert.Equal(t, expectedJSON, actualJSON)
}

func AssertCSVContains(t *testing.T, csvContent, expectedRow string) {
    lines := strings.Split(csvContent, "\n")
    for _, line := range lines {
        if strings.Contains(line, expectedRow) {
            return
        }
    }
    t.Errorf("CSV does not contain expected row: %s", expectedRow)
}
```

## CI/CD 集成

### GitHub Actions 更新
```yaml
name: Test Coverage

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: 1.21
    
    - name: Install dependencies
      run: make deps
    
    - name: Run unit tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Run integration tests
      run: go test -v -tags=integration ./tests/integration/...
    
    - name: Run performance tests
      run: go test -v -bench=. -benchmem ./tests/performance/...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out
```

## 风险评估

**低风险**
- 测试改进不影响生产代码
- 可以渐进式添加测试用例
- 测试失败不影响功能

**潜在风险**
- 测试执行时间可能过长
- 测试数据管理复杂性增加

## 后续任务

- Task 01: 配置管理（测试配置管理）
- Task 02: 日志系统（测试日志输出）
- Task 03: 命令架构（测试新架构）

## 参考资料

- [Go测试最佳实践](https://golang.org/doc/tutorial/add-a-test)
- [表驱动测试](https://github.com/golang/go/wiki/TableDrivenTests)
- [集成测试策略](https://martinfowler.com/articles/practical-test-pyramid.html)
- [性能测试指南](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go) 