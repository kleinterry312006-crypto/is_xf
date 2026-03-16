package adapters

import (
	"database/sql"
	"es-spectre/pkg/core/model"
	"fmt"
	"strings"

	"sync"
	"time"

	_ "gitee.com/XuguDB/go-xugu-driver" // Correct Xugu driver (Gitee)
	_ "gitee.com/chunanyong/dm"         // Correct Dameng driver (Gitee)
	_ "github.com/alexbrainman/odbc"    // ODBC driver for domestic DBs
	_ "github.com/go-sql-driver/mysql"  // Default driver
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Postgres driver (used for Kingbase/Highgo/Vastbase)
)

var (
	driverMu sync.Mutex
)

type GenericAdapter struct {
	db     *sqlx.DB
	dbType string
	schema string
}

func NewGenericAdapter(dbType, connStr, schema, driverPath, driverClass string) (*GenericAdapter, error) {
	driverMu.Lock()
	defer driverMu.Unlock()

	// Map UI dbType to actual Go driver names
	driverMap := map[string]string{
		"mariadb":  "mysql",
		"mysql":    "mysql",
		"pg":       "postgres",
		"kingbase": "postgres",
		"dm":       "dm",
		"highgo":   "postgres",
		"vastbase": "postgres",
		"shentong": "shentong",
		"xugu":     "xugu",
		"odbc":     "odbc",
	}

	// 选择驱动
	driver := driverMap[dbType]
	if driver == "" {
		driver = "mysql"
	}

	// 特殊处理：如果指定了 shentong 且原生驱动未就绪，强制切换至 postgres 驱动（神通 7.0+ 兼容 PG 协议）
	isShentongFallback := false
	if dbType == "shentong" {
		nativeFound := false
		for _, d := range sql.Drivers() {
			if d == "shentong" {
				nativeFound = true
				break
			}
		}
		if !nativeFound {
			driver = "postgres"
			isShentongFallback = true
		}
	}

	// 统一处理 DSN 转换：如果最终使用 postgres 驱动，但连接串是 MySQL 格式，必须转换
	if driver == "postgres" && strings.Contains(connStr, "@tcp(") {
		// 解析 user:pass@tcp(host:port)/dbname
		// 1. 找 @tcp(
		idxAt := strings.Index(connStr, "@tcp(")
		if idxAt > 0 {
			authPart := connStr[:idxAt]
			remaining := connStr[idxAt+5:] // 跳过 @tcp(

			// 2. 找用户名密码
			idxColon := strings.Index(authPart, ":")
			user := authPart
			pass := ""
			if idxColon >= 0 {
				user = authPart[:idxColon]
				pass = authPart[idxColon+1:]
			}

			// 3. 找地址和库名
			idxEndParen := strings.Index(remaining, ")")
			if idxEndParen > 0 {
				addr := remaining[:idxEndParen]
				dbPart := remaining[idxEndParen+1:]
				dbname := strings.TrimPrefix(dbPart, "/")

				host := addr
				port := "54321" // 神通默认 PG 端口
				idxAddrColon := strings.LastIndex(addr, ":")
				if idxAddrColon >= 0 {
					host = addr[:idxAddrColon]
					port = addr[idxAddrColon+1:]
				}

				// 重新拼装为 Postgres 格式
				connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
					host, port, user, pass, dbname)
			}
		}
	}

	// Dynamic Driver Loading Strategy (Platform Dependent)
	// If driverPath is provided and it's a DM or Kingbase,
	// we handle the specialized loading logic here.
	if driverPath != "" && (dbType == "dm" || dbType == "kingbase") {
		// In a real environment, this is where you'd use a dynamic loader
		// to register the driver from the .dll/.so.
		// For now, we assume the environment has the driver registered or
		// the user is providing a standard DSN.
	}

	// 为 MySQL/MariaDB 增加连接超时控制，防止界面卡死
	if (dbType == "mysql" || dbType == "mariadb") && !strings.Contains(connStr, "timeout=") {
		if strings.Contains(connStr, "?") {
			connStr += "&timeout=5s"
		} else {
			connStr += "?timeout=5s"
		}
	}

	// 检查驱动是否可用
	found := false
	for _, d := range sql.Drivers() {
		if d == driver {
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("数据库驱动 [%s] 未在程序中编译 (或者通过 fallback 方式也无法找到驱动)。", driver)
	}

	db, err := sqlx.Open(driver, connStr)
	if err != nil {
		if dbType == "shentong" && driver == "postgres" {
			return nil, fmt.Errorf("试图通过 PostgreSQL 兼容模式连接神通数据库失败: %w。请确认数据库已开启 PG 兼容，或尝试使用 [odbc] 模式。", err)
		}
		return nil, fmt.Errorf("failed to open %s (driver: %s): %w", dbType, driver, err)
	}

	// Optimize Connection Pool for Production
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection immediately
	if err := db.Ping(); err != nil {
		db.Close()
		if isShentongFallback {
			return nil, fmt.Errorf("神通数据库连接成功但 Ping 失败 (强制使用 PG 协议模式): %w。请确认数据库已开启 PG 兼容性。", err)
		}
		return nil, fmt.Errorf("failed to ping %s: %w", dbType, err)
	}

	return &GenericAdapter{
		db:     db,
		dbType: dbType,
		schema: schema,
	}, nil
}

func (a *GenericAdapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

func (a *GenericAdapter) getFullTableName(tableName string) string {
	if a.schema != "" {
		return fmt.Sprintf("%s.%s", a.schema, tableName)
	}
	return tableName
}

func (a *GenericAdapter) wrapIdent(ident string) string {
	// 处理模式名.表名的情况
	if strings.Contains(ident, ".") {
		parts := strings.Split(ident, ".")
		newParts := make([]string, len(parts))
		for i, p := range parts {
			newParts[i] = a.wrapIdent(p)
		}
		return strings.Join(newParts, ".")
	}

	switch a.dbType {
	case "mariadb", "mysql":
		return fmt.Sprintf("`%s`", ident)
	case "kingbase", "pg", "highgo", "vastbase", "dm", "shentong", "xugu":
		return fmt.Sprintf("\"%s\"", ident)
	default:
		return ident
	}
}

func (a *GenericAdapter) QueryDictItems(tableName, codeCol, keyCol, valCol, dicCode string) ([]model.DictItem, error) {
	var items []model.DictItem
	fullTable := a.getFullTableName(tableName)

	// 关键修复：对所有列名和表名进行标识符转义保护
	wCode := a.wrapIdent(codeCol)
	wKey := a.wrapIdent(keyCol)
	wVal := a.wrapIdent(valCol)
	wTable := a.wrapIdent(fullTable)

	// SQL 方言自适应逻辑 (全量覆盖 8 种目标数据库)
	var rawQuery string
	switch a.dbType {
	case "mariadb", "mysql":
		// 使用 = ? 替代 UPPER(col) = UPPER(?) 以确保命中索引
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS CHAR) AS item_value, %s AS item_text FROM %s WHERE %s = ?",
			wCode, wKey, wVal, wTable, wCode)
	case "kingbase", "pg", "highgo", "vastbase":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS TEXT) AS item_value, %s AS item_text FROM %s WHERE %s = ?",
			wCode, wKey, wVal, wTable, wCode)
	case "dm":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS VARCHAR(255)) AS item_value, %s AS item_text FROM %s WHERE %s = ?",
			wCode, wKey, wVal, wTable, wCode)
	case "shentong", "xugu":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS VARCHAR) AS item_value, %s AS item_text FROM %s WHERE %s = ?",
			wCode, wKey, wVal, wTable, wCode)
	default:
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, %s AS item_value, %s AS item_text FROM %s WHERE %s = ?",
			wCode, wKey, wVal, wTable, wCode)
	}

	query := a.db.Rebind(rawQuery)
	// 在传入参数前转换为大写，确保匹配
	err := a.db.Select(&items, query, strings.ToUpper(dicCode))
	if err != nil {
		return nil, fmt.Errorf("查询字典失败 (方言: %s, 表: %s): %w", a.dbType, fullTable, err)
	}
	return items, nil
}

func (a *GenericAdapter) SearchDictCodes(tableName, codeCol, keyword string) ([]string, error) {
	var codes []string
	fullTable := a.getFullTableName(tableName)

	wCode := a.wrapIdent(codeCol)
	wTable := a.wrapIdent(fullTable)

	rawQuery := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s LIKE ?",
		wCode, wTable, wCode)
	query := a.db.Rebind(rawQuery)

	err := a.db.Select(&codes, query, "%"+strings.ToUpper(keyword)+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search dict codes in %s: %w", fullTable, err)
	}
	return codes, nil
}

func (a *GenericAdapter) Ping() error {
	return a.db.Ping()
}
