# GraphChain记忆功能演示

## 🧠 什么是记忆功能？

GraphChain的记忆功能允许智能体在对话中记住之前的交互，提供**上下文感知**的响应。这使得用户可以进行连续性对话，而不需要每次都重复背景信息。

## 🚀 快速开始

### 1. 启用记忆功能

在 `config/graphchain.yaml` 中配置：

```yaml
graphchain:
  agent:
    enable_memory: true    # 🧠 启用对话记忆
    memory_size: 50        # 📝 记住最近50轮对话
```

### 2. 基本使用

启动rocksdb-cli并开始对话：

```bash
rocksdb-cli --db ./testdb --graphchain
```

## 📚 对话示例

### 场景1：上下文关联查询

```
用户：列出所有列族
智能体：数据库中有以下列族：users, products, orders

用户：users里面有什么？  # ← 智能体知道你在说users列族
智能体：users列族包含以下数据类型：
- 用户ID (string)
- 用户信息 (JSON)
- 时间戳 (timestamp)
```

### 场景2：数据探索会话

```
用户：显示products表的统计信息
智能体：products列族统计：
- 总条目：1,234
- 数据大小：45MB
- 主要数据类型：JSON

用户：找一些产品数据看看  # ← 智能体记住你在看products
智能体：以下是products中的一些示例数据：
product:001 -> {"name": "iPhone", "price": 999, "category": "electronics"}
product:002 -> {"name": "MacBook", "price": 1999, "category": "electronics"}
```

### 场景3：跨会话引用

```
用户：查找所有电子产品
智能体：[返回电子产品列表]

用户：这些产品的平均价格是多少？  # ← 智能体知道"这些产品"指什么
智能体：根据刚才查找的电子产品，平均价格是$1,299
```

## 🎯 记忆功能的优势

### ✅ **上下文连续性**
- 不需要重复说明背景
- 智能对话体验
- 自然的交互方式

### ✅ **效率提升**
- 减少重复查询
- 快速建立在先前结果基础上
- 流畅的数据探索

### ✅ **智能引用**
- 自动理解指代关系
- 记住重要的查询结果
- 支持复杂的数据分析流程

## 🛠️ 记忆管理命令

### 查看记忆状态
```bash
memory stats          # 显示记忆使用统计
memory history        # 显示最近5轮对话
memory history 10     # 显示最近10轮对话
memory clear          # 清除所有对话历史
```

### 记忆统计示例
```
📊 Memory Statistics:
  Total Conversations: 15
  Memory Capacity: 50
  Memory Usage: 30.0%
  Total Characters: 2,847
  Oldest Conversation: 2024-01-15 14:23:45
  Newest Conversation: 2024-01-15 14:28:12
```

### 对话历史示例
```
📚 Last 3 Conversation(s):
────────────────────────────────────────────────────────────

💬 Conversation 3 (14:28:12):
🙋 User: 列出所有列族
🤖 Agent: 数据库中有以下列族：users, products, orders
⏱️  Time: 1.2s

💬 Conversation 2 (14:27:45):
🙋 User: users里面有什么？
🤖 Agent: users列族包含用户数据，主要是JSON格式的用户信息...
⏱️  Time: 800ms

💬 Conversation 1 (14:27:20):
🙋 User: 显示数据库统计
🤖 Agent: 数据库总大小：156MB，包含3个列族...
⏱️  Time: 1.5s
```

## ⚙️ 高级配置

### 记忆大小调优

```yaml
agent:
  enable_memory: true
  memory_size: 100      # 较大的记忆容量，适合长时间会话
  # memory_size: 20     # 较小的记忆容量，适合简单查询
```

**推荐设置：**
- **轻量使用**：20-30轮对话
- **常规使用**：50-100轮对话  
- **深度分析**：100-200轮对话

### 记忆格式选择

代码中可以选择不同的记忆格式：

```go
// 聊天消息格式（推荐）
memory.SetReturnMessages(true)

// 文本字符串格式
memory.SetReturnMessages(false)
```

## 🔧 故障排除

### 记忆功能未启用？

检查配置文件：
```yaml
agent:
  enable_memory: true  # 确保设置为true
```

### 记忆容量不足？

增加memory_size或定期清理：
```bash
memory clear  # 清除历史记录
```

### 性能问题？

记忆太大可能影响响应速度，可以：
1. 减少memory_size
2. 定期使用`memory clear`
3. 关闭记忆功能进行对比测试

## 🎉 最佳实践

### 1. **合理设置记忆大小**
- 根据使用场景调整memory_size
- 定期监控memory stats

### 2. **有效利用上下文**
- 使用指代词（"这个"、"那些"、"刚才的"）
- 建立连续的查询链条

### 3. **定期清理记忆**
- 长时间会话后清理无关历史
- 切换主题时重置记忆

### 4. **监控记忆状态**
- 定期检查memory stats
- 观察记忆使用趋势

## 📈 实际应用场景

### 🔍 **数据探索**
```
1. 用户：显示数据库概览
2. 用户：users表有多少条记录？
3. 用户：随机显示几条用户数据
4. 用户：这些用户主要来自哪些地区？  # ← 基于之前的数据
```

### 📊 **性能分析**
```
1. 用户：显示所有列族的大小
2. 用户：哪个列族最大？
3. 用户：分析一下这个大列族的数据分布  # ← 自动理解指代
4. 用户：有什么优化建议？
```

### 🐛 **问题诊断**
```
1. 用户：检查数据库健康状态
2. 用户：有没有发现异常？
3. 用户：详细分析刚才提到的问题  # ← 记住之前的分析结果
4. 用户：给出解决方案
```

通过记忆功能，rocksdb-cli变成了一个真正智能的数据库助手，能够理解你的意图并提供连贯的帮助！ 🚀 