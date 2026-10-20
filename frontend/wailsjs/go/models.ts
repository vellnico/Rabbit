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
	    }
	}

}

