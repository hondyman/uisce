/**
 * Featured Components Section
 * Displays a curated selection of featured components
 */

import type { FC } from 'react';
import { TrendingUp, Star } from 'lucide-react';
import { Component } from '../../data/marketplaceComponents';

interface FeaturedComponentsProps {
  components: Component[];
  onSelectComponent: (component: Component) => void;
}

const FeaturedComponents: FC<FeaturedComponentsProps> = ({
  components,
  onSelectComponent
}) => {
  return (
    <div className="mb-8">
      <h2 className="text-2xl font-bold mb-4 flex items-center gap-2">
        <TrendingUp className="w-6 h-6 text-yellow-400" />
        Featured Components
      </h2>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {components.slice(0, 2).map((comp) => (
          <div
            key={comp.id}
            className="bg-gradient-to-br from-blue-900/30 to-purple-900/30 rounded-xl border border-blue-500/30 p-6 cursor-pointer hover:border-blue-400/50 transition"
            onClick={() => onSelectComponent(comp)}
            role="article"
            aria-label={`Featured component: ${comp.name}`}
          >
            <div className="flex items-start justify-between mb-3">
              <div className="text-4xl">{comp.icon}</div>
              <div className="flex items-center gap-1 text-yellow-400">
                <Star className="w-4 h-4 fill-current" />
                <span className="text-sm font-bold">{comp.rating}</span>
              </div>
            </div>
            <h3 className="font-bold text-lg mb-2">{comp.name}</h3>
            <p className="text-sm text-slate-300 mb-3 line-clamp-2">{comp.description}</p>
            <div className="flex items-center justify-between">
              <div className="text-xs text-slate-400">{comp.downloads.toLocaleString()} downloads</div>
              <div className="text-sm font-bold text-blue-400">{comp.price}</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default FeaturedComponents;
