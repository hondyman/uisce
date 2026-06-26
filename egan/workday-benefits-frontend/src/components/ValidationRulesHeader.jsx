import React from 'react';

const ValidationRulesHeader = () => {
  return (
    <div className="flex flex-wrap items-center justify-between gap-4 mb-6">
      <div className="flex flex-col gap-1">
        <p className="text-gray-900 dark:text-white text-3xl font-bold leading-tight">Validation Rules Management</p>
        <p className="text-gray-500 dark:text-gray-400 text-base font-normal leading-normal">Configure, view, and edit custom validation rules for benefit elections.</p>
      </div>
      <button className="flex items-center justify-center gap-2 overflow-hidden rounded-lg h-10 px-4 bg-primary text-white text-sm font-bold leading-normal tracking-wide hover:bg-primary/90">
        <span className="material-symbols-outlined">add</span>
        <span className="truncate">New Rule</span>
      </button>
    </div>
  );
};

export default ValidationRulesHeader;