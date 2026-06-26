import React, { useState, useEffect } from 'react';
import {
  Package, Search, Star, Users, Clock, ChevronRight, Copy, CheckCircle,
  FileText, AlertCircle, BookOpen, TrendingUp, Award, Filter, X, Eye,
  Download, Heart, MessageSquare, ThumbsUp, BarChart3
} from 'lucide-react';

// ============================================================================
// TYPES
// ============================================================================

interface ProcessTemplate {
  id: string;
  template_key: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  icon_name: string;
  difficulty_level: 'beginner' | 'intermediate' | 'advanced';
  estimated_setup_time_minutes: number;
  is_official: boolean;
  is_featured: boolean;
  template_definition: any;
  customization_guide: string;
  example_use_cases: string[];
  author_name: string;
  author_organization: string;
  version: string;
  usage_count: number;
  clone_count: number;
  favorite_count: number;
  rating_average: number;
  rating_count: number;
  screenshot_url: string;
  documentation_url: string;
  demo_video_url: string;
  created_at: string;
  published_at: string;
}

interface TemplateCategory {
  id: string;
  category_key: string;
  display_name: string;
  description: string;
  icon_name: string;
  template_count: number;
}

interface TemplateRating {
  id: string;
  rating: number;
  review_text: string;
  review_title: string;
  reviewer_name: string;
  reviewer_role: string;
  helpful_count: number;
  is_verified_user: boolean;
  created_at: string;
}

interface Tenant {
  id: string;
  display_name: string;
}

interface Datasource {
  id: string;
  source_name: string;
}

interface Props {
  tenant: Tenant;
  datasource: Datasource;
  onTemplateCloned?: (processId: string) => void;
}

// ============================================================================
// MAIN COMPONENT
// ============================================================================

export const ProcessTemplatesLibrary: React.FC<Props> = ({ tenant, datasource, onTemplateCloned }) => {
  const [viewMode, setViewMode] = useState<'browse' | 'preview' | 'clones'>('browse');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState<'rating' | 'usage' | 'recent' | 'name'>('rating');
  const [difficultyFilter, setDifficultyFilter] = useState<string>('all');
  
  const [templates, setTemplates] = useState<ProcessTemplate[]>([]);
  const [categories, setCategories] = useState<TemplateCategory[]>([]);
  const [featuredTemplates, setFeaturedTemplates] = useState<ProcessTemplate[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<ProcessTemplate | null>(null);
  const [ratings, setRatings] = useState<TemplateRating[]>([]);
  
  const [showCloneModal, setShowCloneModal] = useState(false);
  const [cloning, setCloning] = useState(false);
  const [processName, setProcessName] = useState('');
  const [customizationNotes, setCustomizationNotes] = useState('');

  // Fetch categories
  useEffect(() => {
    fetchCategories();
    fetchFeaturedTemplates();
  }, []);

  // Fetch templates when filters change
  useEffect(() => {
    if (viewMode === 'browse') {
      fetchTemplates();
    }
  }, [selectedCategory, searchQuery, sortBy, difficultyFilter, viewMode]);

  const fetchCategories = async () => {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id
      });
      const response = await fetch(`/api/templates/categories?${params}`);
      const data = await response.json();
      setCategories(data || []);
    } catch (error) {
      console.error('Failed to fetch categories:', error);
    }
  };

  const fetchFeaturedTemplates = async () => {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id
      });
      const response = await fetch(`/api/templates/featured?${params}`);
      const data = await response.json();
      setFeaturedTemplates(data || []);
    } catch (error) {
      console.error('Failed to fetch featured templates:', error);
    }
  };

  const fetchTemplates = async () => {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
        sort_by: sortBy
      });
      
      if (selectedCategory !== 'all') {
        params.set('category', selectedCategory);
      }
      if (searchQuery) {
        params.set('search', searchQuery);
      }
      if (difficultyFilter !== 'all') {
        params.set('difficulty', difficultyFilter);
      }
      
      const response = await fetch(`/api/templates?${params}`);
      const data = await response.json();
      setTemplates(data || []);
    } catch (error) {
      console.error('Failed to fetch templates:', error);
    }
  };

  const fetchTemplateRatings = async (templateId: string) => {
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id
      });
      const response = await fetch(`/api/templates/${templateId}/ratings?${params}`);
      const data = await response.json();
      setRatings(data || []);
    } catch (error) {
      console.error('Failed to fetch ratings:', error);
    }
  };

  const handleTemplateClick = async (template: ProcessTemplate) => {
    setSelectedTemplate(template);
    setProcessName(template.name + ' (Copy)');
    setViewMode('preview');
    fetchTemplateRatings(template.id);
  };

  const handleCloneTemplate = async () => {
    if (!selectedTemplate) return;
    
    setCloning(true);
    try {
      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id
      });
      
      const response = await fetch(`/api/templates/clone/${selectedTemplate.id}?${params}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          process_name: processName,
          customization_notes: customizationNotes,
          cloned_by: 'current_user' // Replace with actual user ID
        })
      });
      
      if (response.ok) {
        const result = await response.json();
        alert(`Template cloned successfully! Process ID: ${result.process_id}`);
        setShowCloneModal(false);
        setViewMode('browse');
        if (onTemplateCloned) {
          onTemplateCloned(result.process_id);
        }
      } else {
        alert('Failed to clone template');
      }
    } catch (error) {
      console.error('Failed to clone template:', error);
      alert('Failed to clone template');
    } finally {
      setCloning(false);
    }
  };

  const getCategoryIcon = (iconName: string) => {
    const icons: any = {
      CheckCircle, FileText, Users, Clock, AlertCircle, Package, TrendingUp
    };
    return icons[iconName] || FileText;
  };

  const getDifficultyColor = (level: string) => {
    switch (level) {
      case 'beginner': return 'bg-green-100 text-green-700';
      case 'intermediate': return 'bg-yellow-100 text-yellow-700';
      case 'advanced': return 'bg-red-100 text-red-700';
      default: return 'bg-gray-100 text-gray-700';
    }
  };

  return (
    <div className="h-full flex flex-col bg-gray-50">
      {/* Header */}
      <div className="bg-gradient-to-r from-indigo-600 to-purple-600 text-white p-6 shadow-lg">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <Package size={32} />
            <div>
              <h1 className="text-2xl font-bold">Process Templates Library</h1>
              <p className="text-indigo-100 text-sm">Start faster with pre-built workflow templates</p>
            </div>
          </div>
          
          {/* View Mode Buttons */}
          <div className="flex gap-2">
            <button
              onClick={() => setViewMode('browse')}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                viewMode === 'browse'
                  ? 'bg-white text-indigo-600'
                  : 'bg-indigo-500 text-white hover:bg-indigo-400'
              }`}
            >
              Browse Templates
            </button>
            <button
              onClick={() => setViewMode('clones')}
              className={`px-4 py-2 rounded-lg font-medium transition-all ${
                viewMode === 'clones'
                  ? 'bg-white text-indigo-600'
                  : 'bg-indigo-500 text-white hover:bg-indigo-400'
              }`}
            >
              My Clones
            </button>
          </div>
        </div>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Browse View */}
        {viewMode === 'browse' && (
          <>
            {/* Sidebar */}
            <div className="w-64 bg-white border-r border-gray-200 p-4 overflow-y-auto">
              <div className="space-y-6">
                {/* Search */}
                <div>
                  <label className="text-sm font-medium text-gray-700 mb-2 block">Search</label>
                  <div className="relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400" size={18} />
                    <input
                      type="text"
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      placeholder="Search templates..."
                      className="w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500 focus:border-transparent"
                    />
                  </div>
                </div>

                {/* Categories */}
                <div>
                  <label className="text-sm font-medium text-gray-700 mb-2 block">Categories</label>
                  <div className="space-y-1">
                    <button
                      onClick={() => setSelectedCategory('all')}
                      className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                        selectedCategory === 'all'
                          ? 'bg-indigo-100 text-indigo-700'
                          : 'text-gray-700 hover:bg-gray-100'
                      }`}
                    >
                      <span>All Templates</span>
                      <span className="text-xs">{templates.length}</span>
                    </button>
                    {categories.map((cat) => {
                      const Icon = getCategoryIcon(cat.icon_name);
                      return (
                        <button
                          key={cat.id}
                          onClick={() => setSelectedCategory(cat.category_key)}
                          className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
                            selectedCategory === cat.category_key
                              ? 'bg-indigo-100 text-indigo-700'
                              : 'text-gray-700 hover:bg-gray-100'
                          }`}
                        >
                          <div className="flex items-center gap-2">
                            <Icon size={16} />
                            <span>{cat.display_name}</span>
                          </div>
                          <span className="text-xs">{cat.template_count}</span>
                        </button>
                      );
                    })}
                  </div>
                </div>

                {/* Difficulty Filter */}
                <div>
                  <label className="text-sm font-medium text-gray-700 mb-2 block">Difficulty</label>
                  <select
                    value={difficultyFilter}
                    onChange={(e) => setDifficultyFilter(e.target.value)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="all">All Levels</option>
                    <option value="beginner">Beginner</option>
                    <option value="intermediate">Intermediate</option>
                    <option value="advanced">Advanced</option>
                  </select>
                </div>

                {/* Sort By */}
                <div>
                  <label className="text-sm font-medium text-gray-700 mb-2 block">Sort By</label>
                  <select
                    value={sortBy}
                    onChange={(e) => setSortBy(e.target.value as any)}
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  >
                    <option value="rating">Highest Rated</option>
                    <option value="usage">Most Popular</option>
                    <option value="recent">Recently Added</option>
                    <option value="name">Name (A-Z)</option>
                  </select>
                </div>
              </div>
            </div>

            {/* Main Content */}
            <div className="flex-1 overflow-y-auto p-6">
              {/* Featured Templates */}
              {selectedCategory === 'all' && featuredTemplates.length > 0 && (
                <div className="mb-8">
                  <div className="flex items-center gap-2 mb-4">
                    <Award className="text-yellow-500" size={24} />
                    <h2 className="text-xl font-bold text-gray-900">Featured Templates</h2>
                  </div>
                  <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {featuredTemplates.map((template) => (
                      <TemplateCard
                        key={template.id}
                        template={template}
                        onClick={() => handleTemplateClick(template)}
                        getDifficultyColor={getDifficultyColor}
                      />
                    ))}
                  </div>
                </div>
              )}

              {/* All Templates */}
              <div>
                <h2 className="text-xl font-bold text-gray-900 mb-4">
                  {selectedCategory === 'all' ? 'All Templates' : categories.find(c => c.category_key === selectedCategory)?.display_name}
                </h2>
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {templates.map((template) => (
                    <TemplateCard
                      key={template.id}
                      template={template}
                      onClick={() => handleTemplateClick(template)}
                      getDifficultyColor={getDifficultyColor}
                    />
                  ))}
                </div>
                {templates.length === 0 && (
                  <div className="text-center py-12">
                    <Package className="mx-auto text-gray-400 mb-4" size={48} />
                    <p className="text-gray-500 text-lg">No templates found</p>
                    <p className="text-gray-400 text-sm">Try adjusting your filters</p>
                  </div>
                )}
              </div>
            </div>
          </>
        )}

        {/* Preview View */}
        {viewMode === 'preview' && selectedTemplate && (
          <div className="flex-1 overflow-y-auto">
            <TemplatePreview
              template={selectedTemplate}
              ratings={ratings}
              onBack={() => setViewMode('browse')}
              onClone={() => setShowCloneModal(true)}
              getDifficultyColor={getDifficultyColor}
            />
          </div>
        )}

        {/* My Clones View */}
        {viewMode === 'clones' && (
          <div className="flex-1 overflow-y-auto p-6">
            <div className="text-center py-12">
              <Copy className="mx-auto text-gray-400 mb-4" size={48} />
              <p className="text-gray-500 text-lg">Your cloned templates will appear here</p>
              <p className="text-gray-400 text-sm">Start by cloning a template from the library</p>
            </div>
          </div>
        )}
      </div>

      {/* Clone Modal */}
      {showCloneModal && selectedTemplate && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50 p-4">
          <div className="bg-white rounded-xl shadow-2xl max-w-2xl w-full max-h-[90vh] overflow-y-auto">
            <div className="p-6 border-b border-gray-200">
              <div className="flex items-center justify-between">
                <h3 className="text-xl font-bold text-gray-900">Clone Template</h3>
                <button
                  onClick={() => setShowCloneModal(false)}
                  className="text-gray-400 hover:text-gray-600"
                >
                  <X size={24} />
                </button>
              </div>
            </div>

            <div className="p-6 space-y-6">
              <div>
                <h4 className="font-semibold text-gray-900 mb-2">{selectedTemplate.name}</h4>
                <p className="text-gray-600 text-sm">{selectedTemplate.description}</p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Process Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={processName}
                  onChange={(e) => setProcessName(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  placeholder="Enter a name for your process"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700 mb-2">
                  Customization Notes (Optional)
                </label>
                <textarea
                  value={customizationNotes}
                  onChange={(e) => setCustomizationNotes(e.target.value)}
                  rows={4}
                  className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                  placeholder="Add notes about how you plan to customize this template..."
                />
              </div>

              {selectedTemplate.customization_guide && (
                <div className="bg-blue-50 border border-blue-200 rounded-lg p-4">
                  <div className="flex items-start gap-2">
                    <BookOpen className="text-blue-600 flex-shrink-0 mt-1" size={20} />
                    <div>
                      <h5 className="font-medium text-blue-900 mb-1">Customization Guide</h5>
                      <p className="text-blue-700 text-sm whitespace-pre-wrap">{selectedTemplate.customization_guide}</p>
                    </div>
                  </div>
                </div>
              )}
            </div>

            <div className="p-6 border-t border-gray-200 flex justify-end gap-3">
              <button
                onClick={() => setShowCloneModal(false)}
                className="px-6 py-2 border border-gray-300 rounded-lg hover:bg-gray-50 font-medium"
                disabled={cloning}
              >
                Cancel
              </button>
              <button
                onClick={handleCloneTemplate}
                disabled={!processName || cloning}
                className="px-6 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
              >
                {cloning ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-2 border-white border-t-transparent" />
                    Cloning...
                  </>
                ) : (
                  <>
                    <Copy size={18} />
                    Clone Template
                  </>
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

// ============================================================================
// TEMPLATE CARD COMPONENT
// ============================================================================

const TemplateCard: React.FC<{
  template: ProcessTemplate;
  onClick: () => void;
  getDifficultyColor: (level: string) => string;
}> = ({ template, onClick, getDifficultyColor }) => {
  return (
    <div
      onClick={onClick}
      className="bg-white border border-gray-200 rounded-xl p-5 hover:shadow-lg transition-all cursor-pointer group"
    >
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1">
          <div className="flex items-center gap-2 mb-2">
            <h3 className="font-semibold text-gray-900 group-hover:text-indigo-600 transition-colors">
              {template.name}
            </h3>
            {template.is_official && (
              <span className="px-2 py-0.5 bg-blue-100 text-blue-700 text-xs font-medium rounded">
                Official
              </span>
            )}
          </div>
          <p className="text-sm text-gray-600 line-clamp-2">{template.description}</p>
        </div>
      </div>

      <div className="flex items-center gap-2 mb-3 flex-wrap">
        <span className={`px-2 py-1 text-xs font-medium rounded ${getDifficultyColor(template.difficulty_level)}`}>
          {template.difficulty_level}
        </span>
        <span className="px-2 py-1 bg-gray-100 text-gray-700 text-xs font-medium rounded">
          {template.category}
        </span>
      </div>

      <div className="flex items-center justify-between text-sm text-gray-500">
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1">
            <Star className="text-yellow-500 fill-yellow-500" size={14} />
            <span className="font-medium">{template.rating_average.toFixed(1)}</span>
            <span className="text-xs">({template.rating_count})</span>
          </div>
          <div className="flex items-center gap-1">
            <Users size={14} />
            <span>{template.clone_count}</span>
          </div>
          <div className="flex items-center gap-1">
            <Clock size={14} />
            <span>{template.estimated_setup_time_minutes}m</span>
          </div>
        </div>
        <ChevronRight className="text-gray-400 group-hover:text-indigo-600 transition-colors" size={20} />
      </div>
    </div>
  );
};

// ============================================================================
// TEMPLATE PREVIEW COMPONENT
// ============================================================================

const TemplatePreview: React.FC<{
  template: ProcessTemplate;
  ratings: TemplateRating[];
  onBack: () => void;
  onClone: () => void;
  getDifficultyColor: (level: string) => string;
}> = ({ template, ratings, onBack, onClone, getDifficultyColor }) => {
  return (
    <div className="max-w-5xl mx-auto p-6">
      {/* Back Button */}
      <button
        onClick={onBack}
        className="flex items-center gap-2 text-indigo-600 hover:text-indigo-700 font-medium mb-6"
      >
        <ChevronRight size={20} className="rotate-180" />
        Back to Templates
      </button>

      {/* Header */}
      <div className="bg-white rounded-xl shadow-lg p-8 mb-6">
        <div className="flex items-start justify-between mb-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <h1 className="text-3xl font-bold text-gray-900">{template.name}</h1>
              {template.is_official && (
                <span className="px-3 py-1 bg-blue-100 text-blue-700 text-sm font-medium rounded-lg">
                  Official
                </span>
              )}
              {template.is_featured && (
                <Award className="text-yellow-500" size={24} />
              )}
            </div>
            <p className="text-gray-600 text-lg mb-4">{template.description}</p>
            <div className="flex items-center gap-3 text-sm text-gray-500">
              <span>by {template.author_name}</span>
              {template.author_organization && <span>• {template.author_organization}</span>}
              <span>• v{template.version}</span>
            </div>
          </div>
          <button
            onClick={onClone}
            className="px-6 py-3 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-semibold flex items-center gap-2 shadow-lg hover:shadow-xl transition-all"
          >
            <Copy size={20} />
            Clone Template
          </button>
        </div>

        <div className="flex items-center gap-6 pt-4 border-t border-gray-200">
          <div className="flex items-center gap-2">
            <Star className="text-yellow-500 fill-yellow-500" size={20} />
            <span className="font-semibold text-lg">{template.rating_average.toFixed(1)}</span>
            <span className="text-gray-500">({template.rating_count} ratings)</span>
          </div>
          <div className="flex items-center gap-2 text-gray-600">
            <Users size={20} />
            <span>{template.clone_count} clones</span>
          </div>
          <div className="flex items-center gap-2 text-gray-600">
            <Clock size={20} />
            <span>{template.estimated_setup_time_minutes} min setup</span>
          </div>
          <span className={`px-3 py-1 text-sm font-medium rounded-lg ${getDifficultyColor(template.difficulty_level)}`}>
            {template.difficulty_level}
          </span>
        </div>
      </div>

      {/* Details Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        {/* Example Use Cases */}
        {template.example_use_cases && template.example_use_cases.length > 0 && (
          <div className="bg-white rounded-xl shadow p-6">
            <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
              <FileText size={20} className="text-indigo-600" />
              Example Use Cases
            </h3>
            <ul className="space-y-2">
              {template.example_use_cases.map((useCase, idx) => (
                <li key={idx} className="flex items-start gap-2 text-sm text-gray-700">
                  <CheckCircle size={16} className="text-green-500 flex-shrink-0 mt-0.5" />
                  <span>{useCase}</span>
                </li>
              ))}
            </ul>
          </div>
        )}

        {/* Template Stats */}
        <div className="bg-white rounded-xl shadow p-6">
          <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <BarChart3 size={20} className="text-indigo-600" />
            Statistics
          </h3>
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Total Clones</span>
              <span className="font-semibold">{template.clone_count}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Views</span>
              <span className="font-semibold">{template.usage_count}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Favorites</span>
              <span className="font-semibold">{template.favorite_count}</span>
            </div>
            <div className="flex justify-between text-sm">
              <span className="text-gray-600">Published</span>
              <span className="font-semibold">{new Date(template.published_at!).toLocaleDateString()}</span>
            </div>
          </div>
        </div>

        {/* Tags */}
        {template.tags && template.tags.length > 0 && (
          <div className="bg-white rounded-xl shadow p-6">
            <h3 className="font-semibold text-gray-900 mb-4">Tags</h3>
            <div className="flex flex-wrap gap-2">
              {template.tags.map((tag, idx) => (
                <span key={idx} className="px-3 py-1 bg-gray-100 text-gray-700 text-sm rounded-full">
                  {tag}
                </span>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Customization Guide */}
      {template.customization_guide && (
        <div className="bg-white rounded-xl shadow p-6 mb-6">
          <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
            <BookOpen size={20} className="text-indigo-600" />
            Customization Guide
          </h3>
          <div className="prose prose-sm max-w-none text-gray-700 whitespace-pre-wrap">
            {template.customization_guide}
          </div>
        </div>
      )}

      {/* Ratings & Reviews */}
      <div className="bg-white rounded-xl shadow p-6">
        <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
          <MessageSquare size={20} className="text-indigo-600" />
          Reviews ({ratings.length})
        </h3>
        {ratings.length > 0 ? (
          <div className="space-y-4">
            {ratings.map((rating) => (
              <div key={rating.id} className="border-b border-gray-200 last:border-0 pb-4 last:pb-0">
                <div className="flex items-start justify-between mb-2">
                  <div>
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-gray-900">{rating.reviewer_name}</span>
                      {rating.is_verified_user && (
                        <span className="px-2 py-0.5 bg-green-100 text-green-700 text-xs font-medium rounded">
                          Verified User
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <div className="flex">
                        {[...Array(5)].map((_, i) => (
                          <Star
                            key={i}
                            size={14}
                            className={i < rating.rating ? 'text-yellow-500 fill-yellow-500' : 'text-gray-300'}
                          />
                        ))}
                      </div>
                      <span className="text-xs text-gray-500">
                        {new Date(rating.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                </div>
                {rating.review_title && (
                  <h4 className="font-medium text-gray-900 mb-1">{rating.review_title}</h4>
                )}
                {rating.review_text && (
                  <p className="text-sm text-gray-600 mb-2">{rating.review_text}</p>
                )}
                <div className="flex items-center gap-4 text-xs text-gray-500">
                  <button className="flex items-center gap-1 hover:text-indigo-600">
                    <ThumbsUp size={14} />
                    Helpful ({rating.helpful_count})
                  </button>
                </div>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-gray-500 text-sm">No reviews yet. Be the first to review this template!</p>
        )}
      </div>
    </div>
  );
};
