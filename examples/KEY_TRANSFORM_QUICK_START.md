# Key Transformation 快速开始

## 🚀 5分钟上手

### 1️⃣ 最简单的例子 - 键转大写

```bash
# 预览（安全，不会修改数据）
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --value-expr="value" \
  --dry-run

# 实际执行
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --value-expr="value"
```

**结果：**
```
user:1001 → USER:1001
user:1002 → USER:1002
```

---

### 2️⃣ 键格式标准化 - 冒号替换为下划线

```bash
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run
```

**结果：**
```
user:1001 → user_1001
admin:500 → admin_500
```

---

### 3️⃣ 添加前缀

```bash
rocksdb-cli transform --db mydb --cf users \
  --key-expr="'v2_' + key" \
  --value-expr="value" \
  --dry-run
```

**结果：**
```
user:1001 → v2_user:1001
product:5 → v2_product:5
```

---

### 4️⃣ 带过滤条件

```bash
# 只转换特定前缀的键
rocksdb-cli transform --db mydb --cf users \
  --filter="key.startswith('user:')" \
  --key-expr="key.replace('user:', 'person_')" \
  --value-expr="value" \
  --dry-run
```

**结果：**
```
user:1001 → person_1001  ✅ 转换
admin:500 → admin:500    ⏭️ 跳过
```

---

### 5️⃣ 同时转换键和值

```bash
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.upper()" \
  --expr="value.upper()" \
  --dry-run
```

**结果：**
```
键: user:1001 → USER:1001
值: alice → ALICE
```

---

## 📋 命令参数快速参考

| 参数 | 说明 | 示例 |
|------|------|------|
| `--key-expr` | 转换键的Python表达式 | `key.upper()` |
| `--expr` | 转换值的Python表达式 | `value.upper()` |
| `--value-expr` | 转换值（同 --expr） | `value.lower()` |
| `--filter` | 过滤条件 | `key.startswith('user:')` |
| `--dry-run` | 预览模式（不修改数据）⭐ | - |
| `--limit` | 限制处理数量 | `--limit=10` |
| `--cf` | 指定列族 | `--cf=users` |

---

## ⚠️ 重要提示

### ✅ 必做事项
1. **总是先使用 `--dry-run`** 预览变更
2. **使用 `--limit=10`** 在小数据集上测试
3. **备份重要数据** 再执行转换

### ⚡ Python 表达式技巧

```python
# 字符串操作
key.upper()              # 大写
key.lower()              # 小写
key.replace(':', '_')    # 替换
'prefix_' + key          # 添加前缀
key + '_suffix'          # 添加后缀
key.split(':')[0]        # 分割取第一部分

# 条件表达式
key.upper() if ':' in key else key

# JSON 操作（需要导入）
import json; json.loads(value)
```

---

## 🎯 常见场景速查

### 场景：数据库迁移
```bash
# 旧格式: user:1001
# 新格式: user_1001
rocksdb-cli transform --db mydb --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run
```

### 场景：版本隔离
```bash
# 为旧数据添加版本前缀
rocksdb-cli transform --db mydb --cf data \
  --key-expr="'v1_' + key" \
  --value-expr="value" \
  --dry-run
```

### 场景：大小写标准化
```bash
# 统一为小写
rocksdb-cli transform --db mydb --cf mixed \
  --key-expr="key.lower()" \
  --value-expr="value" \
  --dry-run
```

---

## 🔧 故障排查

### 问题：键没有变化
```bash
# 检查表达式是否正确
--key-expr="key.upper()"  # ✅ 正确
--key-expr="key"          # ❌ 没有转换
```

### 问题：Python语法错误
```bash
# 确保括号匹配
--key-expr="key.upper()"     # ✅ 正确
--key-expr="key.upper("      # ❌ 括号不匹配
```

### 问题：看不到结果
```bash
# 确保使用了 --dry-run 或查看实际数据
./rocksdb-cli scan --db mydb --cf users --limit=5
```

---

## 📚 更多资源

- **详细文档**: [docs/KEY_TRANSFORMATION_EXAMPLES.md](../docs/KEY_TRANSFORMATION_EXAMPLES.md)
- **演示脚本**: [examples/key_transformation_demo.sh](./key_transformation_demo.sh)
- **帮助命令**: `rocksdb-cli transform --help`

---

## 🎬 运行演示

```bash
# 运行交互式演示
cd examples
./key_transformation_demo.sh
```

演示包含：
- ✅ 键转大写
- ✅ 冒号替换为下划线
- ✅ 添加版本前缀
- ✅ 带过滤条件的转换
- ✅ 同时转换键和值
- ✅ 实际执行示例

---

**祝使用愉快！** 🎉

如有问题，请参考详细文档或提交 Issue。
