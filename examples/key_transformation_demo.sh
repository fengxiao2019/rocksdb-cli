#!/bin/bash
# Key Transformation 功能演示
#
# 这个脚本展示如何使用 key transformation 功能

set -e

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}================================================${NC}"
echo -e "${BLUE}  RocksDB-CLI Key Transformation 演示${NC}"
echo -e "${BLUE}================================================${NC}"
echo ""

# 创建测试数据库
TEST_DB="/tmp/rocksdb-key-transform-demo"
rm -rf "$TEST_DB"

echo -e "${GREEN}步骤 1: 创建测试数据库${NC}"
echo "位置: $TEST_DB"

# 使用 rocksdb-cli 创建一些测试数据
./rocksdb-cli --db "$TEST_DB" put --cf users "user:1001" '{"name":"alice","age":25,"email":"alice@example.com"}'
./rocksdb-cli --db "$TEST_DB" put --cf users "user:1002" '{"name":"bob","age":30,"email":"bob@example.com"}'
./rocksdb-cli --db "$TEST_DB" put --cf users "user:1003" '{"name":"charlie","age":28,"email":"charlie@example.com"}'
./rocksdb-cli --db "$TEST_DB" put --cf users "admin:500" '{"name":"admin","role":"administrator"}'
./rocksdb-cli --db "$TEST_DB" put --cf users "admin:501" '{"name":"superadmin","role":"super_administrator"}'

echo -e "${GREEN}✓ 测试数据创建完成${NC}"
echo ""

# 显示原始数据
echo -e "${GREEN}步骤 2: 查看原始数据${NC}"
./rocksdb-cli --db "$TEST_DB" scan --cf users --pretty
echo ""

# 示例1: 键转大写
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 1: 将所有键转换为大写${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "命令："
echo "  rocksdb-cli transform --db $TEST_DB --cf users \\"
echo "    --key-expr=\"key.upper()\" \\"
echo "    --value-expr=\"value\" \\"
echo "    --dry-run --limit=3"
echo ""
echo "预览结果："
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --key-expr="key.upper()" \
  --value-expr="value" \
  --dry-run --limit=3
echo ""

# 示例2: 冒号替换为下划线
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 2: 将冒号替换为下划线（键格式标准化）${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "命令："
echo "  rocksdb-cli transform --db $TEST_DB --cf users \\"
echo "    --key-expr=\"key.replace(':', '_')\" \\"
echo "    --value-expr=\"value\" \\"
echo "    --dry-run --limit=5"
echo ""
echo "预览结果："
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run --limit=5
echo ""

# 示例3: 添加前缀
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 3: 为所有键添加版本前缀${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "命令："
echo "  rocksdb-cli transform --db $TEST_DB --cf users \\"
echo "    --key-expr=\"'v2_' + key\" \\"
echo "    --value-expr=\"value\" \\"
echo "    --dry-run --limit=3"
echo ""
echo "预览结果："
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --key-expr="'v2_' + key" \
  --value-expr="value" \
  --dry-run --limit=3
echo ""

# 示例4: 带过滤的转换
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 4: 只转换 'user:' 开头的键${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "命令："
echo "  rocksdb-cli transform --db $TEST_DB --cf users \\"
echo "    --filter=\"key.startswith('user:')\" \\"
echo "    --key-expr=\"key.replace('user:', 'person_')\" \\"
echo "    --value-expr=\"value\" \\"
echo "    --dry-run"
echo ""
echo "预览结果："
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --filter="key.startswith('user:')" \
  --key-expr="key.replace('user:', 'person_')" \
  --value-expr="value" \
  --dry-run
echo ""

# 示例5: 同时转换键和值
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 5: 同时转换键和值${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "命令："
echo "  rocksdb-cli transform --db $TEST_DB --cf users \\"
echo "    --key-expr=\"key.upper()\" \\"
echo "    --expr=\"import json; d=json.loads(value); d['name']=d['name'].upper() if 'name' in d else d['name']; json.dumps(d)\" \\"
echo "    --dry-run --limit=3"
echo ""
echo "预览结果："
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --key-expr="key.upper()" \
  --expr="import json; d=json.loads(value); d['name']=d['name'].upper() if 'name' in d else d['name']; json.dumps(d)" \
  --dry-run --limit=3
echo ""

# 实际执行一个转换
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}示例 6: 实际执行转换（将冒号替换为下划线）${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "步骤1: 先预览"
./rocksdb-cli transform --db "$TEST_DB" --cf users \
  --key-expr="key.replace(':', '_')" \
  --value-expr="value" \
  --dry-run --limit=2
echo ""

read -p "确认执行转换吗？(输入 'yes' 继续): " confirm
if [ "$confirm" = "yes" ]; then
    echo ""
    echo "步骤2: 执行转换..."
    ./rocksdb-cli transform --db "$TEST_DB" --cf users \
      --key-expr="key.replace(':', '_')" \
      --value-expr="value" \
      --limit=2
    echo ""

    echo "步骤3: 查看转换后的数据"
    ./rocksdb-cli --db "$TEST_DB" scan --cf users --limit=10 --pretty
else
    echo "取消执行"
fi
echo ""

# 清理
echo -e "${GREEN}演示完成！${NC}"
echo ""
echo "提示："
echo "  • 总是先使用 --dry-run 预览变更"
echo "  • 使用 --limit=10 在小数据集上测试"
echo "  • 重要数据务必先备份"
echo "  • 查看详细文档: docs/KEY_TRANSFORMATION_EXAMPLES.md"
echo ""
echo "测试数据库位置: $TEST_DB"
echo "你可以继续使用这个数据库进行测试"
echo ""
echo -e "${BLUE}================================================${NC}"
