export const operatorOptions = [
    { value: 'equals', label: 'Equals' },
    { value: 'not_equals', label: 'Not Equals' },
    { value: 'contains', label: 'Contains' },
    { value: 'not_contains', label: 'Not Contains' },
    { value: 'starts_with', label: 'Starts With' },
    { value: 'ends_with', label: 'Ends With' },
    { value: 'in', label: 'In' },
    { value: 'not_in', label: 'Not In' },
    { value: 'greater_than', label: 'Greater Than' },
    { value: 'less_than', label: 'Less Than' },
    { value: 'greater_than_or_equal', label: 'Greater Than or Equal' },
    { value: 'less_than_or_equal', label: 'Less Than or Equal' },
];

export const maskTypeOptions = [
    { value: 'redact', label: 'Redact' },
    { value: 'null', label: 'Null' },
    { value: 'hash', label: 'Hash' },
];

export const generateId = () => {
    if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
        return crypto.randomUUID();
    }
    return `policy-${Math.random().toString(36).slice(2, 10)}`;
};