export namespace core {
	
	export class CreateInput {
	    Title: string;
	    Description: string;
	    Status: string;
	    Progress: number;
	
	    static createFrom(source: any = {}) {
	        return new CreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Description = source["Description"];
	        this.Status = source["Status"];
	        this.Progress = source["Progress"];
	    }
	}
	export class ListFilter {
	    Status: string;
	    Search: string;
	    Limit: number;
	    Sort: string;
	    Ascending: boolean;
	    IncludeDone: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ListFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Status = source["Status"];
	        this.Search = source["Search"];
	        this.Limit = source["Limit"];
	        this.Sort = source["Sort"];
	        this.Ascending = source["Ascending"];
	        this.IncludeDone = source["IncludeDone"];
	    }
	}
	export class Task {
	    id: string;
	    title: string;
	    description: string;
	    status: string;
	    progress: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    // Go type: time
	    completed_at?: any;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
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
	export class UpdateInput {
	    Title?: string;
	    Desc?: string;
	    Progress?: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Desc = source["Desc"];
	        this.Progress = source["Progress"];
	    }
	}

}

