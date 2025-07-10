# 🔄 从 go-prompt 迁移到 Cobra 的完整指南

## 📋 概述

这份指南将帮助您将当前基于 go-prompt 的交互式 CLI 迁移到基于 Cobra 的子命令架构。

## 🎯 迁移策略对比

### 当前架构 (go-prompt)
```
rocksdb-cli --db=path [启动 REPL]
  ├── get <key>           (REPL 命令)
  ├── put <key> <value>   (REPL 命令)
  ├── search --key=pattern (REPL 命令)
  └── GraphChain 交互模式
```

### 目标架构 (Cobra)
```
rocksdb-cli
  ├── repl --db=path      (交互模式)
  ├── get --db=path <key> (直接命令)
  ├── put --db=path <key> <value>
  ├── search --db=path --key=pattern
  ├── ai --db=path [query] (GraphChain)
  └── scan --db=path [start] [end]
```

## 🚀 实施步骤

### 步骤 1: 创建新的 Cobra 入口文件

```bash
# 备份现有文件
mv cmd/main.go cmd/main-old.go

# 创建新的 Cobra 版本
```

### 步骤 2: 新的 main.go 结构

```go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"rocksdb-cli/internal/command"
	"rocksdb-cli/internal/db"
	"rocksdb-cli/internal/graphchain"
	"rocksdb-cli/internal/jsonutil"
	"rocksdb-cli/internal/util"

	"github.com/spf13/cobra"
)

var (
	// 全局标志
	dbPath     string
	readOnly   bool
	configPath string
	pretty     bool
)

// 根命令
var rootCmd = &cobra.Command{
	Use:   "rocksdb-cli",
	Short: "一个强大的 RocksDB 数据库 CLI 工具",
	Long: `RocksDB CLI 是一个用于与 RocksDB 数据库交互的综合命令行工具。
支持列族、智能键转换、搜索、导出和 AI 驱动的查询。`,
}

// REPL 命令 - 保持现有交互体验
var replCmd = &cobra.Command{
	Use:   "repl",
	Short: "启动交互式 REPL 模式",
	Long:  `启动用于数据库操作的交互式读取-求值-打印循环`,
	Run: func(cmd *cobra.Command, args []string) {
		// 重用现有的 REPL 逻辑
		rdb := openDatabase()
		defer rdb.Close()
		
		// 使用现有的 repl.Start 函数
		repl.Start(rdb)
	},
}

// 直接命令模式
var getCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "根据键获取值",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()
		
		cf := getColumnFamily(cmd)
		key := args[0]
		
		value, err := rdb.GetCF(cf, key)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Key: %s\n", util.FormatKey(key))
		if pretty {
			fmt.Printf("Value: %s\n", jsonutil.PrettyPrintWithNestedExpansion(value))
		} else {
			fmt.Printf("Value: %s\n", value)
		}
	},
}

// 搜索命令
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "搜索键和值",
	Long:  `模糊搜索键和/或值，支持各种选项包括 .NET tick 转换`,
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()
		
		keyPattern, _ := cmd.Flags().GetString("key")
		valuePattern, _ := cmd.Flags().GetString("value")
		tick, _ := cmd.Flags().GetBool("tick")
		limit, _ := cmd.Flags().GetInt("limit")
		
		if keyPattern == "" && valuePattern == "" {
			fmt.Println("Error: 必须指定至少 --key 或 --value 模式")
			os.Exit(1)
		}
		
		opts := db.SearchOptions{
			KeyPattern:   keyPattern,
			ValuePattern: valuePattern,
			Tick:         tick,
			Limit:        limit,
		}
		
		cf := getColumnFamily(cmd)
		results, err := rdb.SearchCF(cf, opts)
		if err != nil {
			fmt.Printf("搜索失败: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("找到 %d 个匹配项\n", len(results.Results))
		for i, result := range results.Results {
			fmt.Printf("[%d] Key: %s\n", i+1, result.Key)
			if pretty {
				fmt.Printf("    Value: %s\n", jsonutil.PrettyPrintWithNestedExpansion(result.Value))
			} else {
				fmt.Printf("    Value: %s\n", result.Value)
			}
		}
	},
}

// AI 助手命令
var aiCmd = &cobra.Command{
	Use:   "ai [query]",
	Short: "AI 驱动的数据库助手",
	Long: `启动 AI 驱动的 GraphChain 助手进行自然语言数据库查询。
如果没有提供查询，则启动交互模式。`,
	Run: func(cmd *cobra.Command, args []string) {
		rdb := openDatabase()
		defer rdb.Close()
		
		if len(args) == 0 {
			// 交互模式 - 重用现有 GraphChain 逻辑
			runGraphChainInteractive(rdb)
		} else {
			// 单次查询模式
			query := strings.Join(args, " ")
			runGraphChainQuery(rdb, query)
		}
	},
}

// 辅助函数
func openDatabase() db.KeyValueDB {
	var rdb db.KeyValueDB
	var err error
	
	if readOnly {
		rdb, err = db.OpenReadOnly(dbPath)
	} else {
		rdb, err = db.Open(dbPath)
	}
	
	if err != nil {
		fmt.Printf("无法打开数据库: %v\n", err)
		os.Exit(1)
	}
	
	return rdb
}

func getColumnFamily(cmd *cobra.Command) string {
	cf, _ := cmd.Flags().GetString("cf")
	if cf == "" {
		return "default"
	}
	return cf
}

func runGraphChainInteractive(database db.KeyValueDB) {
	// 重用现有的 GraphChain 交互逻辑
	// 这里可以直接调用现有的 runGraphChainAgent 函数
	// 或者重构成独立的函数
}

func runGraphChainQuery(database db.KeyValueDB, query string) {
	// 单次 GraphChain 查询逻辑
}

func init() {
	// 全局标志
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "RocksDB 数据库路径（必需）")
	rootCmd.PersistentFlags().BoolVar(&readOnly, "read-only", false, "以只读模式打开数据库")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config/graphchain.yaml", "GraphChain 配置文件路径")
	rootCmd.PersistentFlags().BoolVar(&pretty, "pretty", false, "美化打印 JSON 值")
	
	// 为需要的命令添加列族标志
	getCmd.Flags().StringP("cf", "c", "default", "列族")
	searchCmd.Flags().StringP("cf", "c", "default", "列族")
	
	// 搜索命令标志
	searchCmd.Flags().String("key", "", "要搜索的键模式")
	searchCmd.Flags().String("value", "", "要搜索的值模式")
	searchCmd.Flags().Bool("tick", false, "将键视为 .NET tick 时间并转换为 UTC 字符串格式")
	searchCmd.Flags().Int("limit", 50, "限制搜索结果")
	
	// 添加所有命令到根命令
	rootCmd.AddCommand(replCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(searchCmd)
	rootCmd.AddCommand(aiCmd)
	
	// 标记必需标志
	rootCmd.MarkPersistentFlagRequired("db")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

## 🔧 迁移优势

### 1. 保持向后兼容
```bash
# 现有用户仍可使用 REPL
rocksdb-cli repl --db=/path/to/db

# 新用户可以使用直接命令
rocksdb-cli get --db=/path/to/db mykey
rocksdb-cli search --db=/path/to/db --key="user:*" --tick
```

### 2. 更好的脚本支持
```bash
# 批量操作脚本
for key in user:1 user:2 user:3; do
  rocksdb-cli get --db=/path/to/db "$key"
done

# CI/CD 集成
rocksdb-cli search --db=/path/to/db --key="error:*" | grep "critical"
```

### 3. 改进的帮助系统
```bash
rocksdb-cli --help
rocksdb-cli search --help
rocksdb-cli ai --help
```

## 📝 具体实施建议

### 方案 A: 渐进式迁移
1. 保留现有 `main.go` 作为 `main-legacy.go`
2. 创建新的 Cobra 版本
3. 通过环境变量选择使用哪个版本
4. 逐步迁移用户

### 方案 B: 混合模式
1. 主入口使用 Cobra
2. REPL 模式继续使用 go-prompt
3. 直接命令使用 Cobra 子命令
4. 最佳的用户体验平衡

### 方案 C: 完全替换
1. 完全移除 go-prompt
2. 使用 Cobra 的交互式提示
3. 或者实现简单的 readline 循环
4. 最小化依赖

## 🎯 推荐方案：混合模式 (方案 B)

这是最佳选择，因为：
- ✅ 保持现有用户体验
- ✅ 添加脚本友好的直接命令
- ✅ 最小化破坏性变更
- ✅ 逐步现代化架构

## 🚀 实施步骤

1. **第一阶段**: 添加 Cobra 子命令
2. **第二阶段**: 保持 REPL 使用 go-prompt
3. **第三阶段**: 评估用户反馈
4. **第四阶段**: 决定是否进一步迁移

这种方法让您在保持现有功能的同时获得 Cobra 的好处！ 