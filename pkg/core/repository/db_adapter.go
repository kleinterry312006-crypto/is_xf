package repository

import "es-spectre/pkg/core/model"

type DictRepository interface {
	// QueryDictItems searches for dictionary items by field name (dictCode) with dynamic mapping
	QueryDictItems(tableName, codeCol, keyCol, valCol, dictCode string) ([]model.DictItem, error)
	// SearchDictCodes performs fuzzy search on dictionary codes with dynamic mapping
	SearchDictCodes(tableName, codeCol, keyword string) ([]string, error)
	// Ping checks the connection status
	Ping() error
}
