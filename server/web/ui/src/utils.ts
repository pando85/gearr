import { Job } from './model';

export const STATUS_FILTER_OPTIONS = [
    'progressing',
    'queued',
    'completed',
    'failed',
];

export const DATE_FILTER_OPTIONS = [
    'Last update',
    'Last 30 minutes',
    'Last 3 hours',
    'Last 6 hours',
    'Last 24 hours',
    'Last 2 days',
    'Last 7 days',
    'Last 30 days',
];

const formatDate = (date: Date, options: Intl.DateTimeFormatOptions): string => {
    if (date == null) {
        return '';
    }

    try {
        return new Intl.DateTimeFormat(navigator.language, options).format(date);
    } catch (error) {
        console.error('Error formatting date:', error);
        return '';
    }
};

export const formatDateDetailed = (date: Date): string => {
    const options: Intl.DateTimeFormatOptions = {
        timeStyle: 'long',
    };
    return formatDate(date, options);
};

export const formatDateShort = (date: Date): string => {
    const options: Intl.DateTimeFormatOptions = {
        dateStyle: 'short',
    };
    const formatedDate = formatDate(date, options)
    return formatedDate;
};

export const getDateFromFilterOption = (filterOption: string) => {
    const currentDate = new Date();

    switch (filterOption) {
        case 'Last 30 minutes':
            return new Date(currentDate.getTime() - 30 * 60 * 1000);

        case 'Last 3 hours':
            return new Date(currentDate.getTime() - 3 * 60 * 60 * 1000);

        case 'Last 6 hours':
            return new Date(currentDate.getTime() - 6 * 60 * 60 * 1000);

        case 'Last 24 hours':
            return new Date(currentDate.getTime() - 24 * 60 * 60 * 1000);

        case 'Last 2 days':
            return new Date(currentDate.getTime() - 2 * 24 * 60 * 60 * 1000);

        case 'Last 7 days':
            return new Date(currentDate.getTime() - 7 * 24 * 60 * 60 * 1000);

        case 'Last 30 days':
            return new Date(currentDate.getTime() - 30 * 24 * 60 * 60 * 1000);

        default:
            return new Date(0);
    }
}

export const getStatusColor = (status: string): string => {
    switch (status) {
        case 'completed':
            return 'green';
        case 'failed':
            return 'red';
        default:
            return 'grey';
    }
};

export const renderPath = (path: string, maxLength: number) => {
    const fileName = path.split('/').pop() || '';
    if (path.length > maxLength) {
        return fileName.substring(0, maxLength) + '...';
    } else {
        return fileName;
    }
};

export const renderPathSmallScreen = (path: string, isSmallScreen: boolean, maxLength: number) => {
    if (isSmallScreen) {
        return renderPath(path, maxLength);
    } else {
        return path;
    }
};

export const sortJobs = (sortColumn: string | null, sortDirection: 'asc' | 'desc', jobs: Job[]) => {
    if (!sortColumn) return jobs;

    const sortedJobs = [...jobs].sort((a, b) => {
        const valueA: any = a[sortColumn as keyof Job];
        const valueB: any = b[sortColumn as keyof Job];

        if (typeof valueA === 'string' && typeof valueB === 'string') {
            return sortDirection === 'asc' ? valueA.localeCompare(valueB) : valueB.localeCompare(valueA);
        } else {
            return sortDirection === 'asc' ? valueA - valueB : valueB - valueA;
        }
    });

    return sortedJobs;
};
