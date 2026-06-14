export namespace dto {
	
	export class ConnectionInput {
	    connectionName: string;
	    host: string;
	    remotePort: number;
	    localPort: number;
	    domain: string;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.connectionName = source["connectionName"];
	        this.host = source["host"];
	        this.remotePort = source["remotePort"];
	        this.localPort = source["localPort"];
	        this.domain = source["domain"];
	    }
	}

}

export namespace model {
	
	export class Forward {
	    id: number;
	    hostId: number;
	    label: string;
	    localPort: number;
	    remoteHost: string;
	    remotePort: number;
	    enabled: boolean;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Forward(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.hostId = source["hostId"];
	        this.label = source["label"];
	        this.localPort = source["localPort"];
	        this.remoteHost = source["remoteHost"];
	        this.remotePort = source["remotePort"];
	        this.enabled = source["enabled"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class Host {
	    id: number;
	    name: string;
	    hostname: string;
	    user: string;
	    port: number;
	    authMethod: string;
	    keyPath: string;
	    hostKeyPolicy: string;
	    forwards: Forward[];
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new Host(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.hostname = source["hostname"];
	        this.user = source["user"];
	        this.port = source["port"];
	        this.authMethod = source["authMethod"];
	        this.keyPath = source["keyPath"];
	        this.hostKeyPolicy = source["hostKeyPolicy"];
	        this.forwards = this.convertValues(source["forwards"], Forward);
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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

