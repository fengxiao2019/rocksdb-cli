# 文档网站使用指南

## ✅ 已完成的工作

文档网站已成功创建并可以使用！

### 创建的内容

1. **配置文件** - `mkdocs.yml`
   - Material for MkDocs 主题
   - 中文界面支持
   - 完整的导航结构
   - 代码高亮和语法支持
   - Mermaid 图表支持

2. **文档页面** - `docs_site/` 目录
   - ✅ 首页（index.md）
   - ✅ 快速开始指南
   - ✅ Web UI 完整文档
   - ✅ 安装指南
   - ✅ Docker 使用指南
   - ✅ 功能概览
   - ✅ 命令参考（占位）
   - ✅ FAQ 和故障排查
   - ✅ Key Transformation 示例（复制自现有文档）
   - ✅ MCP 服务器文档（复制自现有文档）

3. **样式和脚本**
   - ✅ 自定义 CSS（extra.css）
   - ✅ 自定义 JavaScript（extra.js）

## 🚀 如何使用

### 本地预览

```bash
# 1. 安装依赖（首次）
pip install mkdocs mkdocs-material mkdocs-minify-plugin

# 2. 启动开发服务器
cd /Users/daoma/wkspace/rocksdb-cli
mkdocs serve

# 3. 在浏览器中访问
# http://127.0.0.1:8000/rocksdb-cli/
```

### 构建静态网站

```bash
# 构建到 site/ 目录
mkdocs build

# 查看构建结果
ls -la site/
```

### 部署到 GitHub Pages

```bash
# 一键部署（推荐）
mkdocs gh-deploy

# 这会：
# 1. 自动构建网站
# 2. 推送到 gh-pages 分支
# 3. 启用 GitHub Pages
```

## 📂 文档结构

```
rocksdb-cli/
├── mkdocs.yml              # MkDocs 配置文件
├── docs_site/              # 文档源文件目录
│   ├── index.md            # 首页
│   ├── getting-started/    # 快速开始
│   │   ├── installation.md
│   │   ├── quick-start.md
│   │   └── docker.md
│   ├── features/           # 核心功能
│   │   ├── overview.md
│   │   ├── web-ui.md
│   │   ├── transform.md
│   │   ├── graphchain-ai.md
│   │   ├── mcp-server.md
│   │   └── search.md
│   ├── commands/           # 命令参考
│   ├── examples/           # 示例教程
│   ├── api/                # API 文档
│   ├── development/        # 开发者文档
│   ├── reference/          # 参考资料
│   ├── stylesheets/        # 自定义样式
│   └── javascripts/        # 自定义脚本
└── site/                   # 构建输出（自动生成）
```

## 📝 添加新文档

### 1. 创建新页面

```bash
# 创建新的 Markdown 文件
touch docs_site/features/new-feature.md
```

### 2. 编写内容

```markdown
# 新功能

这是一个新功能的文档。

## 使用方法

\`\`\`bash
rocksdb-cli new-feature --help
\`\`\`
```

### 3. 添加到导航

编辑 `mkdocs.yml`：

```yaml
nav:
  - 核心功能:
    - 新功能: features/new-feature.md  # 添加这行
```

### 4. 预览

```bash
mkdocs serve
# 检查 http://127.0.0.1:8000
```

## 🎨 自定义样式

### 修改颜色主题

编辑 `mkdocs.yml`：

```yaml
theme:
  palette:
    - scheme: default
      primary: blue     # 改为你喜欢的颜色
      accent: cyan
```

### 添加自定义CSS

编辑 `docs_site/stylesheets/extra.css`：

```css
/* 添加自定义样式 */
.md-button {
    border-radius: 8px;
}
```

## 📚 Markdown 技巧

### 警告框

```markdown
!!! note "提示"
    这是一个提示信息

!!! warning "警告"
    请注意这一点

!!! tip "小技巧"
    这会很有帮助
```

### 代码块标签页

```markdown
=== "Python"
    \`\`\`python
    print("Hello")
    \`\`\`

=== "Bash"
    \`\`\`bash
    echo "Hello"
    \`\`\`
```

### Mermaid 图表

```markdown
\`\`\`mermaid
graph LR
    A[开始] --> B[处理]
    B --> C[结束]
\`\`\`
```

## 🔧 下一步完善

以下内容可以继续完善：

### 高优先级
- [ ] 完善各个命令的详细文档（commands/）
- [ ] 添加 GraphChain AI 使用示例
- [ ] 补充 API 文档
- [ ] 添加更多实战示例

### 中优先级
- [ ] 添加架构设计文档
- [ ] 编写贡献指南
- [ ] 创建开发者指南
- [ ] 添加截图和演示GIF

### 低优先级
- [ ] 添加更新日志
- [ ] 创建路线图
- [ ] 多语言支持（英文版）

## 📊 文档状态

### 已完成 ✅
- 首页和快速开始
- Web UI 完整文档
- 安装和 Docker 指南
- Key Transformation 详细示例
- MCP 服务器文档
- 基础导航结构
- 样式和主题配置

### 占位（需完善）⚠️
- 各个命令详细文档
- GraphChain AI 示例
- 数据迁移场景
- API 参考
- 开发者文档
- 架构设计

## 🎯 推荐的完善顺序

1. **命令文档**（最重要）
   - 从最常用的命令开始
   - get/put、scan、search、transform

2. **示例教程**
   - 真实使用场景
   - 从简单到复杂的示例

3. **API 文档**
   - Web API 端点说明
   - 请求/响应示例

4. **开发者文档**
   - 架构说明
   - 贡献指南
   - 测试指南

## 📖 参考资源

- [MkDocs 官方文档](https://www.mkdocs.org/)
- [Material for MkDocs](https://squidfunk.github.io/mkdocs-material/)
- [Markdown 语法指南](https://www.markdownguide.org/)
- [Mermaid 图表](https://mermaid.js.org/)

## 🤝 贡献文档

欢迎改进文档！步骤：

1. 编辑 `docs_site/` 下的文件
2. 本地预览确认
3. 提交 Pull Request

## 💡 提示

- 使用 `mkdocs serve` 实时预览变更
- 保持导航结构清晰简洁
- 添加示例代码和实际用例
- 使用警告框突出重要信息
- 添加内部链接方便导航

---

**文档网站已就绪！** 🎉

现在你可以：
1. 运行 `mkdocs serve` 查看效果
2. 继续完善内容
3. 部署到 GitHub Pages

有问题？查看 `docs_site/README.md` 了解更多细节。
