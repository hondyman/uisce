/**
 * Component Card Component for Marketplace
 * Displays individual component card in the grid
 */

import type { FC, MouseEvent } from 'react';
import { Download, Star, CheckCircle } from 'lucide-react';
import { Component } from '../../data/marketplaceComponents';

interface ComponentCardProps {
  component: Component;
  isInstalled: boolean;
  onInstall: (componentId: string) => void;
  onUninstall: (componentId: string) => void;
  onSelect: (component: Component) => void;
}

const ComponentCard: FC<ComponentCardProps> = ({
  component,
  isInstalled,
  onInstall,
  onUninstall,
  onSelect
}) => {
  const handleInstallClick = (e: MouseEvent) => {
    e.stopPropagation();
    onInstall(component.id);
  };

  const handleUninstallClick = (e: MouseEvent) => {
    e.stopPropagation();
    onUninstall(component.id);
  };

  return (
    <div
      className="bg-slate-800 rounded-xl border border-slate-700 p-6 hover:border-slate-600 transition cursor-pointer"
      onClick={() => onSelect(component)}
      role="article"
      aria-label={`${component.name} component card`}
    >
      <div className="flex items-start gap-4">
        <div className="text-5xl flex-shrink-0">{component.icon}</div>

        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between mb-2 gap-2">
            <div className="min-w-0">
              <h3 className="font-bold text-lg truncate">{component.name}</h3>
              <p className="text-sm text-slate-400">
                by {component.author} • v{component.version}
              </p>
            </div>
            <div className="flex items-center gap-2 flex-shrink-0">
              <div className="flex items-center gap-1 text-yellow-400">
                <Star className="w-4 h-4 fill-current" />
                <span className="text-sm font-bold">{component.rating}</span>
                <span className="text-xs text-slate-400">({component.reviews})</span>
              </div>
            </div>
          </div>

          <p className="text-slate-300 mb-3 line-clamp-2">{component.description}</p>

          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-4 text-sm min-w-0">
              <div className="flex items-center gap-1 text-slate-400 flex-shrink-0">
                <Download className="w-4 h-4" />
                {component.downloads.toLocaleString()}
              </div>
              <div className="flex gap-1 flex-wrap">
                {component.tags.slice(0, 2).map((tag, idx) => (
                  <span key={idx} className="px-2 py-1 bg-slate-700 rounded text-xs whitespace-nowrap">
                    {tag}
                  </span>
                ))}
                {component.tags.length > 2 && (
                  <span className="px-2 py-1 bg-slate-700 rounded text-xs text-slate-400">
                    +{component.tags.length - 2}
                  </span>
                )}
              </div>
            </div>

            <div className="flex items-center gap-2 flex-shrink-0">
              <span className="font-bold text-blue-400">{component.price}</span>
              {isInstalled ? (
                <button
                  onClick={handleUninstallClick}
                  className="flex items-center gap-2 px-4 py-2 bg-green-600 hover:bg-green-700 rounded-lg transition"
                  title="Uninstall component"
                >
                  <CheckCircle className="w-4 h-4" />
                  Installed
                </button>
              ) : (
                <button
                  onClick={handleInstallClick}
                  className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 rounded-lg transition"
                  title="Install component"
                >
                  <Download className="w-4 h-4" />
                  Install
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ComponentCard;
