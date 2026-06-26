/**
 * Category Filter Component for Marketplace
 * Displays category options with counts
 */

import type { FC } from 'react';
import { Filter } from 'lucide-react';
import { Category } from '../../data/marketplaceComponents';

interface CategoryFilterProps {
  categories: Category[];
  selectedCategory: string;
  onSelectCategory: (categoryId: string) => void;
}

const CategoryFilter: FC<CategoryFilterProps> = ({
  categories,
  selectedCategory,
  onSelectCategory
}) => {
  return (
    <div className="bg-slate-800 rounded-xl border border-slate-700 p-4">
      <h3 className="font-bold mb-3 flex items-center gap-2">
        <Filter className="w-4 h-4 text-purple-400" />
        Categories
      </h3>
      <div className="space-y-2">
        {categories.map((cat) => (
          <button
            key={cat.id}
            onClick={() => onSelectCategory(cat.id)}
            className={`w-full text-left px-3 py-2 rounded-lg transition ${
              selectedCategory === cat.id
                ? 'bg-blue-600 text-white'
                : 'bg-slate-700 hover:bg-slate-600 text-slate-300'
            }`}
            data-selected={selectedCategory === cat.id}
          >
            <div className="flex items-center justify-between">
              <span className="text-sm">{cat.label}</span>
              <span className="text-xs opacity-70">{cat.count}</span>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
};

export default CategoryFilter;
