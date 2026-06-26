import React, { useState } from 'react';
import { Link2, ExternalLink } from 'lucide-react';
import type { EntityPath, ValidationRule as _ValidationRule } from '../types';

const ENTITY_RELATIONSHIPS: Record<string, Array<{ field: string; targetEntity: string; relationship: string }>> = {
  Employee: [
    { field: 'department_id', targetEntity: 'Department', relationship: 'many-to-one' },
    { field: 'manager_id', targetEntity: 'Employee', relationship: 'many-to-one' },
    { field: 'position_id', targetEntity: 'Position', relationship: 'many-to-one' },
    { field: 'location_id', targetEntity: 'Location', relationship: 'many-to-one' }
  ],
  Department: [
    { field: 'location_id', targetEntity: 'Location', relationship: 'many-to-one' },
    { field: 'cost_center_id', targetEntity: 'Cost Center', relationship: 'many-to-one' },
    { field: 'parent_department_id', targetEntity: 'Department', relationship: 'many-to-one' }
  ],
  Position: [
    { field: 'department_id', targetEntity: 'Department', relationship: 'many-to-one' },
    { field: 'job_family_id', targetEntity: 'Job Family', relationship: 'many-to-one' }
  ],
  Location: [
    { field: 'country_id', targetEntity: 'Country', relationship: 'many-to-one' }
  ]
};

const ENTITY_FIELDS: Record<string, Array<{ name: string; type: string; label: string }>> = {
  Employee: [
    { name: 'employee_id', type: 'string', label: 'Employee ID' },
    { name: 'first_name', type: 'string', label: 'First Name' },
    { name: 'last_name', type: 'string', label: 'Last Name' },
    { name: 'salary', type: 'number', label: 'Salary' },
    { name: 'hire_date', type: 'date', label: 'Hire Date' },
    { name: 'status', type: 'string', label: 'Status' },
    { name: 'age', type: 'number', label: 'Age' }
  ],
  Department: [
    { name: 'department_name', type: 'string', label: 'Department Name' },
    { name: 'budget', type: 'number', label: 'Budget' },
    { name: 'head_count', type: 'number', label: 'Head Count' }
  ],
  Position: [
    { name: 'position_title', type: 'string', label: 'Position Title' },
    { name: 'min_salary', type: 'number', label: 'Minimum Salary' },
    { name: 'max_salary', type: 'number', label: 'Maximum Salary' },
    { name: 'job_level', type: 'number', label: 'Job Level' }
  ],
  Location: [
    { name: 'location_name', type: 'string', label: 'Location Name' },
    { name: 'city', type: 'string', label: 'City' },
    { name: 'country', type: 'string', label: 'Country' }
  ]
};

const EntityPathPicker: React.FC<{
  startEntity: string;
  value: EntityPath | null;
  onChange: (path: EntityPath) => void;
  label: string;
}> = ({ startEntity, value, onChange, label }) => {
  const [currentEntity, setCurrentEntity] = useState(startEntity);
  const [pathSegments, setPathSegments] = useState<EntityPath['segments']>(value?.segments || []);
  const [isOpen, setIsOpen] = useState(false);

  const relationships = ENTITY_RELATIONSHIPS[currentEntity] || [];
  const fields = ENTITY_FIELDS[currentEntity] || [];

  const addSegment = (field: string, targetEntity: string, relationship: string) => {
    const newSegment = { entity: currentEntity, field, relationship };
    const newSegments = [...pathSegments, newSegment];
    setPathSegments(newSegments);
    setCurrentEntity(targetEntity);
  };

  const selectField = (fieldName: string) => {
    const displayPath = [...pathSegments.map(s => s.entity), currentEntity].join(' → ') + '.' + fieldName;
    onChange({ segments: pathSegments, displayPath });
    setIsOpen(false);
  };

  const reset = () => { setPathSegments([]); setCurrentEntity(startEntity); };

  const currentPath = [...pathSegments.map(s => s.entity), currentEntity].join(' → ');

  return (
    <div className="space-y-2">
      <label className="block text-sm font-semibold text-gray-700">{label}</label>
      <div onClick={() => setIsOpen(!isOpen)} className="w-full px-4 py-3 border-2 border-gray-300 rounded-lg cursor-pointer hover:border-blue-400 bg-white">
        {value ? (
          <div className="flex items-center justify-between">
            <span className="font-mono text-sm text-blue-600">{value.displayPath}</span>
            <ExternalLink size={16} className="text-gray-400" />
          </div>
        ) : (
          <div className="text-gray-400 text-sm">Click to select a field path...</div>
        )}
      </div>

      {isOpen && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-lg shadow-2xl w-full max-w-3xl max-h-[80vh] flex flex-col">
            <div className="bg-gradient-to-r from-purple-600 to-purple-700 text-white px-6 py-4 rounded-t-lg">
              <h3 className="text-xl font-semibold">Select Field Path</h3>
              <p className="text-purple-100 text-sm mt-1">Navigate through related entities to select a field</p>
            </div>

            <div className="p-6 flex-1 overflow-y-auto">
              <div className="bg-purple-50 border border-purple-200 rounded-lg p-3 mb-4">
                <div className="text-xs font-semibold text-purple-700 mb-1">CURRENT PATH</div>
                <div className="font-mono text-sm text-purple-900">{currentPath}</div>
                {pathSegments.length > 0 && (
                  <button onClick={reset} className="mt-2 text-xs text-purple-600 hover:text-purple-700 underline">Reset to {startEntity}</button>
                )}
              </div>

              {relationships.length > 0 && (
                <div className="mb-6">
                  <h4 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
                    <Link2 size={18} className="text-purple-600" />
                    Related Entities
                  </h4>
                  <div className="grid grid-cols-2 gap-3">
                    {relationships.map((rel) => (
                      <button key={rel.field} onClick={() => addSegment(rel.field, rel.targetEntity, rel.relationship)} className="text-left p-3 border-2 border-purple-200 rounded-lg hover:border-purple-400 hover:bg-purple-50 transition-all">
                        <div className="font-semibold text-gray-900">{rel.targetEntity}</div>
                        <div className="text-xs text-gray-600 mt-1">via {rel.field}</div>
                        <div className="text-xs text-purple-600 mt-1">{rel.relationship}</div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              <div>
                <h4 className="font-semibold text-gray-900 mb-3">Select Field from {currentEntity}</h4>
                <div className="grid grid-cols-2 gap-2">
                  {fields.map((field) => (
                    <button key={field.name} onClick={() => selectField(field.name)} className="text-left p-3 border border-gray-300 rounded-lg hover:border-blue-400 hover:bg-blue-50 transition-all">
                      <div className="font-semibold text-gray-900 text-sm">{field.label}</div>
                      <div className="text-xs text-gray-500 mt-1">{field.name}</div>
                      <span className={`inline-block px-2 py-0.5 text-xs rounded mt-2 ${field.type === 'string' ? 'bg-blue-100 text-blue-700' : field.type === 'number' ? 'bg-green-100 text-green-700' : 'bg-purple-100 text-purple-700'}`}>{field.type}</span>
                    </button>
                  ))}
                </div>
              </div>
            </div>

            <div className="bg-gray-50 px-6 py-4 border-t rounded-b-lg flex justify-end">
              <button onClick={() => setIsOpen(false)} className="px-6 py-2 bg-gray-200 text-gray-700 rounded-lg hover:bg-gray-300">Cancel</button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default EntityPathPicker;
