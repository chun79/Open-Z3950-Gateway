export namespace main {
	
	export class BookDetail {
	    title: string;
	    author: string;
	    isbn: string;
	    publisher: string;
	    edition: string;
	    summary: string;
	    toc: string;
	    physical: string;
	    series: string;
	    notes: string;
	    fields: z3950.MARCField[];
	    holdings: z3950.Holding[];
	
	    static createFrom(source: any = {}) {
	        return new BookDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.author = source["author"];
	        this.isbn = source["isbn"];
	        this.publisher = source["publisher"];
	        this.edition = source["edition"];
	        this.summary = source["summary"];
	        this.toc = source["toc"];
	        this.physical = source["physical"];
	        this.series = source["series"];
	        this.notes = source["notes"];
	        this.fields = this.convertValues(source["fields"], z3950.MARCField);
	        this.holdings = this.convertValues(source["holdings"], z3950.Holding);
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
	export class ILLRequest {
	    id: number;
	    title: string;
	    author: string;
	    isbn: string;
	    target_db: string;
	    status: string;
	    comments: string;
	    requestor: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new ILLRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.isbn = source["isbn"];
	        this.target_db = source["target_db"];
	        this.status = source["status"];
	        this.comments = source["comments"];
	        this.requestor = source["requestor"];
	        this.created_at = this.convertValues(source["created_at"], null);
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
	export class SavedBook {
	    id: number;
	    title: string;
	    author: string;
	    isbn: string;
	    source_db: string;
	    notes: string;
	    // Go type: time
	    saved_at: any;
	
	    static createFrom(source: any = {}) {
	        return new SavedBook(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.isbn = source["isbn"];
	        this.source_db = source["source_db"];
	        this.notes = source["notes"];
	        this.saved_at = this.convertValues(source["saved_at"], null);
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
	export class SearchParams {
	    dbs: string[];
	    term: string;
	    attr: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dbs = source["dbs"];
	        this.term = source["term"];
	        this.attr = source["attr"];
	    }
	}
	export class SearchResult {
	    source_db: string;
	    title: string;
	    author: string;
	    isbn: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.source_db = source["source_db"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.isbn = source["isbn"];
	    }
	}
	export class Target {
	    name: string;
	    host: string;
	    port: number;
	    db: string;
	    encoding: string;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.db = source["db"];
	        this.encoding = source["encoding"];
	    }
	}
	export class TestResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}

}

export namespace z3950 {
	
	export class Holding {
	    call_number: string;
	    status: string;
	    location: string;
	
	    static createFrom(source: any = {}) {
	        return new Holding(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.call_number = source["call_number"];
	        this.status = source["status"];
	        this.location = source["location"];
	    }
	}
	export class MARCField {
	    Tag: string;
	    Value: string;
	
	    static createFrom(source: any = {}) {
	        return new MARCField(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Tag = source["Tag"];
	        this.Value = source["Value"];
	    }
	}

}

