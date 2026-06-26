/**
 * Search Bar Component for Marketplace
 * Handles search input with debouncing support
 */

import React, { useCallback as _useCallback, useState, useEffect } from 'react';
import { Search } from 'lucide-react';

interface SearchBarProps {
  value: string;
  onChange: (value: string) => void;
  debounceDelay?: number;
}

const SearchBar: React.FC<SearchBarProps> = ({
  value,
  onChange,
  debounceDelay = 300
}) => {
  const [inputValue, setInputValue] = useState(value);

  useEffect(() => {
    const timer = setTimeout(() => {
      onChange(inputValue);
    }, debounceDelay);

    return () => clearTimeout(timer);
  }, [inputValue, onChange, debounceDelay]);

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
  };

  return (
    <div className="relative">
      <Search className="absolute left-4 top-1/2 transform -translate-y-1/2 text-slate-400 w-5 h-5" />
      <input
        type="text"
        placeholder="Search components..."
        value={inputValue}
        onChange={handleChange}
        className="w-full bg-slate-700 text-white pl-12 pr-4 py-3 rounded-lg border border-slate-600 focus:border-blue-500 focus:outline-none transition"
        aria-label="Search components"
      />
    </div>
  );
};

export default SearchBar;
