import React, { useState, useEffect } from 'react';
import './App.css';
import { GetESInfo, GetConfig, SaveConfig, SelectFile, GetMetadata, RunAnalysis, TestESConnection, TestDBConnection, DownloadMetadataTemplate, UploadMetadata, ExportAnalysisReport, FetchRealReport } from "../wailsjs/go/main/App";

function App() {
    const [view, setView] = useState('metadata');
    const [esStatus, setEsStatus] = useState("✨ 正在检测节点...");
    const [config, setConfig] = useState(null);
    const [metadata, setMetadata] = useState([]);
    const [editingMetaIdx, setEditingMetaIdx] = useState(null);
    const [saveStatus, setSaveStatus] = useState("");
    const [analysisResult, setAnalysisResult] = useState([]);
    const [dbTestStatus, setDbTestStatus] = useState("未连接");
    const [esTestStatus, setEsTestStatus] = useState("未连接");

    const [timeRange, setTimeRange] = useState("last_24h");
    const [isAnalyzing, setIsAnalyzing] = useState(false);
    const [showCustomRange, setShowCustomRange] = useState(false);
    const [customRange, setCustomRange] = useState({ start: '', end: '' });

    // UI Tree Table State
    const [expandedRows, setExpandedRows] = useState(new Set());
    const [sortConfig, setSortConfig] = useState({ key: 'count', direction: 'desc' });

    const dbTypes = [
        { id: 'mariadb', name: 'MariaDB' }, { id: 'mysql', name: 'MySQL' },
        { id: 'pg', name: 'PostgreSQL' }, { id: 'kingbase', name: '人大金仓 (Kingbase)' },
        { id: 'dm', name: '达梦 (DM)' }, { id: 'highgo', name: '瀚高 (Highgo)' },
        { id: 'vastbase', name: 'Vastbase' }, { id: 'shentong', name: '神通 (Shentong)' }, { id: 'xugu', name: '虚谷 (Xugu)' },
        { id: 'odbc', name: 'ODBC (国产数据库通用)' }
    ];

    useEffect(() => {
        refreshStatus();
        GetConfig().then(setConfig);
        loadMetadata();
    }, []);

    const refreshStatus = () => { GetESInfo().then(setEsStatus); };
    const loadMetadata = () => { GetMetadata().then(setMetadata); };

    const handleSaveConfig = () => {
        setSaveStatus("⏳ 正在保存...");
        const finalCfg = {
            ...config,
            elasticsearch: { ...config.elasticsearch, port: parseInt(config.elasticsearch.port) || 0 },
            database: { ...config.database, port: parseInt(config.database.port) || 0 }
        };
        SaveConfig(finalCfg).then(res => {
            setSaveStatus(res);
            setTimeout(() => setSaveStatus(""), 3000);
            refreshStatus();
        });
    };

    const handleTestES = () => {
        setEsTestStatus("正在连接...");
        TestESConnection(config.elasticsearch.ip, parseInt(config.elasticsearch.port), config.elasticsearch.user || '', config.elasticsearch.password || '')
            .then(res => {
                if (res === "测试连接成功") setEsTestStatus("连接成功");
                else setEsTestStatus("连接失败");
            });
    };

    const handleTestDB = () => {
        setDbTestStatus("正在连接...");
        TestDBConnection(
            config.database.type,
            config.database.host,
            parseInt(config.database.port),
            config.database.dbname,
            config.database.schema || '',
            config.database.user || '',
            config.database.password || '',
            config.database.driver_path || '',
            config.database.conn_url || '',
            config.database.driver_class || ''
        ).then(res => {
            if (res === "测试成功") {
                setDbTestStatus("连接成功");
            } else {
                setDbTestStatus(res); // 显示具体报错信息
                console.error("DB Test Error:", res);
            }
        });
    };

    const updateConfig = (section, field, value) => {
        setConfig(prev => ({ ...prev, [section]: { ...prev[section], [field]: value } }));
    };

    const toggleFieldInModal = (fIdx) => {
        setMetadata(prev => {
            const next = [...prev];
            next[editingMetaIdx].fields[fIdx].enabled = !next[editingMetaIdx].fields[fIdx].enabled;
            return next;
        });
    };

    const handleRunAnalysis = (targetTimeRange = null) => {
        setIsAnalyzing(true);
        setAnalysisResult([]); // 清空旧数据，提升用户体验
        setExpandedRows(new Set()); // 重置展开状态

        const currentRange = typeof targetTimeRange === 'string' ? targetTimeRange : timeRange;
        const start = customRange.start ? customRange.start.replace('T', ' ') + (customRange.start.includes(':') && customRange.start.length === 16 ? ':00' : '') : '';
        const end = customRange.end ? customRange.end.replace('T', ' ') + (customRange.end.includes(':') && customRange.end.length === 16 ? ':00' : '') : '';

        RunAnalysis(metadata).then(() => {
            FetchRealReport(currentRange, start, end).then(res => {
                setAnalysisResult(res || []);
                setView('dashboard');
            }).catch(err => {
                console.error("FetchRealReport Error:", err);
            }).finally(() => {
                setIsAnalyzing(false);
            });
        }).catch(err => {
            console.error("RunAnalysis Error:", err);
            setIsAnalyzing(false);
        });
    };

    const toggleExpand = (id) => {
        const next = new Set(expandedRows);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        setExpandedRows(next);
    };

    const handleSort = (key) => {
        let direction = 'desc';
        if (sortConfig.key === key && sortConfig.direction === 'desc') direction = 'asc';
        setSortConfig({ key, direction });
    };

    const sortedResults = [...analysisResult].sort((a, b) => {
        if (sortConfig.key === 'count') {
            return sortConfig.direction === 'desc' ? b.count - a.count : a.count - b.count;
        }
        return 0;
    });

    const renderHeader = (title, actions = []) => (
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px', gap: '12px' }}>
            <h2 style={{ fontSize: '1.25rem' }}>{title}</h2>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>{actions}</div>
        </div>
    );

    const renderDashboard = () => (
        <div className="content-container">
            <section style={{ position: 'relative' }}>
                {isAnalyzing && <div className="scanning-line"></div>}
                {renderHeader("行为分析看板", [
                    <select className="form-select" style={{ width: '130px' }} value={timeRange} onChange={e => {
                        const val = e.target.value;
                        setTimeRange(val);
                        if (val !== 'custom') handleRunAnalysis(val);
                        else setShowCustomRange(true);
                    }}>
                        <option value="last_24h">近 24 小时</option>
                        <option value="last_7d">近 7 天</option>
                        <option value="last_30d">近 30 天</option>
                        <option value="last_year">近 1 年</option>
                        <option value="custom">📅 自定义范围</option>
                    </select>,
                    timeRange === 'custom' && (
                        <button key="edit-time" className="btn-pill btn-outline" style={{ width: '42px', padding: '0', fontSize: '1rem' }} onClick={() => setShowCustomRange(true)} title="编辑时间范围">
                            ✏️
                        </button>
                    ),
                    <button className="btn-pill btn-outline" onClick={() => {
                        const timeStr = timeRange === 'custom' ? `${customRange.start}~${customRange.end}` : timeRange;
                        ExportAnalysisReport(JSON.stringify(analysisResult), timeStr).then(res => alert(res));
                    }}>导出报表</button>,
                    <button className="btn-pill btn-primary" onClick={handleRunAnalysis}>🔄 刷新分析</button>
                ])}

                {showCustomRange && (
                    <div className="custom-date-popover">
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', marginBottom: '16px' }}>
                            <div className="form-group"><label>开始日期与时间</label><input type="datetime-local" className="form-input" value={customRange.start} onChange={e => setCustomRange({ ...customRange, start: e.target.value })} /></div>
                            <div className="form-group"><label>结束日期与时间</label><input type="datetime-local" className="form-input" value={customRange.end} onChange={e => setCustomRange({ ...customRange, end: e.target.value })} /></div>
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button className="btn-pill btn-outline" onClick={() => setShowCustomRange(false)}>取消</button>
                            <button className="btn-pill btn-primary" onClick={() => { setShowCustomRange(false); handleRunAnalysis(); }}>执行</button>
                        </div>
                    </div>
                )}

                <div className="audit-table-container">
                    <table className="audit-table">
                        <thead>
                            <tr>
                                <th style={{ width: '40px' }}></th>
                                <th>行为类型 / 下钻指标</th>
                                <th onClick={() => handleSort('count')} style={{ width: '160px' }}>
                                    总次数 {sortConfig.key === 'count' && (sortConfig.direction === 'desc' ? '↓' : '↑')}
                                </th>
                                <th style={{ width: '300px' }}>数据分布占比</th>
                            </tr>
                        </thead>
                        <tbody>
                            {sortedResults.length === 0 && (
                                <tr>
                                    <td colSpan="4" style={{ textAlign: 'center', padding: '80px', color: 'var(--text-dim)' }}>
                                        {isAnalyzing ? (
                                            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '16px' }}>
                                                <div className="spinner"></div>
                                                <div style={{ fontWeight: 800, color: 'var(--accent-color)' }}>深度分析中，请稍候...</div>
                                                <div style={{ fontSize: '0.8rem', opacity: 0.6 }}>正在对 30 亿级行为索引执行大规模并行计算</div>
                                            </div>
                                        ) : "暂无分析结果，请检查连接设置或确认索引内是否有数据。"}
                                    </td>
                                </tr>
                            )}
                            {sortedResults.map((res, idx) => {
                                const hasGroups = res.groups && res.groups.length > 0;
                                return (
                                    <React.Fragment key={`b-${idx}`}>
                                        <tr className="row-behavior" style={{ cursor: hasGroups ? 'pointer' : 'default' }} onClick={() => hasGroups && toggleExpand(res.type_code)}>
                                            <td>
                                                {hasGroups ? (
                                                    <span className="expand-btn" style={{ transform: expandedRows.has(res.type_code) ? 'rotate(90deg)' : 'none' }}>▶</span>
                                                ) : (
                                                    <span style={{ display: 'inline-block', width: '26px' }}></span>
                                                )}
                                            </td>
                                            <td style={{ fontWeight: 800, fontSize: '1.05rem', color: 'var(--primary-color)' }}>
                                                {res.type}
                                                <span style={{ opacity: 0.4, fontWeight: 400, fontSize: '0.8rem', marginLeft: '8px', fontFamily: 'monospace' }}>[{res.type_code}]</span>
                                            </td>
                                            <td style={{ fontSize: '1.1rem', fontWeight: 800 }}>{res.count.toLocaleString()}</td>
                                            <td></td>
                                        </tr>
                                        {expandedRows.has(res.type_code) && res.groups.map((group, gidx) => (
                                            <React.Fragment key={`g-${idx}-${gidx}`}>
                                                <tr className="row-dimension" style={{ backgroundColor: 'rgba(52, 152, 219, 0.05)' }}>
                                                    <td></td>
                                                    <td colSpan="3" style={{ paddingLeft: '20px', fontWeight: 700, color: '#2c3e50', fontSize: '0.9rem' }}>
                                                        🎯 分析维度：{group.fieldLabel}
                                                    </td>
                                                </tr>
                                                {group.items.map((item, iidx) => {
                                                    const percent = res.count > 0 ? (item.count / res.count * 100).toFixed(1) : 0;
                                                    return (
                                                        <tr key={`i-${idx}-${gidx}-${iidx}`} className="row-value" style={{ borderBottom: '1px solid #f0f0f0' }}>
                                                            <td></td>
                                                            <td style={{ paddingLeft: '40px' }}>
                                                                <div style={{ display: 'flex', flexDirection: 'column' }}>
                                                                    <span style={{ fontWeight: 700, color: '#333', fontSize: '0.95rem' }}>{item.label}</span>
                                                                    <span style={{ fontSize: '0.75rem', color: '#999', fontFamily: 'monospace' }}>原始代码: {item.raw_key || '-'}</span>
                                                                </div>
                                                            </td>
                                                            <td style={{ fontWeight: 600, color: '#2c3e50' }}>{item.count.toLocaleString()} <span style={{ fontSize: '0.7rem', color: '#999' }}>次</span></td>
                                                            <td style={{ paddingRight: '20px' }}>
                                                                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                                                                    <div className="progress-container" style={{ flex: 1, height: '8px', background: '#ecf0f1', borderRadius: '4px', overflow: 'hidden' }}>
                                                                        <div className="progress-fill" style={{
                                                                            width: `${percent}%`,
                                                                            height: '100%',
                                                                            background: `linear-gradient(90deg, #3498db, #2980b9)`,
                                                                            borderRadius: '4px',
                                                                            transition: 'width 0.6s ease'
                                                                        }}></div>
                                                                    </div>
                                                                    <span style={{ fontSize: '0.8rem', width: '45px', fontWeight: 800, color: '#34495e', textAlign: 'left' }}>{percent}%</span>
                                                                </div>
                                                            </td>
                                                        </tr>
                                                    );
                                                })}
                                            </React.Fragment>
                                        ))}
                                    </React.Fragment>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            </section>
        </div>
    );

    const renderMetadata = () => (
        <div className="content-container">
            <section>
                {renderHeader("审计行为解析规则", [
                    <button className="btn-pill btn-outline" onClick={() => DownloadMetadataTemplate().then(res => alert(res))}>下载导入模板</button>,
                    <button className="btn-pill btn-outline" onClick={() => UploadMetadata().then(res => { alert(res); loadMetadata(); })}>上传配置文件</button>,
                    <button className="btn-pill btn-primary" onClick={handleRunAnalysis}>生效并分析</button>
                ])}
                <div className="table-wrapper">
                    <table className="metadata-table">
                        <thead>
                            <tr>
                                <th>业务行为名称 (点击配置下钻)</th>
                                <th>已开启的下钻维度</th>
                                <th style={{ textAlign: 'center' }}>是否分析</th>
                            </tr>
                        </thead>
                        <tbody>
                            {metadata.map((m, idx) => (
                                <tr key={idx} style={{ opacity: m.selected ? 1 : 0.4 }}>
                                    <td onClick={() => setEditingMetaIdx(idx)} style={{ cursor: 'pointer', fontWeight: 800, color: 'var(--accent-color)' }}>{m.type}</td>
                                    <td>
                                        <div style={{ display: 'flex', gap: '4px', flexWrap: 'wrap' }}>
                                            {m.fields?.filter(f => f.enabled).map(f => <span key={f.name} className="tag-chip active">{f.label || f.Label}</span>)}
                                            {m.fields?.filter(f => f.enabled).length === 0 && <span className="tag-chip">仅统计总量</span>}
                                        </div>
                                    </td>
                                    <td style={{ textAlign: 'center' }}>
                                        <input type="checkbox" style={{ width: '18px', height: '18px' }} checked={m.selected} onChange={() => {
                                            const nm = [...metadata];
                                            nm[idx].selected = !nm[idx].selected;
                                            setMetadata(nm);
                                        }} />
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            </section>

            {editingMetaIdx !== null && (
                <div className="modal-overlay" onClick={() => setEditingMetaIdx(null)}>
                    <div className="modal-content" onClick={e => e.stopPropagation()}>
                        <h3 style={{ margin: '0 0 20px 0' }}>配置下钻维度</h3>
                        <div style={{ fontSize: '0.9rem', color: 'var(--text-dim)', marginBottom: '20px' }}>当前审计项：{metadata[editingMetaIdx].type}</div>
                        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '8px' }}>
                            {metadata[editingMetaIdx].fields?.map((f, fidx) => (
                                <div key={fidx} className={`tag-chip ${f.enabled ? 'active' : ''}`} style={{ padding: '8px 14px', cursor: 'pointer' }} onClick={() => toggleFieldInModal(fidx)}>
                                    {f.label || f.Label}
                                </div>
                            ))}
                        </div>
                        <button className="btn-pill btn-primary" style={{ marginTop: '30px', width: '100%' }} onClick={() => setEditingMetaIdx(null)}>保存设置</button>
                    </div>
                </div>
            )}
        </div>
    );

    const renderSettings = () => (
        <div className="content-container">
            <section>
                {renderHeader("系统连接设置", [
                    <span style={{ fontSize: '0.8rem', color: 'var(--accent-color)', fontWeight: 800 }}>{saveStatus}</span>,
                    <button key="sv" className="btn-pill btn-primary" onClick={handleSaveConfig}>💾 确认保存配置</button>
                ])}
                <div className="settings-grid">
                    <div className="settings-card">
                        <div className="settings-card-header">
                            <div style={{ fontWeight: 800 }}>Elasticsearch 集群设置</div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <div className="status-badge">
                                    <div className={`dot ${esTestStatus === '连接成功' ? 'online' : (esTestStatus === '连接失败' ? 'offline' : '')}`}></div>
                                    {esTestStatus}
                                </div>
                                <button className="btn-pill btn-outline" style={{ padding: '4px 12px', fontSize: '0.75rem' }} onClick={handleTestES}>连通性测试</button>
                            </div>
                        </div>
                        <div className="form-grid">
                            <div className="form-group"><label>IP 地址</label><input className="form-input" value={config?.elasticsearch?.ip || ''} onChange={e => updateConfig('elasticsearch', 'ip', e.target.value)} /></div>
                            <div className="form-group"><label>端口</label><input className="form-input" value={config?.elasticsearch?.port || ''} onChange={e => updateConfig('elasticsearch', 'port', e.target.value)} /></div>
                            <div className="form-group" style={{ gridColumn: 'span 2' }}><label>索引名称</label><input className="form-input" value={config?.elasticsearch?.index || ''} onChange={e => updateConfig('elasticsearch', 'index', e.target.value)} /></div>
                            <div className="form-group"><label>时间字段名称</label><input className="form-input" value={config?.elasticsearch?.time_field || ''} onChange={e => updateConfig('elasticsearch', 'time_field', e.target.value)} /></div>
                            <div className="form-group"><label>行为类型字段名</label><input className="form-input" placeholder="例如: source_app_type" value={config?.elasticsearch?.type_field || ''} onChange={e => updateConfig('elasticsearch', 'type_field', e.target.value)} /></div>
                            <div className="form-group"><label>账号</label><input className="form-input" value={config?.elasticsearch?.user || ''} onChange={e => updateConfig('elasticsearch', 'user', e.target.value)} /></div>
                            <div className="form-group"><label>密码</label><input className="form-input" type="password" value={config?.elasticsearch?.password || ''} onChange={e => updateConfig('elasticsearch', 'password', e.target.value)} /></div>
                        </div>
                    </div>

                    <div className="settings-card">
                        <div className="settings-card-header">
                            <div style={{ fontWeight: 800 }}>数据库字典配置 (用于值翻译)</div>
                            <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <div className="status-badge">
                                    <div className={`dot ${dbTestStatus === '连接成功' ? 'online' : (dbTestStatus === '连接失败' ? 'offline' : '')}`}></div>
                                    {dbTestStatus}
                                </div>
                                <button className="btn-pill btn-outline" style={{ padding: '4px 12px', fontSize: '0.75rem' }} onClick={handleTestDB}>连通性测试</button>
                            </div>
                        </div>
                        <div className="form-grid">
                            <div className="form-group"><label>数据库类型</label>
                                <select className="form-select" value={config?.database?.type || ''} onChange={e => updateConfig('database', 'type', e.target.value)}>
                                    {dbTypes.map(t => <option key={t.id} value={t.id}>{t.name}</option>)}
                                </select>
                            </div>
                            <div className="form-group"><label>数据库 IP / 主机</label><input className="form-input" value={config?.database?.host || ''} onChange={e => updateConfig('database', 'host', e.target.value)} /></div>
                            <div className="form-group"><label>端口</label><input className="form-input" value={config?.database?.port || ''} onChange={e => updateConfig('database', 'port', e.target.value)} /></div>
                            <div className="form-group"><label>实例 / 数据库名</label><input className="form-input" value={config?.database?.dbname || ''} onChange={e => updateConfig('database', 'dbname', e.target.value)} /></div>
                            <div className="form-group"><label>模式名 (Schema)</label><input className="form-input" value={config?.database?.schema || ''} onChange={e => updateConfig('database', 'schema', e.target.value)} /></div>
                            <div className="form-group" style={{ gridColumn: 'span 2' }}>
                                <label style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                                    独立连接 URL (Go 原生 DSN 格式)
                                    <span title="【注意】本系统由 Go 语言开发，切勿填写 jdbc: 格式！

金仓/PG: postgres://user:pass@127.0.0.1:54321/dbname?sslmode=disable&search_path=模式名
达梦 (原生): dm://user:pass@127.0.0.1:5236
达梦 (ODBC): driver={DM8 ODBC DRIVER};server=127.0.0.1:5236;uid=SYSDBA;pwd=pass
虚谷: xugu://user:pass@127.0.0.1:5138/dbname
海量: postgres://user:pass@127.0.0.1:5432/dbname" style={{ cursor: 'help', color: 'red', fontSize: '0.9rem', fontWeight: 'bold' }}>ⓘ (必看)</span>
                                </label>
                                <input className="form-input"
                                    placeholder="Go DSN，例如: postgres://root:123@192.168.1.100:54321/db?sslmode=disable"
                                    value={config?.database?.conn_url || ''}
                                    onChange={e => updateConfig('database', 'conn_url', e.target.value)}
                                />
                            </div>
                            <div className="form-group"><label>账号</label><input className="form-input" value={config?.database?.user || ''} onChange={e => updateConfig('database', 'user', e.target.value)} /></div>
                            <div className="form-group"><label>密码</label><input className="form-input" type="password" value={config?.database?.password || ''} onChange={e => updateConfig('database', 'password', e.target.value)} /></div>
                        </div>

                        <div style={{ background: '#f8fafc', padding: '16px', borderRadius: '10px', border: '1px solid #edf2f7', marginTop: '10px' }}>
                            <div style={{ fontWeight: 800, fontSize: '0.8rem', color: 'var(--accent-color)', marginBottom: '12px' }}>数据字典映射配置</div>
                            <div className="form-grid">
                                <div className="form-group"><label>字典表名称</label><input className="form-input" value={config?.database?.dict_table || ''} onChange={e => updateConfig('database', 'dict_table', e.target.value)} /></div>
                                <div className="form-group"><label>业务类型列 (dic_code)</label><input className="form-input" value={config?.database?.dict_code_col || ''} onChange={e => updateConfig('database', 'dict_code_col', e.target.value)} /></div>
                                <div className="form-group"><label>原始代码列 (dict_key)</label><input className="form-input" value={config?.database?.dict_key_col || ''} onChange={e => updateConfig('database', 'dict_key_col', e.target.value)} /></div>
                                <div className="form-group"><label>中文翻译列 (dict_value)</label><input className="form-input" value={config?.database?.dict_value_col || ''} onChange={e => updateConfig('database', 'dict_value_col', e.target.value)} /></div>
                            </div>
                        </div>
                    </div>
                </div>
            </section>
        </div>
    );

    return (
        <div id="App">
            <aside>
                <div className="brand">ES-SPECTRE <span style={{ opacity: 0.4, fontWeight: 300 }}>V5.05</span></div>
                <div className="nav-menu">
                    <div className={`nav-item ${view === 'dashboard' ? 'active' : ''}`} onClick={() => setView('dashboard')}>📊 项目分析看板</div>
                    <div className={`nav-item ${view === 'metadata' ? 'active' : ''}`} onClick={() => setView('metadata')}>⚙️ 解析规则配置</div>
                    <div className={`nav-item ${view === 'settings' ? 'active' : ''}`} onClick={() => setView('settings')}>🔌 系统连接设置</div>
                </div>
                <div style={{ padding: '24px', position: 'absolute', bottom: 0, width: 'var(--sidebar-width)', borderTop: '1px solid #f1f5f9' }}>
                    <div style={{ fontSize: '0.7rem', color: 'var(--text-dim)', marginBottom: '8px' }}>节点状态</div>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.8rem', fontWeight: 900, color: esStatus.includes('Connected') || esStatus.includes('已连接') ? '#10b981' : '#ef4444' }}>
                        <div className={`dot ${esStatus.includes('Connected') || esStatus.includes('已连接') ? 'online' : 'offline'}`}></div>
                        {esStatus.includes('Connected') || esStatus.includes('已连接') ? '运行中' : '未连接'}
                    </div>
                </div>
            </aside>
            <main>
                {view === 'dashboard' ? renderDashboard() : (view === 'metadata' ? renderMetadata() : renderSettings())}
            </main>
        </div>
    );
}

export default App;
