package adapters

import (
	"es-spectre/internal/model"
	"fmt"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql" // Default driver
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

	// 映射驱动类型
	driver := "mysql"
	switch dbType {
	case "mariadb", "mysql":
		driver = "mysql"
	case "pg", "kingbase", "highgo", "vastbase":
		driver = "postgres"
	}
	db, err := sqlx.Open(driver, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", dbType, err)
	}

	// Optimize Pool
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
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

func (a *GenericAdapter) wrapIdent(ident string) string {
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
	case "kingbase", "pg", "highgo", "vastbase", "dm", "shentong":
		return fmt.Sprintf("\"%s\"", ident)
	default:
		return ident
	}
}

func (a *GenericAdapter) getFullTableName(tableName string) string {
	if a.schema != "" {
		return fmt.Sprintf("%s.%s", a.schema, tableName)
	}
	return tableName
}

func (a *GenericAdapter) QueryDictItems(tableName, codeCol, keyCol, valCol, dicCode string) ([]model.DictItem, error) {
	var items []model.DictItem
	fullTable := a.getFullTableName(tableName)

	// 核心修复：标识符转义保护
	wCode := a.wrapIdent(codeCol)
	wKey := a.wrapIdent(keyCol)
	wVal := a.wrapIdent(valCol)
	wTable := a.wrapIdent(fullTable)

	var rawQuery string
	switch a.dbType {
	case "mariadb", "mysql":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS CHAR) AS item_value, %s AS item_text FROM %s WHERE UPPER(%s) = UPPER(?)",
			wCode, wKey, wVal, wTable, wCode)
	case "kingbase", "pg", "highgo", "vastbase":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS TEXT) AS item_value, %s AS item_text FROM %s WHERE UPPER(%s) = UPPER(?)",
			wCode, wKey, wVal, wTable, wCode)
	case "dm":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS VARCHAR(255)) AS item_value, %s AS item_text FROM %s WHERE UPPER(%s) = UPPER(?)",
			wCode, wKey, wVal, wTable, wCode)
	case "shentong":
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, CAST(%s AS VARCHAR) AS item_value, %s AS item_text FROM %s WHERE UPPER(%s) = UPPER(?)",
			wCode, wKey, wVal, wTable, wCode)
	default:
		rawQuery = fmt.Sprintf("SELECT %s AS dict_code, %s AS item_value, %s AS item_text FROM %s WHERE UPPER(%s) = UPPER(?)",
			wCode, wKey, wVal, wTable, wCode)
	}

	query := a.db.Rebind(rawQuery)
	err := a.db.Select(&items, query, dicCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query dict items from %s: %w", fullTable, err)
	}
	return items, nil
}

func (a *GenericAdapter) SearchDictCodes(tableName, codeCol, keyword string) ([]string, error) {
	var codes []string
	fullTable := a.getFullTableName(tableName)

	wCode := a.wrapIdent(codeCol)
	wTable := a.wrapIdent(fullTable)

	rawQuery := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE UPPER(%s) LIKE UPPER(?)",
		wCode, wTable, wCode)
	query := a.db.Rebind(rawQuery)

	err := a.db.Select(&codes, query, "%"+keyword+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to search dict codes in %s: %w", fullTable, err)
	}
	return codes, nil
}

func (a *GenericAdapter) Ping() error {
	return a.db.Ping()
}
