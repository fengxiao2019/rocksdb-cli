# Task 07: 并发安全性优化

**优先级**: 🟢 低优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 3-4小时  
**负责人**: 待分配  

## 问题描述

当前项目在并发安全性方面存在潜在问题：

### 发现的问题
1. **映射并发访问不安全**:
   ```go
   type DB struct {
       cfHandles map[string]*grocksdb.ColumnFamilyHandle  // map不是并发安全的
   }
   ```

2. **状态共享缺乏保护**:
   ```go
   type ReplState struct {
       CurrentCF string  // 可能被多个goroutine访问
   }
   ```

3. **资源竞争潜在风险**:
   - 数据库关闭时可能有其他goroutine在访问
   - 缺乏读写锁保护
   - 没有并发访问检测

4. **错误处理的并发问题**:
   - 错误状态可能在并发环境下不一致
   - 缺乏线程安全的错误累积机制

## 影响分析

- **数据一致性**: 并发访问可能导致数据不一致
- **程序稳定性**: 竞态条件可能导致程序崩溃
- **性能**: 不当的锁使用可能影响性能
- **可扩展性**: 限制了多goroutine场景的应用

## 解决方案

### 1. 并发安全架构
```
internal/sync/
├── safe_map.go         # 并发安全的映射
├── rwmutex.go          # 读写锁包装器
├── atomic.go           # 原子操作工具
└── detector.go         # 竞态检测器
```

### 2. 线程安全的数据结构
```go
type SafeDB struct {
    db        *grocksdb.DB
    cfHandles *sync.Map  // 使用sync.Map替代map
    stateMu   sync.RWMutex
    closed    int32      // 原子操作
}
```

### 3. 并发控制策略
- 读写分离锁
- 原子操作优化
- 无锁数据结构
- 优雅关闭机制

## 实施步骤

### Phase 1: 并发安全基础设施 (1.5小时)
1. 创建线程安全的映射包装器
2. 实现原子操作工具
3. 添加读写锁管理器
4. 创建竞态检测工具

### Phase 2: 数据库层并发优化 (1.5小时)
1. 重构DB结构使用线程安全组件
2. 添加操作级别的并发控制
3. 实现优雅关闭机制
4. 优化资源访问模式

### Phase 3: 应用层并发安全 (1小时)
1. 保护共享状态访问
2. 优化命令处理并发性
3. 添加并发性能测试
4. 实现并发监控

## 验收标准

### 功能要求
- [ ] 所有共享数据结构并发安全
- [ ] 数据库操作线程安全
- [ ] 优雅关闭机制完善
- [ ] 竞态条件检测通过

### 性能要求
- [ ] 并发性能不低于单线程95%
- [ ] 锁争用时间 < 1ms
- [ ] 支持至少100个并发连接
- [ ] 内存开销增加 < 10%

### 质量要求
- [ ] 并发安全测试覆盖率 > 80%
- [ ] 压力测试通过
- [ ] 竞态检测器无报告
- [ ] 死锁检测通过

## 测试计划

1. **并发安全测试**
   - 多goroutine数据竞争测试
   - 读写并发测试
   - 资源关闭竞态测试

2. **性能测试**
   - 并发吞吐量测试
   - 锁争用性能测试
   - 扩展性测试

3. **压力测试**
   - 高并发长时间运行
   - 内存泄漏检测
   - 系统资源监控

## 实现示例

### 线程安全的DB包装器
```go
type SafeDB struct {
    db        *grocksdb.DB
    cfHandles *sync.Map  // 线程安全的map
    options   *SafeOptions
    closed    int32
    closeMu   sync.Mutex
    activeOps int64  // 活跃操作计数
}

type SafeOptions struct {
    ro *grocksdb.ReadOptions
    wo *grocksdb.WriteOptions
    mu sync.RWMutex
}

func (d *SafeDB) GetCF(cf, key string) (string, error) {
    // 增加活跃操作计数
    atomic.AddInt64(&d.activeOps, 1)
    defer atomic.AddInt64(&d.activeOps, -1)
    
    // 检查是否已关闭
    if atomic.LoadInt32(&d.closed) == 1 {
        return "", ErrDatabaseClosed
    }
    
    // 获取列族句柄
    handleInterface, ok := d.cfHandles.Load(cf)
    if !ok {
        return "", NewColumnFamilyNotFoundError(cf)
    }
    
    handle := handleInterface.(*grocksdb.ColumnFamilyHandle)
    
    // 读取操作
    d.options.mu.RLock()
    val, err := d.db.GetCF(d.options.ro, handle, []byte(key))
    d.options.mu.RUnlock()
    
    if err != nil {
        return "", WrapDatabaseError(err, "get", map[string]interface{}{
            "column_family": cf,
            "key": key,
        })
    }
    defer val.Free()
    
    if !val.Exists() {
        return "", NewKeyNotFoundError(cf, key)
    }
    
    return string(val.Data()), nil
}

func (d *SafeDB) Close() error {
    d.closeMu.Lock()
    defer d.closeMu.Unlock()
    
    // 防止重复关闭
    if !atomic.CompareAndSwapInt32(&d.closed, 0, 1) {
        return ErrAlreadyClosed
    }
    
    // 等待所有活跃操作完成
    for atomic.LoadInt64(&d.activeOps) > 0 {
        time.Sleep(10 * time.Millisecond)
    }
    
    // 清理资源
    var errors []error
    
    d.cfHandles.Range(func(key, value interface{}) bool {
        handle := value.(*grocksdb.ColumnFamilyHandle)
        if err := d.safeDestroy(handle.Destroy); err != nil {
            errors = append(errors, fmt.Errorf("failed to destroy CF %s: %w", key, err))
        }
        return true
    })
    
    d.options.mu.Lock()
    if d.options.ro != nil {
        d.safeDestroy(d.options.ro.Destroy)
    }
    if d.options.wo != nil {
        d.safeDestroy(d.options.wo.Destroy)
    }
    d.options.mu.Unlock()
    
    if d.db != nil {
        d.db.Close()
    }
    
    if len(errors) > 0 {
        return fmt.Errorf("cleanup errors: %v", errors)
    }
    return nil
}
```

### 并发安全的状态管理
```go
type SafeReplState struct {
    currentCF string
    mu        sync.RWMutex
    history   []string
    historyMu sync.Mutex
}

func (s *SafeReplState) SetCurrentCF(cf string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.currentCF = cf
    
    // 添加到历史记录
    s.historyMu.Lock()
    s.history = append(s.history, cf)
    if len(s.history) > 100 { // 限制历史记录大小
        s.history = s.history[1:]
    }
    s.historyMu.Unlock()
}

func (s *SafeReplState) GetCurrentCF() string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return s.currentCF
}

func (s *SafeReplState) GetHistory() []string {
    s.historyMu.Lock()
    defer s.historyMu.Unlock()
    
    // 返回副本防止外部修改
    history := make([]string, len(s.history))
    copy(history, s.history)
    return history
}
```

### 并发安全的错误收集器
```go
type ConcurrentErrorCollector struct {
    errors []error
    mu     sync.Mutex
}

func (c *ConcurrentErrorCollector) Add(err error) {
    if err == nil {
        return
    }
    
    c.mu.Lock()
    defer c.mu.Unlock()
    c.errors = append(c.errors, err)
}

func (c *ConcurrentErrorCollector) GetErrors() []error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if len(c.errors) == 0 {
        return nil
    }
    
    // 返回副本
    errors := make([]error, len(c.errors))
    copy(errors, c.errors)
    return errors
}

func (c *ConcurrentErrorCollector) HasErrors() bool {
    c.mu.Lock()
    defer c.mu.Unlock()
    return len(c.errors) > 0
}
```

### 并发测试示例
```go
func TestDB_ConcurrentAccess(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    // 预填充数据
    for i := 0; i < 100; i++ {
        key := fmt.Sprintf("key_%d", i)
        value := fmt.Sprintf("value_%d", i)
        db.PutCF("default", key, value)
    }
    
    const numGoroutines = 50
    const numOperations = 100
    
    var wg sync.WaitGroup
    errChan := make(chan error, numGoroutines*numOperations)
    
    // 启动多个并发读取器
    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for j := 0; j < numOperations; j++ {
                key := fmt.Sprintf("key_%d", rand.Intn(100))
                _, err := db.GetCF("default", key)
                if err != nil && !errors.Is(err, db.ErrKeyNotFound) {
                    errChan <- fmt.Errorf("worker %d operation %d: %w", workerID, j, err)
                }
            }
        }(i)
    }
    
    // 启动并发写入器
    for i := 0; i < numGoroutines/2; i++ {
        wg.Add(1)
        go func(workerID int) {
            defer wg.Done()
            
            for j := 0; j < numOperations/2; j++ {
                key := fmt.Sprintf("new_key_%d_%d", workerID, j)
                value := fmt.Sprintf("new_value_%d_%d", workerID, j)
                err := db.PutCF("default", key, value)
                if err != nil {
                    errChan <- fmt.Errorf("writer %d operation %d: %w", workerID, j, err)
                }
            }
        }(i)
    }
    
    wg.Wait()
    close(errChan)
    
    // 检查错误
    var errors []error
    for err := range errChan {
        errors = append(errors, err)
    }
    
    if len(errors) > 0 {
        t.Fatalf("Concurrent access errors: %v", errors)
    }
}

func TestDB_GracefulShutdown(t *testing.T) {
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    const numWorkers = 10
    var wg sync.WaitGroup
    
    // 启动长期运行的工作器
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            
            for {
                _, err := db.GetCF("default", "test_key")
                if err != nil {
                    if errors.Is(err, db.ErrDatabaseClosed) {
                        return // 正常关闭
                    }
                    // 其他错误也继续
                }
                time.Sleep(1 * time.Millisecond)
            }
        }()
    }
    
    // 等待工作器启动
    time.Sleep(100 * time.Millisecond)
    
    // 关闭数据库
    start := time.Now()
    err := db.Close()
    duration := time.Since(start)
    
    assert.NoError(t, err)
    assert.Less(t, duration, 5*time.Second, "shutdown took too long")
    
    wg.Wait()
}
```

### 竞态条件检测
```go
// 使用go run -race检测竞态条件
func TestRaceConditions(t *testing.T) {
    if !testing.Short() {
        t.Skip("Race condition test requires -race flag")
    }
    
    db, cleanup := setupTestDB(t)
    defer cleanup()
    
    state := &SafeReplState{}
    
    // 并发修改状态
    var wg sync.WaitGroup
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            cf := fmt.Sprintf("cf_%d", id%10)
            state.SetCurrentCF(cf)
            _ = state.GetCurrentCF()
        }(i)
    }
    
    wg.Wait()
}
```

## 性能基准测试

```go
func BenchmarkDB_ConcurrentRead(b *testing.B) {
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
```

## 监控和度量

### 并发度量指标
```go
type ConcurrencyMetrics struct {
    ActiveOperations int64
    TotalOperations  int64
    LockWaitTime     time.Duration
    LockHoldTime     time.Duration
}

func (d *SafeDB) GetMetrics() ConcurrencyMetrics {
    return ConcurrencyMetrics{
        ActiveOperations: atomic.LoadInt64(&d.activeOps),
        TotalOperations:  atomic.LoadInt64(&d.totalOps),
        // ... 其他指标
    }
}
```

## 配置示例

```yaml
concurrency:
  max_concurrent_operations: 1000
  operation_timeout: "30s"
  graceful_shutdown_timeout: "10s"
  lock_wait_timeout: "5s"
  enable_race_detection: true
```

## 风险评估

**中等风险**
- 并发控制可能影响性能
- 锁的不当使用可能导致死锁
- 过度同步可能降低并发能力

**缓解措施**
- 充分的并发测试
- 性能基准测试
- 渐进式优化

## 后续任务

- Task 04: 资源管理（并发安全的资源管理）
- Task 02: 日志系统（并发安全的日志记录）
- Task 09: 监控度量（并发性能监控）

## 参考资料

- [Go并发模式](https://blog.golang.org/concurrency-patterns)
- [Go内存模型](https://golang.org/ref/mem)
- [sync包文档](https://pkg.go.dev/sync)
- [竞态检测器](https://golang.org/doc/articles/race_detector.html) 