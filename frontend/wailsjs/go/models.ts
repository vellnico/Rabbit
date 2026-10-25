export namespace main {
	
	export class AccountView {
	    id: string;
	    kind: string;
	    name: string;
	    uuid: string;
	    elySkin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AccountView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.kind = source["kind"];
	        this.name = source["name"];
	        this.uuid = source["uuid"];
	        this.elySkin = source["elySkin"];
	    }
	}
	export class State {
	    settings: store.Settings;
	    accounts: AccountView[];
	    instances: store.Instance[];
	    running: string[];
	    maxMemory: number;
	    dataDir: string;
	    elyBrowser: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new State(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.settings = this.convertValues(source["settings"], store.Settings);
	        this.accounts = this.convertValues(source["accounts"], AccountView);
	        this.instances = this.convertValues(source["instances"], store.Instance);
	        this.running = source["running"];
	        this.maxMemory = source["maxMemory"];
	        this.dataDir = source["dataDir"];
	        this.elyBrowser = source["elyBrowser"];
	        this.version = source["version"];
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
	export class UpdateInfo {
	    version: string;
	    current: string;
	    notes: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.current = source["current"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	    }
	}
	export class VersionInfo {
	    id: string;
	    type: string;
	    releaseTime: string;
	
	    static createFrom(source: any = {}) {
	        return new VersionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.type = source["type"];
	        this.releaseTime = source["releaseTime"];
	    }
	}

}

export namespace modrinth {
	
	export class Hit {
	    project_id: string;
	    slug: string;
	    project_type: string;
	    title: string;
	    description: string;
	    author: string;
	    icon_url: string;
	    downloads: number;
	    follows: number;
	    display_categories: string[];
	    versions: string[];
	
	    static createFrom(source: any = {}) {
	        return new Hit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project_id = source["project_id"];
	        this.slug = source["slug"];
	        this.project_type = source["project_type"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.author = source["author"];
	        this.icon_url = source["icon_url"];
	        this.downloads = source["downloads"];
	        this.follows = source["follows"];
	        this.display_categories = source["display_categories"];
	        this.versions = source["versions"];
	    }
	}
	export class Item {
	    path: string;
	    file: string;
	    type: string;
	    enabled: boolean;
	    projectId: string;
	    title: string;
	    iconUrl: string;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.file = source["file"];
	        this.type = source["type"];
	        this.enabled = source["enabled"];
	        this.projectId = source["projectId"];
	        this.title = source["title"];
	        this.iconUrl = source["iconUrl"];
	        this.version = source["version"];
	    }
	}
	export class SearchParams {
	    query: string;
	    type: string;
	    gameVersion: string;
	    loader: string;
	    sort: string;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.type = source["type"];
	        this.gameVersion = source["gameVersion"];
	        this.loader = source["loader"];
	        this.sort = source["sort"];
	        this.offset = source["offset"];
	    }
	}
	export class SearchResult {
	    hits: Hit[];
	    offset: number;
	    limit: number;
	    total_hits: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hits = this.convertValues(source["hits"], Hit);
	        this.offset = source["offset"];
	        this.limit = source["limit"];
	        this.total_hits = source["total_hits"];
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

export namespace store {
	
	export class Instance {
	    id: string;
	    name: string;
	    gameVersion: string;
	    loader: string;
	    loaderVersion?: string;
	    // Go type: time
	    created: any;
	    // Go type: time
	    lastPlayed?: any;
	
	    static createFrom(source: any = {}) {
	        return new Instance(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.gameVersion = source["gameVersion"];
	        this.loader = source["loader"];
	        this.loaderVersion = source["loaderVersion"];
	        this.created = this.convertValues(source["created"], null);
	        this.lastPlayed = this.convertValues(source["lastPlayed"], null);
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
	export class Settings {
	    memoryMb: number;
	    selectedAccount: string;
	    selectedInstance: string;
	    closeOnLaunch: boolean;
	    jvmArgs: string;
	    accent: string;
	    language: string;
	    checkUpdates: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.memoryMb = source["memoryMb"];
	        this.selectedAccount = source["selectedAccount"];
	        this.selectedInstance = source["selectedInstance"];
	        this.closeOnLaunch = source["closeOnLaunch"];
	        this.jvmArgs = source["jvmArgs"];
	        this.accent = source["accent"];
	        this.language = source["language"];
	        this.checkUpdates = source["checkUpdates"];
	    }
	}

}

