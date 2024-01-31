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
        this.status = responseData.status || 'queued';
        this.status_message = responseData.status_message || '';
        this.last_update = new Date(responseData.last_update || Date.now());
    }

    id: string;
    source_path: string;
    destination_path: string;
    status: string;
    status_message: string;
    last_update: Date;
}

interface JobUpdateNotification {
    id: string;
    status: string;
    message: string;
    event_time: Date;
    source_path: string;
    destination_path: string;
}

// it's needed to parse date
class JobUpdateNotificationClass implements JobUpdateNotification {
    constructor(responseData: Partial<JobUpdateNotification>) {
        this.id = responseData.id || '';
        this.status = responseData.status || '';
        this.message = responseData.message || '';
        this.event_time = new Date(responseData.event_time || Date.now());
        this.source_path = responseData.source_path || '';
        this.destination_path = responseData.destination_path || '';
    }

    id: string;
    status: string;
    message: string;
    event_time: Date;
    source_path: string;
    destination_path: string;
}

export type { Job, JobUpdateNotification };
export {JobClass, JobUpdateNotificationClass};
