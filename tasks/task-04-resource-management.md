# Task 04: 资源管理机制

**优先级**: 🟡 中优先级  
**状态**: ⏳ 待开始  
**预估工作量**: 3-4小时  
**负责人**: 待分配  

## 问题描述

当前RocksDB资源管理存在潜在的内存泄露和错误处理问题：

### 发现的问题
1. **资源释放不够健壮**:
   ```go
   func (d *DB) Close() {
       for _, h := range d.cfHandles {
           h.Destroy()  // 如果某个Destroy失败，其他资源可能泄露
       }
       d.db.Close()
       d.ro.Destroy()
       d.wo.Destroy()
   }
   ```

2. **缺乏资源状态跟踪**:
   - 无法检测资源是否已释放
   - 重复关闭可能导致panic
   - 缺乏资源使用统计

3. **错误恢复机制不完善**:
   - 初始化失败时资源清理不彻底
   - 缺乏资源清理的超时机制

4. **并发访问资源风险**:
   - 关闭过程中其他goroutine可能访问资源
   - 缺乏资源生命周期保护

## 影响分析

- **稳定性**: 资源泄露可能导致内存耗尽
- **可靠性**: 不完整的清理可能导致数据丢失
- **性能**: 无法监控和优化资源使用
- **运维**: 难以诊断资源相关问题

## 解决方案

### 1. 资源管理架构
```
internal/resource/
├── manager.go          # 资源管理器
├── lifecycle.go        # 资源生命周期
├── monitor.go          # 资源监控
└── cleanup.go          # 清理策略
```

### 2. 资源生命周期管理
```go
type ResourceManager interface {
    Register(name string, resource Resource) error
    Unregister(name string) error
    Cleanup() error
    Status() map[string]ResourceStatus
}

type Resource interface {
    Close() error
    IsActive() bool
    ResourceType() string
}
```

### 3. 健壮的清理机制
- 支持优雅关闭和强制关闭
- 资源清理超时控制
- 错误恢复和重试机制
- 资源依赖关系管理

## 实施步骤

### Phase 1: 资源抽象 (1.5小时)
1. 定义Resource接口
2. 创建ResourceManager
3. 实现资源注册和跟踪
4. 添加资源状态管理

### Phase 2: 清理策略 (1.5小时)
1. 实现优雅关闭逻辑
2. 添加超时和重试机制
3. 实现依赖关系管理
4. 添加错误恢复逻辑

### Phase 3: 集成应用 (1小时)
1. 重构DB结构体实现Resource接口
2. 集成ResourceManager到应用
3. 更新测试用例
4. 添加监控指标

## 验收标准

### 功能要求
- [ ] 所有RocksDB资源正确注册和跟踪
- [ ] 优雅关闭和强制关闭机制
- [ ] 资源清理超时控制（默认30秒）
- [ ] 重复关闭保护
- [ ] 资源使用统计和监控

### 质量要求
- [ ] 资源管理模块单元测试覆盖率 > 85%
- [ ] 内存泄露测试通过
- [ ] 并发安全性测试通过
- [ ] 资源清理压力测试

### 性能要求
- [ ] 资源注册开销 < 1ms
- [ ] 正常关闭时间 < 5秒
- [ ] 强制关闭时间 < 1秒

## 测试计划

1. **单元测试**
   - 资源注册和注销
   - 生命周期管理
   - 错误处理和恢复
   - 超时机制

2. **集成测试**
   - 完整应用生命周期
   - 异常情况下的资源清理
   - 并发访问安全性

3. **压力测试**
   - 大量资源管理
   - 频繁开关数据库
   - 内存泄露检测

## 实现示例

### 资源包装器
```go
type DBResource struct {
    db        *grocksdb.DB
    cfHandles map[string]*grocksdb.ColumnFamilyHandle
    ro        *grocksdb.ReadOptions
    wo        *grocksdb.WriteOptions
    closed    int32
    mu        sync.RWMutex
}

func (r *DBResource) Close() error {
    if !atomic.CompareAndSwapInt32(&r.closed, 0, 1) {
        return ErrAlreadyClosed
    }

    r.mu.Lock()
    defer r.mu.Unlock()

    var errors []error

    // 清理列族句柄
    for name, handle := range r.cfHandles {
        if err := r.safeDestroy(handle.Destroy); err != nil {
            errors = append(errors, fmt.Errorf("failed to destroy CF %s: %w", name, err))
        }
    }

    // 清理选项
    if err := r.safeDestroy(r.ro.Destroy); err != nil {
        errors = append(errors, fmt.Errorf("failed to destroy read options: %w", err))
    }

    if err := r.safeDestroy(r.wo.Destroy); err != nil {
        errors = append(errors, fmt.Errorf("failed to destroy write options: %w", err))
    }

    // 关闭数据库
    if r.db != nil {
        r.db.Close()
    }

    if len(errors) > 0 {
        return fmt.Errorf("cleanup errors: %v", errors)
    }
    return nil
}

func (r *DBResource) IsActive() bool {
    return atomic.LoadInt32(&r.closed) == 0
}

func (r *DBResource) safeDestroy(destroyFunc func()) error {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("panic during resource cleanup: %v", r)
        }
    }()
    destroyFunc()
    return nil
}
```

### 资源管理器
```go
type Manager struct {
    resources map[string]Resource
    mu        sync.RWMutex
    shutdown  chan struct{}
    wg        sync.WaitGroup
}

func (m *Manager) Register(name string, resource Resource) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if _, exists := m.resources[name]; exists {
        return fmt.Errorf("resource %s already exists", name)
    }

    m.resources[name] = resource
    return nil
}

func (m *Manager) Cleanup() error {
    return m.CleanupWithTimeout(30 * time.Second)
}

func (m *Manager) CleanupWithTimeout(timeout time.Duration) error {
    done := make(chan error, 1)
    
    go func() {
        done <- m.doCleanup()
    }()

    select {
    case err := <-done:
        return err
    case <-time.After(timeout):
        return fmt.Errorf("cleanup timeout after %v", timeout)
    }
}

func (m *Manager) doCleanup() error {
    m.mu.Lock()
    resources := make([]Resource, 0, len(m.resources))
    for _, r := range m.resources {
        resources = append(resources, r)
    }
    m.mu.Unlock()

    var errors []error
    for _, resource := range resources {
        if err := resource.Close(); err != nil {
            errors = append(errors, err)
        }
    }

    if len(errors) > 0 {
        return fmt.Errorf("cleanup errors: %v", errors)
    }
    return nil
}
```

## 配置示例

```yaml
resource_management:
  cleanup_timeout: "30s"
  force_cleanup_timeout: "5s"
  monitor_interval: "10s"
  max_retries: 3
  retry_delay: "1s"
```

## 监控指标

1. **资源计数**
   - 活跃资源数量
   - 已注册资源数量
   - 清理失败资源数量

2. **性能指标**
   - 资源清理耗时
   - 清理成功率
   - 内存使用量

3. **错误统计**
   - 清理失败次数
   - 超时事件
   - 恢复重试次数

## 风险评估

**低风险**
- 主要是新增功能
- 不影响现有业务逻辑
- 可以渐进式集成

**潜在风险**
- 资源管理器本身的资源开销
- 清理超时可能影响应用关闭速度

## 后续任务

- Task 07: 并发安全性（资源管理的并发保护）
- Task 09: 监控度量（资源使用监控）
- Task 02: 日志系统（记录资源管理事件）

## 参考资料

- [Go资源管理最佳实践](https://golang.org/doc/effective_go.html#defer)
- [RocksDB内存管理](https://github.com/facebook/rocksdb/wiki/Memory-usage-in-RocksDB)
- [优雅关闭模式](https://blog.golang.org/context-and-structs)
- [资源清理策略](https://dave.cheney.net/2017/06/11/go-without-generics) 