export namespace core {
	
	export class Activity {
	    id: string;
	    task_id: string;
	    activity: string;
	    comment: string;
	    // Go type: time
	    created_at: any;
	
	    static createFrom(source: any = {}) {
	        return new Activity(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.task_id = source["task_id"];
	        this.activity = source["activity"];
	        this.comment = source["comment"];
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
	export class ActivityFilter {
	    TaskIDs: string[];
	    Limit: number;
	    IncludeArchived: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ActivityFilter(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.TaskIDs = source["TaskIDs"];
	        this.Limit = source["Limit"];
	        this.IncludeArchived = source["IncludeArchived"];
	    }
	}
	export class ActivityInput {
	    Activity: string;
	    Comment: string;
	
	    static createFrom(source: any = {}) {
	        return new ActivityInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Activity = source["Activity"];
	        this.Comment = source["Comment"];
	    }
	}
	export class CreateInput {
	    Title: string;
	    Description: string;
	    Feedback: string;
	    Status: string;
	    Progress: number;
	    ParentID: string;
	
	    static createFrom(source: any = {}) {
	        return new CreateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Description = source["Description"];
	        this.Feedback = source["Feedback"];
	        this.Status = source["Status"];
	        this.Progress = source["Progress"];
	        this.ParentID = source["ParentID"];
	    }
	}
	export class ListFilter {
	    Status: string;
	    Search: string;
	    Limit: number;
	    Sort: string;
	    Ascending: boolean;
	    IncludeDone: boolean;
	    Archived: boolean;
	    RootsOnly: boolean;
	
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
	        this.Archived = source["Archived"];
	        this.RootsOnly = source["RootsOnly"];
	    }
	}
	export class Task {
	    id: string;
	    title: string;
	    description: string;
	    feedback: string;
	    status: string;
	    progress: number;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    // Go type: time
	    completed_at?: any;
	    // Go type: time
	    archived_at?: any;
	    parent_id?: string;
	    board_rank?: number;
	
	    static createFrom(source: any = {}) {
	        return new Task(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.description = source["description"];
	        this.feedback = source["feedback"];
	        this.status = source["status"];
	        this.progress = source["progress"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.completed_at = this.convertValues(source["completed_at"], null);
	        this.archived_at = this.convertValues(source["archived_at"], null);
	        this.parent_id = source["parent_id"];
	        this.board_rank = source["board_rank"];
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
	    Feedback?: string;
	    Progress?: number;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Desc = source["Desc"];
	        this.Feedback = source["Feedback"];
	        this.Progress = source["Progress"];
	    }
	}

}

