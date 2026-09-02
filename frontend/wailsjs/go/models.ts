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
	    Cwd: string;
	    HumanOnly: boolean;
	    SlackThread: string;
	
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
	        this.Cwd = source["Cwd"];
	        this.HumanOnly = source["HumanOnly"];
	        this.SlackThread = source["SlackThread"];
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
	    IncludeHumanOnly: boolean;
	
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
	        this.IncludeHumanOnly = source["IncludeHumanOnly"];
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
	    cwd: string;
	    human_only: boolean;
	    include_in_report: boolean;
	    slack_thread: string;
	
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
	        this.cwd = source["cwd"];
	        this.human_only = source["human_only"];
	        this.include_in_report = source["include_in_report"];
	        this.slack_thread = source["slack_thread"];
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
	    Cwd?: string;
	    HumanOnly?: boolean;
	    IncludeInReport?: boolean;
	    SlackThread?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Title = source["Title"];
	        this.Desc = source["Desc"];
	        this.Feedback = source["Feedback"];
	        this.Progress = source["Progress"];
	        this.Cwd = source["Cwd"];
	        this.HumanOnly = source["HumanOnly"];
	        this.IncludeInReport = source["IncludeInReport"];
	        this.SlackThread = source["SlackThread"];
	    }
	}

}

export namespace integrations {
	
	export class HerdrTaskStatus {
	    ready: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new HerdrTaskStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.error = source["error"];
	    }
	}

}

export namespace settings {
	
	export class ClaudeConfig {
	    enabled: boolean;
	    binary: string;
	    env_start: string;
	    ticket_prompt: string;
	    close_tab_on_done: boolean;
	    require_cwd: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ClaudeConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.binary = source["binary"];
	        this.env_start = source["env_start"];
	        this.ticket_prompt = source["ticket_prompt"];
	        this.close_tab_on_done = source["close_tab_on_done"];
	        this.require_cwd = source["require_cwd"];
	    }
	}
	export class HerdrConfig {
	    enabled: boolean;
	    binary: string;
	    env_start: string;
	    space_name: string;
	
	    static createFrom(source: any = {}) {
	        return new HerdrConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.binary = source["binary"];
	        this.env_start = source["env_start"];
	        this.space_name = source["space_name"];
	    }
	}
	export class GUISettings {
	    default_cwd: string;
	    default_human_only: boolean;
	    archive_done_subtasks: boolean;
	    start_hidden: boolean;
	    claude: ClaudeConfig;
	    herdr: HerdrConfig;
	
	    static createFrom(source: any = {}) {
	        return new GUISettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.default_cwd = source["default_cwd"];
	        this.default_human_only = source["default_human_only"];
	        this.archive_done_subtasks = source["archive_done_subtasks"];
	        this.start_hidden = source["start_hidden"];
	        this.claude = this.convertValues(source["claude"], ClaudeConfig);
	        this.herdr = this.convertValues(source["herdr"], HerdrConfig);
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

