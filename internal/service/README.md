# Service Layer

## 概述

Service 层是 RocksDB CLI 的业务逻辑层，位于表示层（CLI/API）和数据访问层（internal/db）之间。它提供了可复用的业务逻辑，支持 CLI 和 Web API 共享相同的功能实现。

## 架构

```
┌─────────────────────────────────────────┐
│     Presentation Layer                  │
│  ┌────────────┐      ┌──────────────┐  │
│  │    CLI     │      │   Web API    │  │
│  └─────┬──────┘      └──────┬───────┘  │
└────────┼────────────────────┼───────────┘
         │                    │
┌────────▼────────────────────▼───────────┐
│      Service Layer (This Layer)         │
│  ┌──────────────────────────────────┐  │
│  │  - DatabaseService               │  │
│  │  - ScanService                   │  │
│  │  - SearchService                 │  │
│  │  - StatsService                  │  │
│  │  │  - ExportService                 │  │
│  │  - TransformService              │  │
│  └──────────────────────────────────┘  │
└─────────────────────────────────────────┘
         │
┌────────▼─────────────────────────────────┐
│      Data Access Layer                   │
│  ┌──────────────────────────────────┐   │
│  │  internal/db                     │   │
│  │  internal/transform              │   │
│  └──────────────────────────────────┘   │
└──────────────────────────────────────────┘
```

## Services

### 1. DatabaseService

提供基础的数据库操作功能。

**主要方法：**
- `GetValue(cf, key)` - 获取指定 key 的值
- `PutValue(cf, key, value)` - 写入或更新 key-value
- `DeleteValue(cf, key)` - 删除指定 key
- `ListColumnFamilies()` - 列出所有列族
- `CreateColumnFamily(name)` - 创建新列族
- `DropColumnFamily(name)` - 删除列族
- `GetLastEntry(cf)` - 获取最后一个条目

**使用示例：**
```go
import "rocksdb-cli/internal/service"

// 创建服务
dbService := service.NewDatabaseService(database)

// 获取值
value, err := dbService.GetValue("users", "user:1001")
if err != nil {
    // 处理错误
}

// 写入值
err = dbService.PutValue("users", "user:1002", `{"name":"Alice"}`)
```

### 2. ScanService

提供数据扫描和遍历功能。

**主要方法：**
- `Scan(cf, opts)` - 范围扫描，支持分页
- `PrefixScan(cf, prefix, limit)` - 前缀扫描
- `GetAll(cf, limit)` - 获取所有数据

**使用示例：**
```go
scanService := service.NewScanService(database)

// 前缀扫描
result, err := scanService.PrefixScan("users", "user:", 100)
fmt.Printf("Found %d entries\n", result.Count)

// 分页扫描
opts := service.ScanOptions{
    StartKey: "",
    Limit:    50,
    KeysOnly: false,
}
result, err := scanService.Scan("users", opts)

// 下一页
if result.HasMore {
    opts.After = result.NextCursor
    nextPage, err := scanService.Scan("users", opts)
}
```

### 3. SearchService

提供高级搜索功能。

**主要方法：**
- `Search(cf, opts)` - 高级搜索（支持正则、分页）
- `JSONQuery(cf, field, value)` - JSON 字段查询

**使用示例：**
```go
searchService := service.NewSearchService(database)

// 搜索包含 "admin" 的 key
opts := service.SearchOptions{
    KeyPattern: "admin",
    Limit:      100,
}
result, err := searchService.Search("users", opts)

// JSON 字段查询
result, err := searchService.JSONQuery("users", "age", "25")
```

### 4. StatsService

提供数据库统计功能。

**主要方法：**
- `GetDatabaseStats()` - 获取整体数据库统计
- `GetColumnFamilyStats(cf)` - 获取列族统计

**使用示例：**
```go
statsService := service.NewStatsService(database)

// 获取数据库统计
stats, err := statsService.GetDatabaseStats()
fmt.Printf("Total keys: %d\n", stats.TotalKeyCount)
fmt.Printf("Column families: %d\n", stats.ColumnFamilyCount)

// 获取列族统计
cfStats, err := statsService.GetColumnFamilyStats("users")
fmt.Printf("Keys in users: %d\n", cfStats.KeyCount)
```

### 5. ExportService

提供数据导出功能。

**主要方法：**
- `ExportToCSV(writer, opts)` - 导出为 CSV
- `ExportToJSON(writer, opts)` - 导出为 JSON
- `ExportSearchResults(writer, cf, searchOpts, csvSep)` - 导出搜索结果

**使用示例：**
```go
exportService := service.NewExportService(database)

// 导出到 CSV
file, _ := os.Create("users.csv")
defer file.Close()

opts := service.ExportOptions{
    CF:        "users",
    Format:    "csv",
    Separator: ",",
    Header:    true,
}
result, err := exportService.ExportToCSV(file, opts)
fmt.Printf("Exported %d records\n", result.RecordCount)

// 导出为 JSON
jsonFile, _ := os.Create("users.json")
defer jsonFile.Close()

opts.Format = "json"
opts.Pretty = true
result, err = exportService.ExportToJSON(jsonFile, opts)
```

### 6. TransformService

提供数据转换功能。

**主要方法：**
- `Transform(opts)` - 执行数据转换

**使用示例：**
```go
transformService := service.NewTransformService(database)

// Dry-run 模式预览
opts := service.TransformOptions{
    CF:         "users",
    Expression: "value.upper()",
    DryRun:     true,
    Limit:      10,
}
result, err := transformService.Transform(opts)

fmt.Printf("Preview: %d entries will be modified\n", len(result.Preview))

// 执行转换
opts.DryRun = false
result, err = transformService.Transform(opts)
fmt.Printf("Modified %d entries\n", result.Modified)
```

## 设计原则

### 1. 单一职责
每个 Service 只负责特定的业务领域：
- DatabaseService：基础 CRUD 操作
- ScanService：数据扫描
- SearchService：搜索查询
- StatsService：统计分析
- ExportService：数据导出
- TransformService：数据转换

### 2. 依赖注入
所有 Service 通过构造函数注入依赖：
```go
func NewDatabaseService(database db.KeyValueDB) *DatabaseService {
    return &DatabaseService{db: database}
}
```

### 3. 接口隔离
Service 层定义自己的数据结构，不直接暴露底层 DB 结构：
```go
// Service 层的 ScanResult
type ScanResult struct {
    Data       map[string]string
    Count      int
    HasMore    bool
    NextCursor string
}

// 而不是直接使用 db.ScanPageResult
```

### 4. 错误处理
Service 层传递底层错误，必要时添加上下文：
```go
func (s *DatabaseService) PutValue(cf, key, value string) error {
    if s.db.IsReadOnly() {
        return db.ErrReadOnlyMode
    }
    return s.db.PutCF(cf, key, value)
}
```

## 测试

Service 层使用 Mock 对象进行单元测试：

```go
// 创建 Mock DB
mockDB := NewMockDB()
mockDB.data["users"] = map[string]string{
    "user:1001": `{"name":"Alice"}`,
}

// 测试 Service
service := NewDatabaseService(mockDB)
value, err := service.GetValue("users", "user:1001")
```

运行测试：
```bash
go test ./internal/service/... -v
```

## 与其他层的关系

### CLI 层使用示例

```go
// cmd/main.go
func executeGet(cf, key string) {
    db := openDatabase()
    defer db.Close()

    // 使用 Service
    dbService := service.NewDatabaseService(db)
    value, err := dbService.GetValue(cf, key)

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // CLI 展示
    fmt.Printf("Key: %s\n", key)
    fmt.Printf("Value: %s\n", value)
}
```

### API 层使用示例

```go
// internal/api/handlers/database_handler.go
func (h *DatabaseHandler) GetValue(c *gin.Context) {
    cf := c.Param("cf")
    key := c.Param("key")

    // 使用 Service
    value, err := h.dbService.GetValue(cf, key)

    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{
            "success": false,
            "error":   err.Error(),
        })
        return
    }

    // API 响应
    c.JSON(http.StatusOK, gin.H{
        "success": true,
        "data": gin.H{
            "key":   key,
            "value": value,
        },
    })
}
```

## 下一步

Service 层已经实现了核心功能，接下来可以：

1. **重构 CLI** - 将 cmd/main.go 中的业务逻辑迁移到 Service 层
2. **创建 API 层** - 基于 Service 层实现 HTTP API
3. **添加更多测试** - 为每个 Service 添加完整的单元测试
4. **性能优化** - 添加缓存、批处理等优化

## 文件列表

```
internal/service/
├── README.md                      # 本文档
├── database_service.go            # 基础数据库操作
├── database_service_test.go       # 测试
├── scan_service.go                # 扫描服务
├── search_service.go              # 搜索服务
├── stats_service.go               # 统计服务
├── export_service.go              # 导出服务
└── transform_service.go           # 转换服务
```

## 参考

- [代码重构计划](../../docs/CODE_REFACTORING_PLAN.md)
- [架构决策](../../docs/ARCHITECTURE_DECISIONS.md)
- [Web UI MVP 路线图](../../docs/WEB_UI_MVP_ROADMAP.md)
