
import { 
    IconNumbers, 
    IconCalendar, 
    IconToggleLeft, 
    IconTextSize, 
    IconHash,
    IconAlertTriangle,
    IconStack3,
    IconChartBar,
    IconFilter,
    IconCheck
} from './icons';
import { ColumnMapping } from './types';

export const isNumericType = (type: string) => {
    return type.includes('int') || type.includes('float') || type.includes('decimal') || type.includes('numeric');
};

export const getDataTypeIcon = (type: string) => {
    if (type.includes('int') || type.includes('serial')) return <IconNumbers size={14} className="text-blue-500" />;
    if (type.includes('float') || type.includes('decimal') || type.includes('numeric')) return <IconNumbers size={14} className="text-green-500" />;
    if (type.includes('date') || type.includes('timestamp')) return <IconCalendar size={14} className="text-purple-500" />;
    if (type.includes('bool')) return <IconToggleLeft size={14} className="text-orange-500" />;
    if (type.includes('char') || type.includes('text')) return <IconTextSize size={14} className="text-gray-500" />;
    return <IconHash size={14} className="text-gray-400" />;
};

export const getMappingColor = (mappingType: ColumnMapping['mappingType'] | null): string => {
    if (!mappingType || mappingType === 'none') return '#ef4444'; // Red for not mapped or 'none'

    const colors = {
      dimension: '#3b82f6', // blue-500
      measure: '#10b981',   // emerald-500
      filter: '#f97316',    // orange-500
    };
    // The type of mappingType is narrowed by the check above, so this is safe.
    return colors[mappingType] || '#e5e7eb'; // default gray
};

export const getMappingIcon = (mapping: ColumnMapping | null) => {
    if (!mapping || mapping.mappingType === 'none') return <IconAlertTriangle size={14} />;
    if (mapping.mappingType === 'dimension') return <IconStack3 size={14} />;
    if (mapping.mappingType === 'measure') return <IconChartBar size={14} />;
    if (mapping.mappingType === 'filter') return <IconFilter size={14} />;
    return <IconCheck size={14} />;
};