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

export const renderPath = (isSmallScreen: boolean, path: string) => {
    if (isSmallScreen) {
        const shortPath = path.split('/').pop();
        return shortPath ? shortPath : path;
    } else {
        return path;
    }
};
