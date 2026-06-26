/**
 * RuleTemplateMarketplace.tsx
 * 
 * Shareable rule template marketplace providing:
 * - Pre-built rule templates by category
 * - Industry-specific compliance templates
 * - Template versioning and ratings
 * - One-click deployment with customization
 * - Template sharing across tenants
 */

import React, { useState, useEffect, useCallback, useMemo } from 'react';
import {
  Package,
  Search,
  Filter,
  Star,
  Download,
  Upload,
  Check,
  Eye,
  Copy
} from 'lucide-react';

// ============================================================================
// Types
// ============================================================================

interface RuleTemplate {
  id: string;
  name: string;
  description: string;
  category: TemplateCategory;
  industry?: string[];
  tags: string[];
  author: TemplateAuthor;
  version: string;
  rating: number;
  ratingCount: number;
  downloadCount: number;
  isOfficial: boolean;
  isPremium: boolean;
  isPublic: boolean;
  createdAt: Date;
  updatedAt: Date;
  preview: RuleTemplatePreview;
  documentation?: string;
  compatibility: string[];
}

interface TemplateCategory {
  id: string;
  name: string;
  icon: string;
  color: string;
}

interface TemplateAuthor {
  id: string;
  name: string;
  organization?: string;
  verified: boolean;
  avatarUrl?: string;
}

interface RuleTemplatePreview {
  conditionCount: number;
  actionCount: number;
  estimatedComplexity: 'simple' | 'moderate' | 'complex';
  sampleConditions: string[];
  sampleActions: string[];
}

interface TemplateFilter {
  search: string;
  categories: string[];
  industries: string[];
  tags: string[];
  onlyOfficial: boolean;
  onlyFree: boolean;
  minRating: number;
  sortBy: 'popular' | 'newest' | 'rating' | 'name';
}

interface RuleTemplateMarketplaceProps {
  tenantId: string;
  datasourceId: string;
  onSelectTemplate: (template: RuleTemplate) => void;
  onDeployTemplate: (template: RuleTemplate, customizations?: Record<string, unknown>) => void;
}

// ============================================================================
// Constants
// ============================================================================

const CATEGORIES: TemplateCategory[] = [
  { id: 'data_quality', name: 'Data Quality', icon: 'database', color: 'blue' },
  { id: 'compliance', name: 'Compliance', icon: 'shield', color: 'green' },
  { id: 'fraud', name: 'Fraud Detection', icon: 'alert', color: 'red' },
  { id: 'aml', name: 'AML/KYC', icon: 'users', color: 'purple' },
  { id: 'trading', name: 'Trading Rules', icon: 'trending', color: 'orange' },
  { id: 'reporting', name: 'Regulatory Reporting', icon: 'file', color: 'indigo' }
];

const INDUSTRIES = [
  'Financial Services',
  'Banking',
  'Asset Management',
  'Insurance',
  'Healthcare',
  'Retail',
  'Manufacturing'
];

// ============================================================================
// Mock Data
// ============================================================================

const generateMockTemplates = (): RuleTemplate[] => [
  {
    id: 'tmpl-1',
    name: 'SEC Rule 10b5-1 Compliance',
    description: 'Comprehensive insider trading detection and pre-clearance workflow for SEC Rule 10b5-1 compliance.',
    category: CATEGORIES[1],
    industry: ['Financial Services', 'Asset Management'],
    tags: ['SEC', 'insider-trading', 'pre-clearance', 'compliance'],
    author: { id: 'official', name: 'Semlayer Team', verified: true },
    version: '2.1.0',
    rating: 4.8,
    ratingCount: 156,
    downloadCount: 2340,
    isOfficial: true,
    isPremium: false,
    isPublic: true,
    createdAt: new Date('2024-06-15'),
    updatedAt: new Date('2024-11-01'),
    preview: {
      conditionCount: 12,
      actionCount: 5,
      estimatedComplexity: 'complex',
      sampleConditions: [
        'trade.employee_type in ["executive", "insider"]',
        'trade.window_status == "closed"',
        'trade.pre_clearance_status != "approved"'
      ],
      sampleActions: ['block_trade', 'notify_compliance', 'log_violation']
    },
    compatibility: ['v2.0+'],
    documentation: '# SEC Rule 10b5-1 Compliance Template\n\nThis template provides...'
  },
  {
    id: 'tmpl-2',
    name: 'AML Transaction Monitoring',
    description: 'Anti-money laundering transaction monitoring with suspicious activity detection and SAR filing workflow.',
    category: CATEGORIES[3],
    industry: ['Banking', 'Financial Services'],
    tags: ['AML', 'SAR', 'transaction-monitoring', 'FinCEN'],
    author: { id: 'official', name: 'Semlayer Team', verified: true },
    version: '3.0.0',
    rating: 4.9,
    ratingCount: 289,
    downloadCount: 4521,
    isOfficial: true,
    isPremium: true,
    isPublic: true,
    createdAt: new Date('2024-01-10'),
    updatedAt: new Date('2024-10-28'),
    preview: {
      conditionCount: 18,
      actionCount: 8,
      estimatedComplexity: 'complex',
      sampleConditions: [
        'transaction.amount > threshold_config.sar_threshold',
        'transaction.pattern matches "structuring"',
        'customer.risk_score > 75'
      ],
      sampleActions: ['flag_suspicious', 'generate_sar', 'escalate_to_bsa']
    },
    compatibility: ['v2.5+']
  },
  {
    id: 'tmpl-3',
    name: 'Wash Trade Detection',
    description: 'Real-time wash trade detection with configurable thresholds for time windows and counterparty matching.',
    category: CATEGORIES[4],
    industry: ['Financial Services', 'Asset Management'],
    tags: ['wash-trade', 'market-manipulation', 'surveillance'],
    author: { id: 'community-1', name: 'TradeTech Solutions', organization: 'TradeTech', verified: true },
    version: '1.5.2',
    rating: 4.6,
    ratingCount: 87,
    downloadCount: 1234,
    isOfficial: false,
    isPremium: false,
    isPublic: true,
    createdAt: new Date('2024-03-20'),
    updatedAt: new Date('2024-09-15'),
    preview: {
      conditionCount: 8,
      actionCount: 4,
      estimatedComplexity: 'moderate',
      sampleConditions: [
        'trade.buy_time within 5min of trade.sell_time',
        'trade.buy_counterparty == trade.sell_counterparty',
        'trade.net_position_change < threshold'
      ],
      sampleActions: ['flag_wash_trade', 'alert_surveillance', 'block_settlement']
    },
    compatibility: ['v2.0+']
  },
  {
    id: 'tmpl-4',
    name: 'Data Quality - Null Checks',
    description: 'Comprehensive null and missing value detection across critical data fields with configurable severity.',
    category: CATEGORIES[0],
    industry: [],
    tags: ['null-check', 'data-quality', 'completeness'],
    author: { id: 'official', name: 'Semlayer Team', verified: true },
    version: '1.2.0',
    rating: 4.5,
    ratingCount: 412,
    downloadCount: 8765,
    isOfficial: true,
    isPremium: false,
    isPublic: true,
    createdAt: new Date('2023-11-05'),
    updatedAt: new Date('2024-08-20'),
    preview: {
      conditionCount: 5,
      actionCount: 3,
      estimatedComplexity: 'simple',
      sampleConditions: [
        'record.required_field is null',
        'record.email is empty',
        'record.account_id is missing'
      ],
      sampleActions: ['flag_record', 'quarantine', 'notify_steward']
    },
    compatibility: ['v1.5+']
  },
  {
    id: 'tmpl-5',
    name: 'GDPR Data Subject Rights',
    description: 'Automated GDPR compliance for data subject access requests, erasure, and portability workflows.',
    category: CATEGORIES[1],
    industry: [],
    tags: ['GDPR', 'privacy', 'DSAR', 'data-subject-rights'],
    author: { id: 'community-2', name: 'DataPrivacy Pro', verified: true },
    version: '2.0.1',
    rating: 4.7,
    ratingCount: 198,
    downloadCount: 3421,
    isOfficial: false,
    isPremium: true,
    isPublic: true,
    createdAt: new Date('2024-02-01'),
    updatedAt: new Date('2024-10-05'),
    preview: {
      conditionCount: 10,
      actionCount: 6,
      estimatedComplexity: 'complex',
      sampleConditions: [
        'request.type == "access_request"',
        'request.deadline < 30 days',
        'data_subject.consent_withdrawn == true'
      ],
      sampleActions: ['generate_report', 'notify_dpo', 'initiate_erasure']
    },
    compatibility: ['v2.0+']
  },
  {
    id: 'tmpl-6',
    name: 'Customer Deduplication',
    description: 'MDM rule for identifying and flagging potential duplicate customer records using fuzzy matching.',
    category: CATEGORIES[0],
    industry: [],
    tags: ['MDM', 'deduplication', 'fuzzy-match', 'customer-data'],
    author: { id: 'official', name: 'Semlayer Team', verified: true },
    version: '1.8.0',
    rating: 4.4,
    ratingCount: 234,
    downloadCount: 5678,
    isOfficial: true,
    isPremium: false,
    isPublic: true,
    createdAt: new Date('2024-04-10'),
    updatedAt: new Date('2024-11-10'),
    preview: {
      conditionCount: 7,
      actionCount: 4,
      estimatedComplexity: 'moderate',
      sampleConditions: [
        'fuzzy_match(customer.name, existing.name) > 0.85',
        'customer.email == existing.email',
        'customer.phone matches existing.phone'
      ],
      sampleActions: ['flag_duplicate', 'merge_suggestion', 'notify_steward']
    },
    compatibility: ['v1.8+']
  }
];

// ============================================================================
// Components
// ============================================================================

const TemplateCard: React.FC<{
  template: RuleTemplate;
  onSelect: () => void;
  onDeploy: () => void;
  onPreview: () => void;
}> = ({ template, onSelect, onDeploy, onPreview }) => {
  const getCategoryColor = (color: string) => {
    const colors: Record<string, string> = {
      blue: 'bg-blue-100 text-blue-700 border-blue-200',
      green: 'bg-green-100 text-green-700 border-green-200',
      red: 'bg-red-100 text-red-700 border-red-200',
      purple: 'bg-purple-100 text-purple-700 border-purple-200',
      orange: 'bg-orange-100 text-orange-700 border-orange-200',
      indigo: 'bg-indigo-100 text-indigo-700 border-indigo-200'
    };
    return colors[color] || colors.blue;
  };

  const getComplexityBadge = (complexity: string) => {
    const badges: Record<string, string> = {
      simple: 'bg-green-50 text-green-700',
      moderate: 'bg-yellow-50 text-yellow-700',
      complex: 'bg-red-50 text-red-700'
    };
    return badges[complexity] || badges.moderate;
  };

  return (
    <div className="bg-white rounded-lg border shadow-sm hover:shadow-md transition-shadow overflow-hidden">
      <div className="p-4">
        <div className="flex items-start justify-between mb-3">
          <div className="flex items-center gap-2">
            <span className={`text-xs px-2 py-0.5 rounded border ${getCategoryColor(template.category.color)}`}>
              {template.category.name}
            </span>
            {template.isOfficial && (
              <span className="flex items-center gap-1 text-xs text-blue-600">
                <Check size={12} />
                Official
              </span>
            )}
            {template.isPremium && (
              <span className="flex items-center gap-1 text-xs text-purple-600">
                <Star size={12} />
                Premium
              </span>
            )}
          </div>
          <span className="text-xs text-gray-400">v{template.version}</span>
        </div>
        
        <h3 className="font-semibold text-gray-900 mb-1">{template.name}</h3>
        <p className="text-sm text-gray-600 line-clamp-2 mb-3">{template.description}</p>
        
        <div className="flex items-center gap-4 mb-3">
          <div className="flex items-center gap-1">
            <Star size={14} className="text-yellow-500 fill-yellow-500" />
            <span className="text-sm font-medium">{template.rating.toFixed(1)}</span>
            <span className="text-xs text-gray-400">({template.ratingCount})</span>
          </div>
          <div className="flex items-center gap-1 text-gray-500">
            <Download size={14} />
            <span className="text-sm">{template.downloadCount.toLocaleString()}</span>
          </div>
          <span className={`text-xs px-2 py-0.5 rounded ${getComplexityBadge(template.preview.estimatedComplexity)}`}>
            {template.preview.estimatedComplexity}
          </span>
        </div>
        
        <div className="flex flex-wrap gap-1 mb-3">
          {template.tags.slice(0, 4).map(tag => (
            <span key={tag} className="text-xs px-2 py-0.5 bg-gray-100 text-gray-600 rounded">
              {tag}
            </span>
          ))}
          {template.tags.length > 4 && (
            <span className="text-xs text-gray-400">+{template.tags.length - 4} more</span>
          )}
        </div>
        
        <div className="flex items-center gap-2 text-xs text-gray-500">
          <span className="flex items-center gap-1">
            {template.author.verified && <Check size={12} className="text-green-500" />}
            {template.author.name}
          </span>
          <span>•</span>
          <span>Updated {new Date(template.updatedAt).toLocaleDateString()}</span>
        </div>
      </div>
      
      <div className="flex border-t divide-x">
        <button
          onClick={onPreview}
          className="flex-1 flex items-center justify-center gap-1 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
        >
          <Eye size={14} />
          Preview
        </button>
        <button
          onClick={onSelect}
          className="flex-1 flex items-center justify-center gap-1 px-3 py-2 text-sm text-gray-600 hover:bg-gray-50 transition-colors"
        >
          <Copy size={14} />
          Customize
        </button>
        <button
          onClick={onDeploy}
          className="flex-1 flex items-center justify-center gap-1 px-3 py-2 text-sm text-blue-600 hover:bg-blue-50 transition-colors font-medium"
        >
          <Download size={14} />
          Deploy
        </button>
      </div>
    </div>
  );
};

const TemplatePreviewModal: React.FC<{
  template: RuleTemplate;
  onClose: () => void;
  onDeploy: () => void;
}> = ({ template, onClose, onDeploy }) => (
  <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div className="bg-white rounded-lg shadow-xl max-w-2xl w-full max-h-[90vh] overflow-hidden flex flex-col">
      <div className="flex items-center justify-between p-4 border-b">
        <div>
          <h2 className="font-semibold text-lg">{template.name}</h2>
          <span className="text-sm text-gray-500">v{template.version} by {template.author.name}</span>
        </div>
        <button
          onClick={onClose}
          className="p-2 text-gray-400 hover:text-gray-600 rounded hover:bg-gray-100"
          title="Close preview"
          aria-label="Close preview"
        >
          ×
        </button>
      </div>
      
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        <div>
          <h3 className="font-medium text-gray-900 mb-2">Description</h3>
          <p className="text-gray-600">{template.description}</p>
        </div>
        
        <div className="grid grid-cols-3 gap-4">
          <div className="bg-gray-50 rounded p-3">
            <div className="text-xs text-gray-500 mb-1">Conditions</div>
            <div className="text-xl font-semibold">{template.preview.conditionCount}</div>
          </div>
          <div className="bg-gray-50 rounded p-3">
            <div className="text-xs text-gray-500 mb-1">Actions</div>
            <div className="text-xl font-semibold">{template.preview.actionCount}</div>
          </div>
          <div className="bg-gray-50 rounded p-3">
            <div className="text-xs text-gray-500 mb-1">Complexity</div>
            <div className="text-xl font-semibold capitalize">{template.preview.estimatedComplexity}</div>
          </div>
        </div>
        
        <div>
          <h3 className="font-medium text-gray-900 mb-2">Sample Conditions</h3>
          <div className="bg-gray-50 rounded p-3 space-y-2">
            {template.preview.sampleConditions.map((cond, i) => (
              <code key={i} className="block text-sm text-gray-700 font-mono">{cond}</code>
            ))}
          </div>
        </div>
        
        <div>
          <h3 className="font-medium text-gray-900 mb-2">Sample Actions</h3>
          <div className="flex flex-wrap gap-2">
            {template.preview.sampleActions.map((action, i) => (
              <span key={i} className="px-2 py-1 bg-purple-100 text-purple-700 rounded text-sm">
                {action}
              </span>
            ))}
          </div>
        </div>
        
        {template.industry && template.industry.length > 0 && (
          <div>
            <h3 className="font-medium text-gray-900 mb-2">Industries</h3>
            <div className="flex flex-wrap gap-2">
              {template.industry.map((ind, i) => (
                <span key={i} className="px-2 py-1 bg-gray-100 text-gray-700 rounded text-sm">
                  {ind}
                </span>
              ))}
            </div>
          </div>
        )}
        
        <div>
          <h3 className="font-medium text-gray-900 mb-2">Compatibility</h3>
          <span className="text-sm text-gray-600">
            Requires Semlayer {template.compatibility.join(', ')}
          </span>
        </div>
      </div>
      
      <div className="flex items-center justify-end gap-2 p-4 border-t">
        <button
          onClick={onClose}
          className="px-4 py-2 text-gray-700 border rounded hover:bg-gray-50 transition-colors"
        >
          Cancel
        </button>
        <button
          onClick={onDeploy}
          className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors flex items-center gap-2"
        >
          <Download size={16} />
          Deploy Template
        </button>
      </div>
    </div>
  </div>
);

// ============================================================================
// Main Component
// ============================================================================

export const RuleTemplateMarketplace: React.FC<RuleTemplateMarketplaceProps> = ({
  tenantId: _tenantId,
  datasourceId: _datasourceId,
  onSelectTemplate,
  onDeployTemplate
}) => {
  const [templates, setTemplates] = useState<RuleTemplate[]>([]);
  const [filters, setFilters] = useState<TemplateFilter>({
    search: '',
    categories: [],
    industries: [],
    tags: [],
    onlyOfficial: false,
    onlyFree: false,
    minRating: 0,
    sortBy: 'popular'
  });
  const [previewTemplate, setPreviewTemplate] = useState<RuleTemplate | null>(null);
  const [showFilters, setShowFilters] = useState(false);

  useEffect(() => {
    setTemplates(generateMockTemplates());
  }, []);

  const filteredTemplates = useMemo(() => {
    let result = [...templates];
    
    if (filters.search) {
      const searchLower = filters.search.toLowerCase();
      result = result.filter(t => 
        t.name.toLowerCase().includes(searchLower) ||
        t.description.toLowerCase().includes(searchLower) ||
        t.tags.some(tag => tag.toLowerCase().includes(searchLower))
      );
    }
    
    if (filters.categories.length > 0) {
      result = result.filter(t => filters.categories.includes(t.category.id));
    }
    
    if (filters.industries.length > 0) {
      result = result.filter(t => 
        t.industry?.some(ind => filters.industries.includes(ind))
      );
    }
    
    if (filters.onlyOfficial) {
      result = result.filter(t => t.isOfficial);
    }
    
    if (filters.onlyFree) {
      result = result.filter(t => !t.isPremium);
    }
    
    if (filters.minRating > 0) {
      result = result.filter(t => t.rating >= filters.minRating);
    }
    
    // Sort
    switch (filters.sortBy) {
      case 'popular':
        result.sort((a, b) => b.downloadCount - a.downloadCount);
        break;
      case 'newest':
        result.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime());
        break;
      case 'rating':
        result.sort((a, b) => b.rating - a.rating);
        break;
      case 'name':
        result.sort((a, b) => a.name.localeCompare(b.name));
        break;
    }
    
    return result;
  }, [templates, filters]);

  const handleCategoryToggle = useCallback((categoryId: string) => {
    setFilters(prev => ({
      ...prev,
      categories: prev.categories.includes(categoryId)
        ? prev.categories.filter(c => c !== categoryId)
        : [...prev.categories, categoryId]
    }));
  }, []);

  return (
    <div className="flex flex-col h-full bg-gray-50">
      {/* Header */}
      <div className="flex items-center justify-between p-4 border-b bg-white">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-indigo-100 rounded-lg">
            <Package size={20} className="text-indigo-600" />
          </div>
          <div>
            <h2 className="font-semibold text-gray-900">Rule Template Marketplace</h2>
            <p className="text-xs text-gray-500">{filteredTemplates.length} templates available</p>
          </div>
        </div>
        
        <div className="flex items-center gap-2">
          <button
            className="flex items-center gap-2 px-3 py-1.5 border rounded hover:bg-gray-50 text-sm"
            title="Share your template"
          >
            <Upload size={14} />
            Publish Template
          </button>
        </div>
      </div>
      
      {/* Search and Filters */}
      <div className="p-4 border-b bg-white">
        <div className="flex items-center gap-3 mb-3">
          <div className="flex-1 relative">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" />
            <input
              type="text"
              value={filters.search}
              onChange={(e) => setFilters(prev => ({ ...prev, search: e.target.value }))}
              placeholder="Search templates by name, description, or tags..."
              className="w-full pl-9 pr-4 py-2 border rounded-lg text-sm"
            />
          </div>
          
          <button
            onClick={() => setShowFilters(!showFilters)}
            className={`flex items-center gap-2 px-3 py-2 border rounded-lg text-sm transition-colors ${
              showFilters ? 'bg-gray-100' : 'hover:bg-gray-50'
            }`}
          >
            <Filter size={14} />
            Filters
            {(filters.categories.length > 0 || filters.onlyOfficial || filters.onlyFree) && (
              <span className="w-5 h-5 bg-blue-600 text-white rounded-full text-xs flex items-center justify-center">
                {filters.categories.length + (filters.onlyOfficial ? 1 : 0) + (filters.onlyFree ? 1 : 0)}
              </span>
            )}
          </button>
          
          <select
            value={filters.sortBy}
            onChange={(e) => setFilters(prev => ({ ...prev, sortBy: e.target.value as TemplateFilter['sortBy'] }))}
            className="px-3 py-2 border rounded-lg text-sm"
            title="Sort templates"
            aria-label="Sort templates"
          >
            <option value="popular">Most Popular</option>
            <option value="newest">Newest</option>
            <option value="rating">Highest Rated</option>
            <option value="name">Name A-Z</option>
          </select>
        </div>
        
        {/* Category Pills */}
        <div className="flex flex-wrap gap-2">
          {CATEGORIES.map(cat => (
            <button
              key={cat.id}
              onClick={() => handleCategoryToggle(cat.id)}
              className={`px-3 py-1.5 rounded-full text-sm transition-colors ${
                filters.categories.includes(cat.id)
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
              }`}
            >
              {cat.name}
            </button>
          ))}
        </div>
        
        {/* Expanded Filters */}
        {showFilters && (
          <div className="mt-4 pt-4 border-t grid grid-cols-4 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-2">Industry</label>
              <select
                value=""
                onChange={(e) => {
                  if (e.target.value) {
                    setFilters(prev => ({
                      ...prev,
                      industries: [...prev.industries, e.target.value]
                    }));
                  }
                }}
                className="w-full px-3 py-2 border rounded text-sm"
                title="Select industry"
                aria-label="Select industry"
              >
                <option value="">All Industries</option>
                {INDUSTRIES.map(ind => (
                  <option key={ind} value={ind}>{ind}</option>
                ))}
              </select>
            </div>
            
            <div>
              <label className="block text-xs font-medium text-gray-500 mb-2">Min Rating</label>
              <select
                value={filters.minRating}
                onChange={(e) => setFilters(prev => ({ ...prev, minRating: parseFloat(e.target.value) }))}
                className="w-full px-3 py-2 border rounded text-sm"
                title="Minimum rating"
                aria-label="Minimum rating"
              >
                <option value={0}>Any Rating</option>
                <option value={4}>4+ Stars</option>
                <option value={4.5}>4.5+ Stars</option>
              </select>
            </div>
            
            <div className="flex flex-col gap-2">
              <label className="block text-xs font-medium text-gray-500">Options</label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={filters.onlyOfficial}
                  onChange={(e) => setFilters(prev => ({ ...prev, onlyOfficial: e.target.checked }))}
                  className="rounded"
                />
                Official Only
              </label>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={filters.onlyFree}
                  onChange={(e) => setFilters(prev => ({ ...prev, onlyFree: e.target.checked }))}
                  className="rounded"
                />
                Free Only
              </label>
            </div>
            
            <div className="flex items-end">
              <button
                onClick={() => setFilters({
                  search: '',
                  categories: [],
                  industries: [],
                  tags: [],
                  onlyOfficial: false,
                  onlyFree: false,
                  minRating: 0,
                  sortBy: 'popular'
                })}
                className="px-3 py-2 text-sm text-gray-600 hover:text-gray-900"
              >
                Clear All Filters
              </button>
            </div>
          </div>
        )}
      </div>
      
      {/* Template Grid */}
      <div className="flex-1 overflow-auto p-4">
        {filteredTemplates.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredTemplates.map(template => (
              <TemplateCard
                key={template.id}
                template={template}
                onSelect={() => onSelectTemplate(template)}
                onDeploy={() => onDeployTemplate(template)}
                onPreview={() => setPreviewTemplate(template)}
              />
            ))}
          </div>
        ) : (
          <div className="text-center py-12">
            <Package size={48} className="mx-auto text-gray-300 mb-4" />
            <h3 className="font-medium text-gray-600 mb-2">No templates found</h3>
            <p className="text-sm text-gray-500">Try adjusting your filters or search terms</p>
          </div>
        )}
      </div>
      
      {/* Preview Modal */}
      {previewTemplate && (
        <TemplatePreviewModal
          template={previewTemplate}
          onClose={() => setPreviewTemplate(null)}
          onDeploy={() => {
            onDeployTemplate(previewTemplate);
            setPreviewTemplate(null);
          }}
        />
      )}
    </div>
  );
};

export default RuleTemplateMarketplace;
