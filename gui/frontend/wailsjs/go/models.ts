export namespace config {
	
	export class AppConfig {
	    debug: boolean;
	    export_path: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.debug = source["debug"];
	        this.export_path = source["export_path"];
	    }
	}
	export class DBConfig {
	    type: string;
	    host: string;
	    port: number;
	    user: string;
	    password: string;
	    dbname: string;
	    schema: string;
	    conn_url: string;
	    driver_path: string;
	    driver_class: string;
	    dict_table: string;
	    dict_code_col: string;
	    dict_key_col: string;
	    dict_value_col: string;
	
	    static createFrom(source: any = {}) {
	        return new DBConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.dbname = source["dbname"];
	        this.schema = source["schema"];
	        this.conn_url = source["conn_url"];
	        this.driver_path = source["driver_path"];
	        this.driver_class = source["driver_class"];
	        this.dict_table = source["dict_table"];
	        this.dict_code_col = source["dict_code_col"];
	        this.dict_key_col = source["dict_key_col"];
	        this.dict_value_col = source["dict_value_col"];
	    }
	}
	export class ESConfig {
	    ip: string;
	    port: number;
	    index: string;
	    user: string;
	    password: string;
	    time_field: string;
	    type_field: string;
	
	    static createFrom(source: any = {}) {
	        return new ESConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.port = source["port"];
	        this.index = source["index"];
	        this.user = source["user"];
	        this.password = source["password"];
	        this.time_field = source["time_field"];
	        this.type_field = source["type_field"];
	    }
	}
	export class Config {
	    elasticsearch: ESConfig;
	    database: DBConfig;
	    app: AppConfig;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.elasticsearch = this.convertValues(source["elasticsearch"], ESConfig);
	        this.database = this.convertValues(source["database"], DBConfig);
	        this.app = this.convertValues(source["app"], AppConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	

}

export namespace main {
	
	export class AnalysisItem {
	    label: string;
	    raw_key: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.label = source["label"];
	        this.raw_key = source["raw_key"];
	        this.count = source["count"];
	    }
	}
	export class AnalysisGroup {
	    fieldLabel: string;
	    items: AnalysisItem[];
	
	    static createFrom(source: any = {}) {
	        return new AnalysisGroup(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fieldLabel = source["fieldLabel"];
	        this.items = this.convertValues(source["items"], AnalysisItem);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class AnalysisResult {
	    type: string;
	    type_code: string;
	    count: number;
	    groups: AnalysisGroup[];
	
	    static createFrom(source: any = {}) {
	        return new AnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.type = source["type"];
	        this.type_code = source["type_code"];
	        this.count = source["count"];
	        this.groups = this.convertValues(source["groups"], AnalysisGroup);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class MetadataField {
	    name: string;
	    label: string;
	    enabled: boolean;
	    mapping_code: string;
	
	    static createFrom(source: any = {}) {
	        return new MetadataField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.label = source["label"];
	        this.enabled = source["enabled"];
	        this.mapping_code = source["mapping_code"];
	    }
	}
	export class BehaviorMetadata {
	    standard: string;
	    type: string;
	    type_code: string;
	    fields: MetadataField[];
	    selected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new BehaviorMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.standard = source["standard"];
	        this.type = source["type"];
	        this.type_code = source["type_code"];
	        this.fields = this.convertValues(source["fields"], MetadataField);
	        this.selected = source["selected"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

