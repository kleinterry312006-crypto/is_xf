package service

import (
	"es-spectre/internal/model"
	"es-spectre/internal/repository"
	"strings"
	"sync"
)

type DictEngine struct {
	repo      repository.DictRepository
	cache     map[string][]model.DictItem
	mu        sync.RWMutex
	tableName string
	codeCol   string
	keyCol    string
	valCol    string
}

func NewDictEngine(repo repository.DictRepository, tableName, codeCol, keyCol, valCol string) *DictEngine {
	return &DictEngine{
		repo:      repo,
		cache:     make(map[string][]model.DictItem),
		tableName: tableName,
		codeCol:   codeCol,
		keyCol:    keyCol,
		valCol:    valCol,
	}
}

// MatchField attempts to match an ES field name to a dictionary code with optional override
func (e *DictEngine) MatchField(fieldName string, mappingCode string) (string, []model.DictItem, error) {
	dictCode := mappingCode
	if dictCode == "" {
		dictCode = strings.ToUpper(fieldName)
	}

	e.mu.RLock()
	if items, ok := e.cache[dictCode]; ok {
		e.mu.RUnlock()
		return dictCode, items, nil
	}
	e.mu.RUnlock()

	items, err := e.repo.QueryDictItems(e.tableName, e.codeCol, e.keyCol, e.valCol, dictCode)
	if err != nil {
		return dictCode, nil, err
	}

	e.mu.Lock()
	e.cache[dictCode] = items
	e.mu.Unlock()

	return dictCode, items, nil
}

// TranslateValue maps a raw value to its dictionary text
func (e *DictEngine) TranslateValue(dictCode string, value string) string {
	dictKey := strings.ToUpper(dictCode)

	e.mu.RLock()
	items, ok := e.cache[dictKey]
	e.mu.RUnlock()

	if !ok {
		var err error
		items, err = e.repo.QueryDictItems(e.tableName, e.codeCol, e.keyCol, e.valCol, dictKey)
		if err != nil {
			return value
		}
		e.mu.Lock()
		e.cache[dictKey] = items
		e.mu.Unlock()
	}

	target := strings.TrimSpace(value)
	for _, item := range items {
		iv := strings.TrimSpace(item.ItemValue)
		// 1. 标准匹配 (1 == 1)
		// 2. 数值兼容匹配 (01 == 1)
		if strings.EqualFold(iv, target) || (strings.TrimLeft(iv, "0") == strings.TrimLeft(target, "0") && iv != "" && target != "") {
			return item.ItemText
		}
	}

	return value
}

func (e *DictEngine) GetAvailableKeys(dictCode string) []string {
	dictKey := strings.ToUpper(dictCode)
	e.mu.RLock()
	defer e.mu.RUnlock()
	items := e.cache[dictKey]
	var keys []string
	for _, item := range items {
		keys = append(keys, item.ItemValue)
	}
	return keys
}
