# Key Transformation 使用示例

## 目录
- [概述](#概述)
- [基础用法](#基础用法)
- [实际应用场景](#实际应用场景)
- [高级用法](#高级用法)
- [最佳实践](#最佳实践)

## 概述

Key transformation 功能允许你同时转换键(key)和值(value)。这对于数据迁移、键格式标准化等场景非常有用。

**核心特性：**
- 🔑 支持 key 和 value 的独立转换
- 🔍 Dry-run 模式预览变更
- 🎯 支持 Python 表达式和脚本文件
- 📊 详细的统计信息
- ✅ 遵循 TDD 开发，测试覆盖完整

## 基础用法

### 1. 仅转换 Value（传统用法）

```bash
# 将所有值转换为大写
rocksdb-cli transform --db mydb --cf users \
  --expr="value.upper()" \
  --dry-run
```

### 2. 仅转换 Key

```bash
# 将所有键转换为大写
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --value-expr="value" \
  --dry-run
```

**输出示例：**
```
Key transformation: "user:1001" -> "USER:1001"
Key transformation: "user:1002" -> "USER:1002"
```

### 3. 同时转换 Key 和 Value

```bash
# 键转大写，值也转大写
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --expr="value.upper()" \
  --dry-run
```

**输出示例：**
```
Key: "user:1001" -> "USER:1001"
Value: "alice" -> "ALICE"
```

## 实际应用场景

### 场景1: 键格式标准化 - 冒号替换为下划线

**问题：** 旧系统使用 `user:1001` 格式，新系统要求 `user_1001` 格式

```bash
# 预览变更
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run --limit=5

# 确认无误后执行
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value"
```

**结果：**
```
✓ user:1001 → user_1001
✓ user:1002 → user_1002
✓ admin:500 → admin_500
```

### 场景2: 添加前缀进行命名空间隔离

**问题：** 需要为所有旧键添加 `v1_` 前缀以区分版本

```bash
# 添加前缀
rocksdb-cli transform --db mydb --cf users \
  --key-expr="'v1_' + key" \
  --value-expr="value" \
  --dry-run --limit=10

# 实际执行
rocksdb-cli transform --db mydb --cf users \
  --key-expr="'v1_' + key" \
  --value-expr="value"
```

**结果：**
```
✓ user:1001 → v1_user:1001
✓ user:1002 → v1_user:1002
✓ product:5  → v1_product:5
```

### 场景3: 键值联合转换 - JSON字段迁移

**问题：** 将 JSON 中的 `user_id` 字段移到 key 中

```bash
# 使用Python表达式
rocksdb-cli transform --db mydb --cf users \
  --key-expr="import json; 'user_' + str(json.loads(value).get('user_id', key))" \
  --expr="import json; d=json.loads(value); d.pop('user_id', None); json.dumps(d)" \
  --dry-run --limit=3
```

**转换前：**
```
key:   "temp_001"
value: {"user_id": 1001, "name": "Alice", "email": "alice@example.com"}
```

**转换后：**
```
key:   "user_1001"
value: {"name": "Alice", "email": "alice@example.com"}
```

### 场景4: 带过滤的键转换

**问题：** 只转换特定前缀的键

```bash
# 只转换以 "user:" 开头的键
rocksdb-cli transform --db mydb --cf mixed_data \
  --filter="key.startswith('user:')" \
  --key-expr="key.replace('user:', 'person_')" \
  --value-expr="value" \
  --dry-run
```

**结果：**
```
✓ user:1001 → person_1001  (转换)
✓ admin:500 → admin:500    (跳过 - 不匹配过滤条件)
✓ user:1002 → person_1002  (转换)
```

### 场景5: 键大小写标准化

**问题：** 历史数据键大小写不一致，需要统一为小写

```bash
# 统一转为小写
rocksdb-cli transform --db mydb --cf products \
  --key-expr="key.lower()" \
  --value-expr="value" \
  --dry-run --limit=10

# 执行转换
rocksdb-cli transform --db mydb --cf products \
  --key-expr="key.lower()" \
  --value-expr="value"
```

**结果：**
```
✓ Product:001 → product:001
✓ ADMIN:500   → admin:500
✓ User:1001   → user:1001
```

## 高级用法

### 1. 使用脚本文件进行复杂转换

创建 `scripts/transform/key_migration.py`:

```python
import json
import hashlib

def should_process(key, value):
    """只处理JSON格式的值"""
    try:
        json.loads(value)
        return True
    except:
        return False

def transform_key(key, value):
    """基于value内容生成新key"""
    data = json.loads(value)
    # 使用email生成key
    if 'email' in data:
        return f"user_by_email:{data['email']}"
    return key

def transform_value(key, value):
    """添加迁移时间戳"""
    data = json.loads(value)
    data['migrated_at'] = '2025-01-01T00:00:00Z'
    return json.dumps(data)
```

**使用脚本：**
```bash
rocksdb-cli transform --db mydb --cf users \
  --script=scripts/transform/key_migration.py \
  --dry-run --limit=5
```

**注意：** 当前脚本文件功能主要用于 value transformation。Key transformation 主要通过 `--key-expr` 实现。

### 2. 批量处理大数据集

```bash
# 分批处理，避免内存问题
rocksdb-cli transform --db mydb --cf huge_data \
  --key-expr="key.upper()" \
  --value-expr="value" \
  --batch-size=5000 \
  --verbose
```

### 3. 条件性键转换

```bash
# 只转换包含特定字符的键
rocksdb-cli transform --db mydb --cf users \
  --filter="':' in key" \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run
```

## 最佳实践

### 1. 安全第一 - 总是先 Dry-run

```bash
# ✅ 推荐：先预览
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --expr="value.upper()" \
  --dry-run --limit=10

# ✅ 检查输出确认无误

# ✅ 然后执行
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --expr="value.upper()"
```

### 2. 小批量测试

```bash
# ✅ 先在小数据集上测试
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --value-expr="value" \
  --limit=10
```

### 3. 检查统计信息

执行后查看输出：
```
Transform Statistics:
  Processed: 1000
  Modified:  950
  Skipped:   50
  Errors:    0
  Duration:  2.3s
```

### 4. 备份重要数据

```bash
# ⚠️ 对重要数据，先备份
cp -r /path/to/mydb /path/to/mydb.backup

# 然后执行转换
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --value-expr="value"
```

### 5. 理解键变更的影响

**重要：** 当键发生变化时：
- ✅ 新键会被写入数据库
- ⚠️ 旧键目前**不会**自动删除（需要手动清理）
- 💡 如果需要完全迁移，考虑：
  1. 先转换写入新键
  2. 验证新键数据正确
  3. 手动删除旧键

**示例：完整迁移流程**
```bash
# 步骤1: 转换并写入新键
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value"

# 步骤2: 验证新键
rocksdb-cli prefix --db mydb --cf users --prefix "user_"

# 步骤3: 手动删除旧键（如果需要）
# 注意：当前CLI可能需要添加批量删除功能
```

## Python 表达式可用变量

在 `--key-expr` 和 `--expr` / `--value-expr` 中可以使用：

| 变量 | 类型 | 说明 |
|------|------|------|
| `key` | string | 当前条目的键 |
| `value` | string | 当前条目的值 |

**Python 模块导入：**
```bash
# 可以使用 import 导入标准库
--key-expr="import json; key.upper()"
--expr="import hashlib; hashlib.md5(value.encode()).hexdigest()"
```

## 常见错误与解决

### 错误1: KeyExpression 返回空值

```bash
# ❌ 错误：表达式返回None
--key-expr="key.split(':')[5]"  # 如果索引不存在会报错

# ✅ 正确：添加错误处理
--key-expr="key.split(':')[1] if ':' in key and len(key.split(':')) > 1 else key"
```

### 错误2: 键冲突

```bash
# ⚠️ 如果多个旧键转换为同一个新键，后者会覆盖前者
# 解决：在转换前检查是否会产生重复
--key-expr="key.split(':')[0] if ':' in key else key"
```

### 错误3: Python语法错误

```bash
# ❌ 语法错误
--key-expr="key.upper("  # 括号不匹配

# ✅ 正确
--key-expr="key.upper()"
```

## 性能优化

### 大数据集处理建议

```bash
# 1. 使用批处理
--batch-size=10000

# 2. 限制处理数量进行测试
--limit=1000

# 3. 启用详细输出监控进度
--verbose

# 完整示例
rocksdb-cli transform --db mydb --cf huge_table \
  --key-expr="key.lower()" \
  --value-expr="value" \
  --batch-size=10000 \
  --verbose
```

## 总结

Key transformation 是一个强大的功能，适用于：
- ✅ 数据库迁移
- ✅ 键格式标准化
- ✅ 命名空间隔离
- ✅ 批量重命名

**记住：**
1. 🔍 总是先 `--dry-run`
2. 📊 检查统计信息
3. 💾 重要数据先备份
4. 🧪 小批量测试
5. 📖 理解键变更的影响

## 相关文档

- [Transform命令完整文档](../README.md#transform-command)
- [Python脚本示例](../scripts/transform/README.md)
- [API参考](../internal/transform/README.md)
