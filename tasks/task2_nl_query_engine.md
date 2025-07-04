# Task 2: Natural Language Query Engine Development

## 任务概述 (Task Overview)
开发自然语言查询解析引擎，负责将用户的自然语言输入转换为结构化的数据库查询操作。

## 优先级 (Priority)
**高 (High)** - 核心功能模块

## 预估工时 (Estimated Hours)
20-24 小时

## 具体任务 (Specific Tasks)

### 2.1 意图识别系统 (Intent Recognition System)
- [ ] 设计查询意图分类体系
- [ ] 实现基于规则的意图识别
- [ ] 集成 LLM 进行复杂意图解析
- [ ] 创建意图置信度评估机制

**支持的查询意图类型**:
```go
type QueryIntent string

const (
    IntentGet        QueryIntent = "get"         // 单个键查询
    IntentSearch     QueryIntent = "search"      // 模糊搜索
    IntentScan       QueryIntent = "scan"        // 范围扫描
    IntentPrefix     QueryIntent = "prefix"      // 前缀查询
    IntentJSONQuery  QueryIntent = "json_query"  // JSON字段查询
    IntentStats      QueryIntent = "stats"       // 统计信息
    IntentList       QueryIntent = "list"        // 列表操作
    IntentCount      QueryIntent = "count"       // 计数查询
    IntentAnalyze    QueryIntent = "analyze"     // 数据分析
    IntentHelp       QueryIntent = "help"        // 帮助信息
)
```

**输出文件**:
- `internal/nlquery/intent.go`
- `internal/nlquery/patterns.go`
- `internal/nlquery/classifier.go`

### 2.2 参数提取器 (Parameter Extractor)
- [ ] 实现命名实体识别（NER）
- [ ] 提取查询关键参数（键名、值、范围等）
- [ ] 处理时间表达式和数值范围
- [ ] 实现参数验证和标准化

**参数提取示例**:
```go
type QueryParameters struct {
    ColumnFamily string            `json:"column_family"`
    Key          string            `json:"key"`
    Value        string            `json:"value"`
    Prefix       string            `json:"prefix"`
    StartKey     string            `json:"start_key"`
    EndKey       string            `json:"end_key"`
    Field        string            `json:"field"`
    Operator     ComparisonOp      `json:"operator"`
    Limit        int               `json:"limit"`
    TimeRange    *TimeRange        `json:"time_range"`
    Filters      map[string]string `json:"filters"`
}

type TimeRange struct {
    Start time.Time `json:"start"`
    End   time.Time `json:"end"`
}
```

**输出文件**:
- `internal/nlquery/extractor.go`
- `internal/nlquery/entities.go`
- `internal/nlquery/timeparser.go`

### 2.3 查询规划器 (Query Planner)
- [ ] 将意图和参数转换为数据库操作序列
- [ ] 实现查询优化策略
- [ ] 支持复合查询分解
- [ ] 创建查询执行计划

**查询计划结构**:
```go
type QueryPlan struct {
    ID          string              `json:"id"`
    Intent      QueryIntent         `json:"intent"`
    Parameters  QueryParameters     `json:"parameters"`
    Operations  []DatabaseOperation `json:"operations"`
    Explanation string              `json:"explanation"`
    EstimatedCost int               `json:"estimated_cost"`
    CreatedAt   time.Time           `json:"created_at"`
}

type DatabaseOperation struct {
    Type       OperationType         `json:"type"`
    Target     string               `json:"target"`
    Parameters map[string]interface{} `json:"parameters"`
    Order      int                  `json:"order"`
}
```

**输出文件**:
- `internal/nlquery/planner.go`
- `internal/nlquery/operations.go`
- `internal/nlquery/optimizer.go`

### 2.4 结果格式化器 (Result Formatter)
- [ ] 实现查询结果人性化展示
- [ ] 支持多种输出格式（表格、JSON、描述性文本）
- [ ] 创建智能摘要生成器
- [ ] 实现错误信息友好化

**格式化功能**:
```go
type ResultFormatter interface {
    FormatQueryResult(result *QueryResult, format OutputFormat) (string, error)
    FormatError(err error, context *QueryContext) string
    GenerateSummary(result *QueryResult) string
    SuggestNextActions(result *QueryResult) []string
}

type OutputFormat string

const (
    FormatTable       OutputFormat = "table"
    FormatJSON        OutputFormat = "json"
    FormatDescription OutputFormat = "description"
    FormatMarkdown    OutputFormat = "markdown"
)
```

**输出文件**:
- `internal/nlquery/formatter.go`
- `internal/nlquery/templates.go`
- `internal/nlquery/summary.go`

## 接口设计 (Interface Design)

### 1. Natural Language Query Engine Interface
```go
type NLQueryEngine interface {
    // 解析自然语言查询
    ParseQuery(ctx context.Context, query string, context *QueryContext) (*QueryPlan, error)
    
    // 执行查询计划
    ExecutePlan(ctx context.Context, plan *QueryPlan) (*QueryResult, error)
    
    // 完整的查询处理流程
    ProcessQuery(ctx context.Context, query string, context *QueryContext) (*QueryResult, error)
    
    // 获取建议查询
    GetSuggestions(ctx context.Context, partial string) ([]QuerySuggestion, error)
    
    // 解释查询执行过程
    ExplainQuery(ctx context.Context, query string) (*QueryExplanation, error)
}
```

### 2. Query Context Interface
```go
type QueryContext struct {
    UserID          string                 `json:"user_id"`
    SessionID       string                 `json:"session_id"`
    DatabaseSchema  *DatabaseSchema        `json:"database_schema"`
    ConversationHistory []ConversationTurn `json:"conversation_history"`
    CurrentCF       string                 `json:"current_cf"`
    Preferences     *UserPreferences       `json:"preferences"`
    Timestamp       time.Time              `json:"timestamp"`
}

type DatabaseSchema struct {
    ColumnFamilies []ColumnFamilyInfo `json:"column_families"`
    KeyPatterns    []KeyPattern       `json:"key_patterns"`
    CommonPrefixes []string           `json:"common_prefixes"`
    SampleData     map[string]string  `json:"sample_data"`
}
```

### 3. Query Processing Pipeline
```go
type QueryProcessor struct {
    intentClassifier *IntentClassifier
    paramExtractor   *ParameterExtractor
    queryPlanner     *QueryPlanner
    resultFormatter  *ResultFormatter
    dbInterface      db.KeyValueDB
    logger           *Logger
}

func (qp *QueryProcessor) ProcessQuery(ctx context.Context, input string, context *QueryContext) (*QueryResult, error) {
    // 1. 意图识别
    intent, confidence := qp.intentClassifier.Classify(input, context)
    
    // 2. 参数提取
    params := qp.paramExtractor.Extract(input, intent, context)
    
    // 3. 查询规划
    plan := qp.queryPlanner.CreatePlan(intent, params, context)
    
    // 4. 执行查询
    rawResult := qp.executePlan(ctx, plan)
    
    // 5. 结果格式化
    formattedResult := qp.resultFormatter.Format(rawResult, context.Preferences)
    
    return formattedResult, nil
}
```

## 核心算法设计 (Core Algorithm Design)

### 1. 意图识别算法
```go
func (ic *IntentClassifier) Classify(query string, context *QueryContext) (QueryIntent, float64) {
    // 规则基础分类
    ruleScore := ic.ruleBasedClassification(query)
    
    // 基于上下文的调整
    contextScore := ic.contextualAdjustment(ruleScore, context)
    
    // LLM 辅助验证（对于低置信度查询）
    if contextScore.Confidence < 0.8 {
        llmScore := ic.llmClassification(query, context)
        return ic.combineScores(contextScore, llmScore)
    }
    
    return contextScore.Intent, contextScore.Confidence
}
```

### 2. 智能参数提取
```go
func (pe *ParameterExtractor) Extract(query string, intent QueryIntent, context *QueryContext) *QueryParameters {
    // 基于意图的模板匹配
    template := pe.getTemplateForIntent(intent)
    
    // 正则表达式提取
    regexMatches := pe.extractWithRegex(query, template)
    
    // 命名实体识别
    entities := pe.extractEntities(query, context.DatabaseSchema)
    
    // 时间表达式解析
    timeRanges := pe.parseTimeExpressions(query)
    
    // 参数验证和标准化
    return pe.validateAndNormalize(regexMatches, entities, timeRanges, context)
}
```

### 3. 查询优化策略
```go
func (qp *QueryPlanner) OptimizePlan(plan *QueryPlan, schema *DatabaseSchema) *QueryPlan {
    // 1. 选择最优的列族
    plan.Parameters.ColumnFamily = qp.selectOptimalColumnFamily(plan, schema)
    
    // 2. 索引利用优化
    plan.Operations = qp.optimizeIndexUsage(plan.Operations, schema)
    
    // 3. 查询顺序优化
    plan.Operations = qp.reorderOperations(plan.Operations)
    
    // 4. 结果集大小预估
    plan.EstimatedCost = qp.estimateQueryCost(plan, schema)
    
    return plan
}
```

## 测试要求 (Testing Requirements)

### 2.1 单元测试
- [ ] 意图识别准确性测试（测试集覆盖各种查询类型）
- [ ] 参数提取精确度测试
- [ ] 查询规划逻辑测试
- [ ] 结果格式化测试

### 2.2 集成测试
- [ ] 端到端查询处理测试
- [ ] 与数据库接口集成测试
- [ ] 错误处理流程测试

### 2.3 性能测试
- [ ] 查询解析延迟测试
- [ ] 并发查询处理能力测试
- [ ] 内存使用优化测试

### 2.4 用户测试
- [ ] 自然语言理解准确性评估
- [ ] 用户查询意图覆盖度测试
- [ ] 查询结果满意度调查

## 示例查询处理 (Example Query Processing)

### 输入示例
```
用户输入: "帮我找一下用户 'alice' 的配置信息"
```

### 处理流程
```go
// 1. 意图识别
Intent: IntentPrefix
Confidence: 0.95

// 2. 参数提取
Parameters: {
    ColumnFamily: "users" (推断)
    Prefix: "user:alice:config"
    Key: "alice"
    Limit: 10 (默认)
}

// 3. 查询计划
Plan: {
    Operations: [
        {
            Type: "prefix_scan",
            Target: "users",
            Parameters: {"prefix": "user:alice:config", "limit": 10}
        }
    ]
}

// 4. 执行结果
Result: {
    Success: true,
    Data: [
        {"key": "user:alice:config:theme", "value": "dark"},
        {"key": "user:alice:config:language", "value": "zh-CN"}
    ],
    Explanation: "找到了2条关于用户alice的配置信息"
}
```

## 验收标准 (Acceptance Criteria)

- [ ] 支持至少20种常见查询模式
- [ ] 意图识别准确率 ≥ 90%
- [ ] 参数提取准确率 ≥ 85%
- [ ] 查询响应时间 < 2秒（简单查询）
- [ ] 支持中英文混合查询
- [ ] 错误处理友好，提供有用的建议
- [ ] 集成测试通过率 100%
- [ ] 用户满意度 ≥ 4.0/5.0

## 依赖关系 (Dependencies)
- **前置任务**: Task 1 (GraphChain Integration) - 需要工具接口
- **并行任务**: Task 3 (Conversational UI) - 界面集成
- **后续任务**: Task 4 (Context Management), Task 5 (Intent Recognition)

## 技术栈 (Technology Stack)
- **NLP 库**: 自然语言处理工具包
- **正则表达式**: Go regexp 包
- **时间解析**: 自定义时间表达式解析器
- **模板引擎**: Go template 包
- **日志系统**: 结构化日志记录

## 风险和缓解 (Risks & Mitigations)

### 风险
1. **自然语言理解复杂性**: 中文语义解析困难
2. **查询歧义性**: 同一句话可能有多种理解
3. **性能瓶颈**: 复杂NLP处理可能影响响应时间

### 缓解策略
1. **渐进式实现**: 从简单查询开始，逐步支持复杂语义
2. **确认机制**: 对于歧义查询，提供确认选项
3. **缓存优化**: 缓存常见查询的解析结果 