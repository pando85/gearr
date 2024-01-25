interface Job {
    id: string;
    source_path: string;
    destination_path: string;
    status: string;
    status_message: string;
    last_update: Date;
}

class JobClass implements Job {
    constructor(responseData: Partial<Job>) {
        this.id = responseData.id || '';
        this.source_path = responseData.source_path || '';
        this.destination_path = responseData.destination_path || '';
        this.status = responseData.status || '';
        this.status_message = responseData.status_message || '';
        this.last_update = new Date(responseData.last_update || ''); // Parse date or default to empty string
    }

    id: string;
    source_path: string;
    destination_path: string;
    status: string;
    status_message: string;
    last_update: Date;
}

export type { Job };
export {JobClass};
