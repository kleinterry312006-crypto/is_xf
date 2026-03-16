# ES-Spectre (ES 幽灵分析师)

一款集成 ES 数据聚合、异构数据库字典映射、多维交互展示的独立终端分析工具。

## 功能特性
- **多维聚合**：通过点选实现 ES 字段交叉统计。
- **字典引擎**：自动将 ES 枚举值映射为业务含义。
- **国产数据库支持**：适配达梦 (DM)、金仓 (Kingbase) 等异构数据库。
- **高级 TUI**：基于 Bubble Tea & Lipgloss 的现代化终端交互。

## 数据库连接 URL 格式说明

由于本系统采用 Go 语言开发，在“系统连接设置”中填写“独立连接 URL”时，请使用 **DSN (Data Source Name)** 格式，**切勿包含 `jdbc:` 前缀**。

| 数据库 | 推荐 DSN 格式示例 | 驱动驱动库参考 |
| :--- | :--- | :--- |
| **人大金仓 (Kingbase)** | `postgres://user:pass@127.0.0.1:54321/db?sslmode=disable` | 兼容 PostgreSQL 模式 |
| **达梦 (DM)** | `dm://user:pass@127.0.0.1:5236` | 需加载 dm 驱动 |
| **海量 (Vastbase)** | `postgres://user:pass@127.0.0.1:5432/db?sslmode=disable` | 基于 PG 核心 |
| **神通 (Shentong)** | `shentong://user:pass@127.0.0.1:2003/db` | |
| **瀚高 (Highgo)** | `postgres://user:pass@127.0.0.1:5866/db?sslmode=disable` | 基于 PG 核心 |
| **虚谷 (Xugu)** | `xugu://user:pass@127.0.0.1:5138/db` | |
| **MySQL / MariaDB** | `user:pass@tcp(127.0.0.1:3306)/db` | 标准格式 |
| **PostgreSQL** | `postgres://user:pass@127.0.0.1:5432/db?sslmode=disable` | 标准格式 |

> **提示**：如果账号密码包含特殊字符（如 `@`, `:`, `/`），请先进行 URL 编码（例如 `@` 编码为 `%40`），否则会导致解析失败。
