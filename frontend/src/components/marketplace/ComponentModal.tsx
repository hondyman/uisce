/**
 * Component Modal Component for Marketplace
 * Displays detailed component information in a modal
 */

import type { FC } from 'react';
import { devDebug } from '../../utils/devLogger';
import {
  Download,
  Star,
  Package,
  Eye,
  Heart,
  Share2,
  CheckCircle,
  X
} from 'lucide-react';
import { Component } from '../../data/marketplaceComponents';

interface ComponentModalProps {
  component: Component | null;
  isInstalled: boolean;
  onClose: () => void;
  onInstall: (componentId: string) => void;
  onUninstall: (componentId: string) => void;
}

const ComponentModal: FC<ComponentModalProps> = ({
  component,
  isInstalled,
  onClose,
  onInstall,
  onUninstall
}) => {
  if (!component) return null;

  const handlePreview = () => {
    if (component.preview) {
      window.open(component.preview, '_blank');
    }
  };

  const handleShare = async () => {
    const shareText = `Check out ${component.name} on the Component Marketplace! ${component.description}`;
    if (navigator.share) {
      try {
        await navigator.share({
          title: component.name,
          text: shareText,
          url: window.location.href
        });
      } catch (err) {
        devDebug('Share cancelled');
      }
    } else {
      // Fallback: copy to clipboard
      navigator.clipboard.writeText(shareText);
    }
  };

  return (
    <div
      className="fixed inset-0 bg-black/80 flex items-center justify-center z-50 p-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div className="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <div className="p-8">
          {/* Header */}
          <div className="flex items-start justify-between mb-6">
            <div className="flex items-center gap-4">
              <div className="text-6xl">{component.icon}</div>
              <div>
                <h2 id="modal-title" className="text-3xl font-bold">
                  {component.name}
                </h2>
                <p className="text-slate-400">
                  by {component.author} • v{component.version}
                </p>
              </div>
            </div>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-white text-2xl"
              aria-label="Close modal"
            >
              <X className="w-6 h-6" />
            </button>
          </div>

          <p className="text-lg text-slate-300 mb-6">{component.description}</p>

          {/* Stats Grid */}
          <div className="grid grid-cols-4 gap-4 mb-6">
            <div className="bg-slate-700 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Star className="w-5 h-5 text-yellow-400 fill-current" />
                <span className="text-2xl font-bold">{component.rating}</span>
              </div>
              <div className="text-sm text-slate-400">{component.reviews} reviews</div>
            </div>
            <div className="bg-slate-700 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Download className="w-5 h-5 text-blue-400" />
                <span className="text-2xl font-bold">
                  {(component.downloads / 1000).toFixed(1)}k
                </span>
              </div>
              <div className="text-sm text-slate-400">Downloads</div>
            </div>
            <div className="bg-slate-700 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <Package className="w-5 h-5 text-purple-400" />
                <span className="text-2xl font-bold">{component.version}</span>
              </div>
              <div className="text-sm text-slate-400">Version</div>
            </div>
            <div className="bg-slate-700 rounded-lg p-4">
              <div className="flex items-center gap-2 mb-2">
                <span className="text-2xl font-bold text-blue-400">{component.price}</span>
              </div>
              <div className="text-sm text-slate-400">Price</div>
            </div>
          </div>

          {/* Details Section */}
          <div className="space-y-6">
            {/* Tags */}
            <div>
              <h3 className="font-bold text-lg mb-3">Tags</h3>
              <div className="flex flex-wrap gap-2">
                {component.tags.map((tag, idx) => (
                  <span key={idx} className="px-3 py-1 bg-slate-700 rounded-full text-sm">
                    {tag}
                  </span>
                ))}
              </div>
            </div>

            {/* Dependencies */}
            <div>
              <h3 className="font-bold text-lg mb-3">Dependencies</h3>
              <div className="bg-slate-900 rounded-lg p-4">
                <code className="text-sm text-green-400">
                  {component.dependencies.join(', ')}
                </code>
              </div>
            </div>

            {/* Configuration */}
            <div>
              <h3 className="font-bold text-lg mb-3">Configuration</h3>
              <div className="bg-slate-900 rounded-lg p-4 overflow-x-auto">
                <pre className="text-sm text-blue-400">
                  {JSON.stringify(component.config, null, 2)}
                </pre>
              </div>
            </div>

            {/* Installation */}
            <div>
              <h3 className="font-bold text-lg mb-3">Installation</h3>
              <div className="bg-slate-900 rounded-lg p-4 mb-3">
                <code className="text-sm text-green-400">
                  npm install @portfolio-dashboard/{component.id}
                </code>
              </div>
              <div className="bg-slate-900 rounded-lg p-4">
                <pre className="text-sm text-blue-400">
                  {`// Import in your dashboard
import { ${component.name.replace(/\s+/g, '')} } from '@portfolio-dashboard/${component.id}';

// Add to your component registry
componentRegistry.register('${component.config.type}', ${component.name.replace(/\s+/g, '')});`}
                </pre>
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex gap-3 mt-8 flex-wrap">
            {isInstalled ? (
              <>
                <button className="flex-1 flex items-center justify-center gap-2 px-6 py-3 bg-green-600 rounded-lg min-w-fit">
                  <CheckCircle className="w-5 h-5" />
                  Installed
                </button>
                <button
                  onClick={() => onUninstall(component.id)}
                  className="px-6 py-3 bg-red-600 hover:bg-red-700 rounded-lg transition"
                >
                  Uninstall
                </button>
              </>
            ) : (
              <button
                onClick={() => onInstall(component.id)}
                className="flex-1 flex items-center justify-center gap-2 px-6 py-3 bg-blue-600 hover:bg-blue-700 rounded-lg transition min-w-fit"
              >
                <Download className="w-5 h-5" />
                Install Component
              </button>
            )}
            <button
              onClick={handlePreview}
              disabled={!component.preview}
              className="flex items-center justify-center gap-2 px-6 py-3 bg-slate-700 hover:bg-slate-600 rounded-lg transition disabled:opacity-50 disabled:cursor-not-allowed"
              title={component.preview ? 'View preview' : 'No preview available'}
            >
              <Eye className="w-5 h-5" />
              Preview
            </button>
            <button
              onClick={handleShare}
              className="flex items-center justify-center gap-2 px-6 py-3 bg-slate-700 hover:bg-slate-600 rounded-lg transition"
              title="Share component"
            >
              <Share2 className="w-5 h-5" />
            </button>
            <button
              className="flex items-center justify-center gap-2 px-6 py-3 bg-slate-700 hover:bg-slate-600 rounded-lg transition"
              title="Add to favorites"
            >
              <Heart className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ComponentModal;
