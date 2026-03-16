package service

import (
	"es-spectre/pkg/core/model"
	"es-spectre/pkg/core/repository"
	"strings"
	"sync"
)

type DictEngine struct {
	repo      repository.DictRepository
	cache     map[string][]model.DictItem
	lookup    map[string]map[string]string // 关键优化：[dictCode][itemValue]itemText 的 O(1) 映射表
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
		lookup:    make(map[string]map[string]string),
		tableName: tableName,
		codeCol:   codeCol,
		keyCol:    keyCol,
		valCol:    valCol,
	}
}

// MatchField attempts to match an ES field name to a dictionary code with optional manual override
func (e *DictEngine) MatchField(fieldName string, mappingCode string) (string, []model.DictItem, error) {
	dictCode := strings.ToUpper(mappingCode)
	if dictCode == "" {
		dictCode = strings.ToUpper(fieldName)
	}

	// 1. 第一轮检查 (读锁)
	e.mu.RLock()
	if items, ok := e.cache[dictCode]; ok {
		e.mu.RUnlock()
		return dictCode, items, nil
	}
	e.mu.RUnlock()

	// 2. 查库前加写锁，防止并发击穿 (Double-Checked Locking)
	e.mu.Lock()
	defer e.mu.Unlock()

	// 二次检查，可能其他协程已经填好了
	if items, ok := e.cache[dictCode]; ok {
		return dictCode, items, nil
	}

	items, err := e.repo.QueryDictItems(e.tableName, e.codeCol, e.keyCol, e.valCol, dictCode)
	if err != nil {
		return dictCode, nil, err
	}

	// 3. 构建高性能 O(1) 查找表
	lookupMap := make(map[string]string)
	for _, item := range items {
		val := strings.TrimSpace(item.ItemValue)
		lookupMap[val] = item.ItemText
		// 增强：存入一份归一化后的键 (去掉前导0)，以支持 O(1) 的数值兼容匹配
		normVal := strings.TrimLeft(val, "0")
		if normVal != "" && normVal != val {
			if _, exists := lookupMap[normVal]; !exists {
				lookupMap[normVal] = item.ItemText
			}
		}
	}

	e.cache[dictCode] = items
	e.lookup[dictCode] = lookupMap

	return dictCode, items, nil
}

// TranslateValue maps a raw value to its dictionary text
func (e *DictEngine) TranslateValue(dictCode string, value string) string {
	dictKey := strings.ToUpper(dictCode)
	v := strings.TrimSpace(value)

	e.mu.RLock()
	lookup, ok := e.lookup[dictKey]
	e.mu.RUnlock()

	if !ok {
		// 如果未命中，尝试走 MatchField 逻辑（由于 MatchField 在 FetchRealReport 循环外调用，此处理论上不应发生，但为了逻辑严密保留）
		_, _, err := e.MatchField("", dictKey)
		if err != nil {
			return value
		}
		e.mu.RLock()
		lookup = e.lookup[dictKey]
		e.mu.RUnlock()
	}

	if lookup == nil {
		return value
	}

	// 1. 首先尝试直接匹配 (性能最高)
	if text, ok := lookup[v]; ok {
		return text
	}

	// 2. 尝试数值归一化匹配 (去掉前导0，如 "01" 匹配 "1")
	normV := strings.TrimLeft(v, "0")
	if normV != "" && normV != v {
		if text, ok := lookup[normV]; ok {
			return text
		}
	}

	return value
}

func (e *DictEngine) GetAvailableKeys(dictCode string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	items := e.cache[strings.ToUpper(dictCode)]
	var keys []string
	for _, it := range items {
		keys = append(keys, it.ItemValue)
	}
	return strings.Join(keys, ", ")
}

func (e *DictEngine) GetCacheSize(dictCode string) int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.cache[strings.ToUpper(dictCode)])
}
