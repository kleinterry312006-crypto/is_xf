package main

import (
	"context"
	"encoding/json"
	"es-spectre/pkg/core/config"
	"es-spectre/pkg/core/repository/adapters"
	"es-spectre/pkg/core/service"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"runtime"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/xuri/excelize/v2"
)

func (a *App) writeLog(content string) {
	logDir := "logs"
	os.MkdirAll(logDir, 0755)
	logPath := filepath.Join(logDir, "query.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Log error: %v\n", err)
		return
	}
	defer f.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	f.WriteString(fmt.Sprintf("[%s] %s\n", timestamp, content))
}

// Helper to get portable paths
func getPath(target string) string {
	localPath := filepath.Join("configs", target)
	if _, err := os.Stat(localPath); err == nil {
		return localPath
	}
	return filepath.Join("..", "configs", target)
}

type App struct {
	ctx      context.Context
	cfg      *config.Config
	esClient *service.ESClient
	adapter  *adapters.GenericAdapter
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	cfgPath := getPath("config.yaml")
	cfg, err := config.LoadConfig(cfgPath)
	if err != nil {
		fmt.Printf("Config error: %v\n", err)
		return
	}
	a.cfg = cfg

	// Init ES
	esPort := cfg.Elasticsearch.Port
	if esPort == 0 {
		esPort = 9200
	}
	addr := fmt.Sprintf("http://%s:%d", cfg.Elasticsearch.IP, esPort)
	esClient, err := service.NewESClient(addr, cfg.Elasticsearch.User, cfg.Elasticsearch.Password)
	if err == nil {
		a.esClient = esClient
	}

	// Init DB Adapter
	connStr := a.generateDSN(cfg.Database.Type, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName, cfg.Database.User, cfg.Database.Password, cfg.Database.Schema, cfg.Database.ConnUrl)
	adapter, err := adapters.NewGenericAdapter(cfg.Database.Type, connStr, cfg.Database.Schema, cfg.Database.DriverPath, cfg.Database.DriverClass)
	if err == nil {
		a.adapter = adapter
	}
}

func (a *App) GetConfig() config.Config {
	if a.cfg == nil {
		return config.Config{}
	}
	return *a.cfg
}

func (a *App) SaveConfig(newCfg config.Config) string {
	cfgPath := getPath("config.yaml")
	err := config.UpdateAndSaveConfig(cfgPath, &newCfg)
	if err != nil {
		return fmt.Sprintf("保存失败: %v", err)
	}
	a.cfg = &newCfg

	// Re-init ES
	esPort := a.cfg.Elasticsearch.Port
	if esPort == 0 {
		esPort = 9200
	}
	addr := fmt.Sprintf("http://%s:%d", a.cfg.Elasticsearch.IP, esPort)
	esClient, err := service.NewESClient(addr, a.cfg.Elasticsearch.User, a.cfg.Elasticsearch.Password)
	if err == nil {
		a.esClient = esClient
	}

	// Re-init DB Adapter
	if a.adapter != nil {
		a.adapter.Close()
	}
	connStr := a.generateDSN(a.cfg.Database.Type, a.cfg.Database.Host, a.cfg.Database.Port, a.cfg.Database.DBName, a.cfg.Database.User, a.cfg.Database.Password, a.cfg.Database.Schema, a.cfg.Database.ConnUrl)
	adapter, err := adapters.NewGenericAdapter(a.cfg.Database.Type, connStr, a.cfg.Database.Schema, a.cfg.Database.DriverPath, a.cfg.Database.DriverClass)
	if err == nil {
		a.adapter = adapter
	}

	return "设置已保存"
}

func (a *App) GetESInfo() string {
	if a.esClient == nil {
		return "ES 未连接"
	}
	return fmt.Sprintf("已连接至: %s:%d", a.cfg.Elasticsearch.IP, a.cfg.Elasticsearch.Port)
}

func (a *App) TestESConnection(ip string, port int, user string, password string) string {
	addr := fmt.Sprintf("http://%s:%d", ip, port)
	client, err := service.NewESClient(addr, user, password)
	if err != nil {
		return fmt.Sprintf("连接失败: %v", err)
	}
	_, err = client.GetInfo()
	if err != nil {
		return "无法获取 ES 节点信息"
	}
	return "测试连接成功"
}

func (a *App) generateDSN(dbType, host string, port int, dbname, user, password, schema, connUrl string) string {
	if connUrl != "" {
		return connUrl
	}

	dbIdent := dbname
	if (dbType == "mariadb" || dbType == "mysql") && dbIdent == "" {
		dbIdent = schema
	}

	switch dbType {
	case "mariadb", "mysql":
		p := port
		if p == 0 {
			p = 3306
		}
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, p, dbIdent)
	case "pg", "kingbase", "highgo", "vastbase":
		p := port
		if p == 0 {
			p = 54321
		}
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, p, user, password, dbname)
		if schema != "" {
			dsn += fmt.Sprintf(" search_path=%s", schema)
		}
		return dsn
	default:
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbname)
	}
}

func (a *App) TestDBConnection(dbType, host string, port int, dbname, schema, user, password, driverPath, connUrl, driverClass string) string {
	connStr := a.generateDSN(dbType, host, port, dbname, user, password, schema, connUrl)
	adapter, err := adapters.NewGenericAdapter(dbType, connStr, schema, driverPath, driverClass)
	if err != nil {
		return fmt.Sprintf("连接创建失败: %v", err)
	}
	defer adapter.Close()

	err = adapter.Ping()
	if err != nil {
		return fmt.Sprintf("数据库响应超时或认证失败: %v", err)
	}
	return "测试成功"
}

func (a *App) SelectFile() string {
	selection, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   "选择数据库驱动",
		Filters: []wailsRuntime.FileFilter{{DisplayName: "驱动 (DLL/JAR)", Pattern: "*.dll;*.jar;*.so"}},
	})
	if err != nil {
		return ""
	}
	return selection
}

// Metadata structures
type MetadataField struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Enabled     bool   `json:"enabled"`
	MappingCode string `json:"mapping_code"`
}

type BehaviorMetadata struct {
	Standard string          `json:"standard"`
	Type     string          `json:"type"`
	TypeCode string          `json:"type_code"`
	Fields   []MetadataField `json:"fields"`
	Selected bool            `json:"selected"`
}

func (a *App) GetMetadata() []BehaviorMetadata {
	metadataPath := getPath("metadata.json")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return []BehaviorMetadata{}
	}
	var meta []BehaviorMetadata
	json.Unmarshal(data, &meta)
	return meta
}

func (a *App) SaveMetadata(meta []BehaviorMetadata) string {
	metadataPath := getPath("metadata.json")
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metadataPath, data, 0644)
	return "元数据配置已保存"
}

func (a *App) DownloadMetadataTemplate() string {
	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出业务元数据导入模板",
		DefaultFilename: "ES_下钻配置模板_V4.xlsx",
		Filters:         []wailsRuntime.FileFilter{{DisplayName: "Excel 文件", Pattern: "*.xlsx"}},
	})
	if err != nil || savePath == "" {
		return ""
	}

	f := excelize.NewFile()
	sheet := "Sheet1"
	headers := []string{"数据标准类型", "行为展示名称 (中文)", "行为代号 (英文)", "特定字段 (英文名)", "特定字段 (中文字段名)", "字典映射编码 (选填)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	samples := [][]string{
		{"日志数据标准", "主机硬件变更操作", "host_hardware_change", "hard_type", "硬件介质类型", "HARD_TYPE_DIC"},
		{"日志数据标准", "主机硬件变更操作", "host_hardware_change", "hard_op_type", "变更行为", ""},
		{"日志数据标准", "文件打印操作", "file_print", "print_copys", "打印份数", ""},
	}

	for i, row := range samples {
		for j, val := range row {
			cell, _ := excelize.CoordinatesToCellName(j+1, i+2)
			f.SetCellValue(sheet, cell, val)
		}
	}

	if err := f.SaveAs(savePath); err != nil {
		return fmt.Sprintf("导出失败: %v", err)
	}
	return "模板成功导出至: " + savePath
}

func (a *App) UploadMetadata() string {
	filePath, err := wailsRuntime.OpenFileDialog(a.ctx, wailsRuntime.OpenDialogOptions{
		Title:   "上传元数据配置文件 (全量同步模式)",
		Filters: []wailsRuntime.FileFilter{{DisplayName: "Excel 业务模板", Pattern: "*.xlsx"}},
	})
	if err != nil || filePath == "" {
		return ""
	}

	var f *excelize.File
	var openErr error
	for i := 0; i < 3; i++ {
		f, openErr = excelize.OpenFile(filePath)
		if openErr == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	if openErr != nil {
		return fmt.Sprintf("读取 Excel 失败: %v", openErr)
	}
	defer f.Close()

	rows, err := f.GetRows("Sheet1")
	if err != nil || len(rows) < 2 {
		return "未找到有效数据，请检查 Sheet1 名称及内容"
	}

	type orderKey struct{ std, code string }
	var keys []orderKey
	metaMap := make(map[orderKey]*BehaviorMetadata)

	for i, row := range rows {
		if i == 0 || len(row) < 5 {
			continue
		}

		std := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		code := strings.TrimSpace(row[2])
		fName := strings.TrimSpace(row[3])
		fLabel := strings.TrimSpace(row[4])

		var mCode string
		if len(row) >= 6 {
			mCode = strings.TrimSpace(row[5])
		}

		if std == "" || code == "" {
			continue
		}

		k := orderKey{std, code}
		if _, exists := metaMap[k]; !exists {
			metaMap[k] = &BehaviorMetadata{
				Standard: std,
				Type:     name,
				TypeCode: code,
				Fields:   []MetadataField{},
				Selected: true,
			}
			keys = append(keys, k)
		}

		isDuplicate := false
		for _, ef := range metaMap[k].Fields {
			if ef.Name == fName {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate && fName != "" {
			metaMap[k].Fields = append(metaMap[k].Fields, MetadataField{
				Name: fName, Label: fLabel, Enabled: false, MappingCode: mCode,
			})
		}
	}

	var finalMeta []BehaviorMetadata
	for _, k := range keys {
		finalMeta = append(finalMeta, *metaMap[k])
	}

	a.SaveMetadata(finalMeta)
	return fmt.Sprintf("全量同步成功！已更新 %d 类行为解析规则", len(finalMeta))
}

func (a *App) RunAnalysis(metadata []BehaviorMetadata) string {
	a.SaveMetadata(metadata)
	return "分析配置已同步"
}

// FetchRealReport gets actual counts and aggregations from ES using 13-digit timestamps.
// Optimized for 3 billion+ records using parallel chunking and sampling tech.
func (a *App) FetchRealReport(timeRange string, customStart, customEnd string) []AnalysisResult {
	if a.esClient == nil || a.cfg == nil {
		return []AnalysisResult{}
	}

	indexExpr := a.cfg.Elasticsearch.Index
	timeField := a.cfg.Elasticsearch.TimeField
	if timeField == "" {
		timeField = "@timestamp"
	}
	typeField := a.cfg.Elasticsearch.TypeField
	if typeField == "" {
		typeField = "type_code"
	}

	var startTime, endTime int64
	endTime = time.Now().UnixMilli()

	if timeRange == "custom" {
		ts, _ := time.ParseInLocation("2006-01-02 15:04:05", customStart, time.Local)
		te, _ := time.ParseInLocation("2006-01-02 15:04:05", customEnd, time.Local)
		startTime = ts.UnixMilli()
		if !te.IsZero() {
			endTime = te.UnixMilli()
		}
	} else {
		switch timeRange {
		case "last_7d":
			startTime = time.Now().AddDate(0, 0, -7).UnixMilli()
		case "last_30d":
			startTime = time.Now().AddDate(0, 0, -30).UnixMilli()
		case "last_year":
			startTime = time.Now().AddDate(-1, 0, 0).UnixMilli()
		default:
			startTime = time.Now().Add(-24 * time.Hour).UnixMilli()
		}
	}

	a.writeLog(fmt.Sprintf("🚀 [大数据模式] 发起分析: 范围=%d-%d, 索引=%s", startTime, endTime, indexExpr))

	// 系统检测 ES 版本以适配功能
	major, minor, vErr := a.esClient.GetVersion()
	// random_sampler 只有在 ES 8.2+ 才支持
	canUseRandomSampler := vErr == nil && (major > 8 || (major == 8 && minor >= 2))

	// 记录版本信息方便后续排查日志
	verStr := "未探测到"
	if vErr == nil {
		verStr = fmt.Sprintf("%d.%d", major, minor)
	}
	a.writeLog(fmt.Sprintf("🔍 ES 版本检测: [%s], 状态码: [%v], 启用采样: %t", verStr, vErr, canUseRandomSampler))

	meta := a.GetMetadata()
	selectedMeta := make([]BehaviorMetadata, 0)
	for _, m := range meta {
		if m.Selected {
			selectedMeta = append(selectedMeta, m)
		}
	}
	if len(selectedMeta) == 0 {
		return []AnalysisResult{}
	}

	duration := endTime - startTime
	const weekMs = 7 * 24 * 3600 * 1000

	// 如果不支持采样且数据量大，启用“重载分治模式”
	isHeavyLoad := !canUseRandomSampler && duration > weekMs

	// 1. 时间维度分段
	timeChunks := 1
	if duration > weekMs*4 {
		timeChunks = 12 // 一个月及以上拆分 12 段
	} else if duration > weekMs {
		timeChunks = 6
	} else if duration > 24*3600*1000 {
		timeChunks = 3
	}

	// 2. 行为维度分页 (解决 54 类行为导致的 search.max_buckets 超限及 ES 内存压力)
	// 54 类行为如果全部放入一个 Aggs 请求，在老版本 ES 上极易超出 10000 桶限制或 OOM
	const behaviorsPerBatch = 12
	var behaviorBatches [][]BehaviorMetadata
	for i := 0; i < len(selectedMeta); i += behaviorsPerBatch {
		end := i + behaviorsPerBatch
		if end > len(selectedMeta) {
			end = len(selectedMeta)
		}
		behaviorBatches = append(behaviorBatches, selectedMeta[i:end])
	}

	totalTasks := timeChunks * len(behaviorBatches)
	resultsChan := make(chan []AnalysisResult, totalTasks)
	var wg sync.WaitGroup

	// 全局超时设定：针对重载模式放宽到 300s
	globalTimeout := 60 * time.Second
	if isHeavyLoad {
		globalTimeout = 300 * time.Second
		a.writeLog("⚠️ [深度优化] 检测到 54 类复杂行为分析且版本不支持采样，已启用 [12段分时 × 5页分流] 加固策略")
	}

	ctxTotal, cancel := context.WithTimeout(context.Background(), globalTimeout)
	defer cancel()

	// 信号量：控制同时向 ES 发起的并发请求数，防止压垮老版本集群
	semaphore := make(chan struct{}, 3)

	a.writeLog(fmt.Sprintf("⚡ [分治调度] 总任务数: %d (时间分段: %d, 行为分页: %d)", totalTasks, timeChunks, len(behaviorBatches)))

	chunkSize := (endTime - startTime) / int64(timeChunks)
	for i := 0; i < timeChunks; i++ {
		cStart := startTime + int64(i)*chunkSize
		cEnd := cStart + chunkSize
		if i == timeChunks-1 {
			cEnd = endTime
		}

		for bIdx := range behaviorBatches {
			wg.Add(1)
			go func(s, e int64, batch []BehaviorMetadata) {
				defer wg.Done()

				select {
				case semaphore <- struct{}{}: // 获取令牌
					defer func() { <-semaphore }() // 释放令牌
					res := a.executeSingleChunk(ctxTotal, s, e, batch, indexExpr, timeField, typeField, canUseRandomSampler)
					resultsChan <- res
				case <-ctxTotal.Done():
					return
				}
			}(cStart, cEnd, behaviorBatches[bIdx])
		}
	}

	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	mergedMap := make(map[string]*AnalysisResult)
	for batch := range resultsChan {
		if batch == nil {
			continue
		}
		for _, bRes := range batch {
			if _, exists := mergedMap[bRes.TypeCode]; !exists {
				resCopy := bRes
				mergedMap[bRes.TypeCode] = &resCopy
			} else {
				target := mergedMap[bRes.TypeCode]
				target.Count += bRes.Count
				// 合并下钻维度
				for _, bGrp := range bRes.Groups {
					foundGrp := false
					for gi := range target.Groups {
						if target.Groups[gi].FieldLabel == bGrp.FieldLabel {
							foundGrp = true
							itemMap := make(map[string]*AnalysisItem)
							for ii := range target.Groups[gi].Items {
								itemMap[target.Groups[gi].Items[ii].RawKey] = &target.Groups[gi].Items[ii]
							}
							for _, bItem := range bGrp.Items {
								if tItem, ok := itemMap[bItem.RawKey]; ok {
									tItem.Count += bItem.Count
								} else {
									target.Groups[gi].Items = append(target.Groups[gi].Items, bItem)
								}
							}
							break
						}
					}
					if !foundGrp {
						target.Groups = append(target.Groups, bGrp)
					}
				}
			}
		}
	}

	final := make([]AnalysisResult, 0)
	for _, m := range selectedMeta {
		if res, ok := mergedMap[m.TypeCode]; ok {
			final = append(final, *res)
		}
	}

	a.writeLog(fmt.Sprintf("🏁 [大数据模式] 分析结束: 共处理 %d 条行为分类结果", len(final)))

	// 主动清理内存，归还操作系统，防范 30 亿数据后的应用端 OOM
	go func() {
		runtime.GC()
		debug.FreeOSMemory()
	}()

	return final
}

func (a *App) executeSingleChunk(ctx context.Context, s, e int64, meta []BehaviorMetadata, indexExpr, timeField, typeField string, useSampler bool) []AnalysisResult {
	agg := service.NewAggregator(a.esClient.GetClient())

	queryBody := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": []interface{}{
				map[string]interface{}{"range": map[string]interface{}{timeField: map[string]interface{}{"gte": s, "lte": e}}},
			},
		},
	}

	mainAggs := make(map[string]interface{})
	// 采样阈值调整：超过 3 天且支持采样时开启
	const threeDaysMs = 3 * 24 * 3600 * 1000
	isLargeRange := useSampler && (e-s) > threeDaysMs
	var samplingProbability float64 = 1.0

	var targetAggMap map[string]interface{} = mainAggs

	if isLargeRange {
		samplingProbability = 0.1
		samplerAgg := map[string]interface{}{
			"random_sampler": map[string]interface{}{"probability": samplingProbability},
			"aggs":           make(map[string]interface{}),
		}
		mainAggs["overspeed_sampler"] = samplerAgg
		targetAggMap = samplerAgg["aggs"].(map[string]interface{})
	}

	for _, m := range meta {
		behaviorFilterAgg := map[string]interface{}{
			"filter": map[string]interface{}{"term": map[string]interface{}{typeField: m.TypeCode}},
		}
		subAggs := make(map[string]interface{})
		for _, f := range m.Fields {
			if f.Enabled {
				safeAggName := "field_" + strings.ReplaceAll(f.Name, ".", "_")
				subAggs[safeAggName] = map[string]interface{}{
					"terms": map[string]interface{}{"field": f.Name, "size": 100},
				}
			}
		}
		if len(subAggs) > 0 {
			behaviorFilterAgg["aggs"] = subAggs
		}
		targetAggMap["behavior_"+m.TypeCode] = behaviorFilterAgg
	}

	raw, err := agg.ExecuteQueryAggregation(ctx, indexExpr, queryBody, mainAggs)
	if err != nil {
		a.writeLog(fmt.Sprintf("  ⚠️ Chunk [%d-%d] 请求异常: %v (是否采样: %v)", s, e, err, isLargeRange))
		return nil
	}

	aggregations, ok := raw["aggregations"].(map[string]interface{})
	if !ok {
		return nil
	}

	sourceAggs := aggregations
	if isLargeRange {
		if samplerBucket, ok := aggregations["overspeed_sampler"].(map[string]interface{}); ok {
			sourceAggs = samplerBucket
		}
	}

	var dict *service.DictEngine
	if a.adapter != nil && a.cfg.Database.DictTable != "" {
		dict = service.NewDictEngine(a.adapter, a.cfg.Database.DictTable, a.cfg.Database.DictCodeCol, a.cfg.Database.DictKeyCol, a.cfg.Database.DictValueCol)
	}

	results := make([]AnalysisResult, 0)
	for _, m := range meta {
		bucketKey := "behavior_" + m.TypeCode
		if bucket, ok := sourceAggs[bucketKey].(map[string]interface{}); ok {
			rawCount := 0.0
			if dc, ok := bucket["doc_count"].(float64); ok {
				rawCount = dc
			}
			estimatedCount := int(rawCount / samplingProbability)

			res := AnalysisResult{Type: m.Type, TypeCode: m.TypeCode, Count: estimatedCount, Groups: []AnalysisGroup{}}
			for _, f := range m.Fields {
				if !f.Enabled {
					continue
				}
				safeAggName := "field_" + strings.ReplaceAll(f.Name, ".", "_")
				if fieldAgg, ok := bucket[safeAggName].(map[string]interface{}); ok {
					if buckets, ok := fieldAgg["buckets"].([]interface{}); ok {
						group := AnalysisGroup{FieldLabel: f.Label, Items: []AnalysisItem{}}
						var dictCode string
						if dict != nil {
							dictCode, _, _ = dict.MatchField(f.Name, f.MappingCode)
						}

						for _, b := range buckets {
							itemB := b.(map[string]interface{})
							rawKey := fmt.Sprintf("%v", itemB["key"])
							itemDocCount := 0.0
							if idc, ok := itemB["doc_count"].(float64); ok {
								itemDocCount = idc
							}
							estItemCount := int(itemDocCount / samplingProbability)

							label := rawKey
							if dict != nil {
								label = dict.TranslateValue(dictCode, rawKey)
							}
							group.Items = append(group.Items, AnalysisItem{
								Label: label, RawKey: rawKey, Count: estItemCount,
							})
						}
						res.Groups = append(res.Groups, group)
					}
				}
			}
			results = append(results, res)
		}
	}
	return results
}

type AnalysisItem struct {
	Label  string `json:"label"`
	RawKey string `json:"raw_key"`
	Count  int    `json:"count"`
}

type AnalysisGroup struct {
	FieldLabel string         `json:"fieldLabel"`
	Items      []AnalysisItem `json:"items"`
}

type AnalysisResult struct {
	Type     string          `json:"type"`
	TypeCode string          `json:"type_code"`
	Count    int             `json:"count"`
	Groups   []AnalysisGroup `json:"groups"`
}

func (a *App) ExportAnalysisReport(resultsJson string, timeRangeStr string) string {
	var results []AnalysisResult
	json.Unmarshal([]byte(resultsJson), &results)

	savePath, err := wailsRuntime.SaveFileDialog(a.ctx, wailsRuntime.SaveDialogOptions{
		Title:           "导出行为分析结构化报表",
		DefaultFilename: fmt.Sprintf("行为审计报表_%s.xlsx", time.Now().Format("20060102_1504")),
		Filters:         []wailsRuntime.FileFilter{{DisplayName: "Excel 文件", Pattern: "*.xlsx"}},
	})
	if err != nil || savePath == "" {
		return ""
	}

	f := excelize.NewFile()
	defer f.Close()
	mainSheet := "行为审计报表"
	f.SetSheetName("Sheet1", mainSheet)

	style, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E0EBF5"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 11},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
		},
	})

	f.SetCellValue(mainSheet, "A1", "行为审计深度分析报告")
	f.MergeCell(mainSheet, "A1", "E1")
	f.SetCellValue(mainSheet, "A2", "审计周期："+timeRangeStr)
	f.MergeCell(mainSheet, "A2", "E2")

	headers := []string{"行为类型", "采集总数", "分析维度", "键值映射 (翻译后)", "出现频次"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 4)
		f.SetCellValue(mainSheet, cell, h)
		f.SetCellStyle(mainSheet, cell, cell, style)
	}

	currRow := 5
	for _, res := range results {
		startRow := currRow
		f.SetCellValue(mainSheet, fmt.Sprintf("A%d", currRow), res.Type)
		f.SetCellValue(mainSheet, fmt.Sprintf("B%d", currRow), res.Count)

		dimCount := 0
		for _, g := range res.Groups {
			for _, item := range g.Items {
				f.SetCellValue(mainSheet, fmt.Sprintf("C%d", currRow), g.FieldLabel)
				f.SetCellValue(mainSheet, fmt.Sprintf("D%d", currRow), item.Label)
				f.SetCellValue(mainSheet, fmt.Sprintf("E%d", currRow), item.Count)
				currRow++
				dimCount++
			}
		}

		if dimCount > 1 {
			f.MergeCell(mainSheet, fmt.Sprintf("A%d", startRow), fmt.Sprintf("A%d", currRow-1))
			f.MergeCell(mainSheet, fmt.Sprintf("B%d", startRow), fmt.Sprintf("B%d", currRow-1))
		} else if dimCount == 0 {
			currRow++
		}
	}

	f.SetColWidth(mainSheet, "A", "A", 25)
	f.SetColWidth(mainSheet, "C", "D", 20)

	f.SaveAs(savePath)
	return "报表已导出: " + savePath
}
