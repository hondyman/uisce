import { useState, useEffect } from 'react';
import { devWarn, devError } from '../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

interface DAXFunction {
  name: string;
  class: string;
  badge: string;
  description: string;
  category?: string;
  minArgs?: number;
  maxArgs?: number;
  returnType?: string;
  examples?: string[];
}

interface FunctionUsage {
  bundle_id: string;
  domain: string;
  metrics_count: number;
  sample_metrics: string[];
}

export default function DAXFunctionReference() {
  const [functions, setFunctions] = useState<DAXFunction[]>([]);
  const [functionUsage, setFunctionUsage] = useState<Record<string, FunctionUsage[]>>({});
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('all');

  useEffect(() => {
    loadDAXFunctions();
  }, []);

  const loadDAXFunctions = async () => {
    try {
      // Load DAX functions from all bundles
      const domains = ['banking', 'insurance', 'capital_markets', 'regulatory', 'healthcare', 'retail', 'wealth_management'];
      const allFunctions = new Map<string, DAXFunction>();
      const usageMap: Record<string, FunctionUsage[]> = {};

      for (const domain of domains) {
        try {
          const response = await fetch(`/api/bundles/${domain}`);
          if (response.ok) {
            const bundle = await response.json();

            // Collect functions from bundle
            if (bundle.functions) {
              bundle.functions.forEach((fn: DAXFunction) => {
                if (!allFunctions.has(fn.name)) {
                  allFunctions.set(fn.name, fn);
                }
              });
            }

            // Track usage in metrics
            bundle.metrics.forEach((metric: any) => {
              if (metric.functions_used) {
                metric.functions_used.forEach((fnName: string) => {
                  if (!usageMap[fnName]) {
                    usageMap[fnName] = [];
                  }

                  const existing = usageMap[fnName].find(u => u.bundle_id === bundle.bundle_id);
                  if (existing) {
                    existing.metrics_count++;
                    if (existing.sample_metrics.length < 3) {
                      existing.sample_metrics.push(metric.node_id);
                    }
                  } else {
                    usageMap[fnName].push({
                      bundle_id: bundle.bundle_id,
                      domain: bundle.domain,
                      metrics_count: 1,
                      sample_metrics: [metric.node_id]
                    });
                  }
                });
              }
            });
          }
        } catch (error) {
          devWarn(`Failed to load ${domain} bundle:`, error);
        }
      }

      setFunctions(Array.from(allFunctions.values()));
      setFunctionUsage(usageMap);
    } catch (error) {
      devError('Failed to load DAX functions:', error);
    } finally {
      setLoading(false);
    }
  };

  const filteredFunctions = functions.filter(fn => {
    const matchesSearch = fn.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
                         fn.description.toLowerCase().includes(searchTerm.toLowerCase());
    const matchesCategory = selectedCategory === 'all' || fn.class === selectedCategory;
    return matchesSearch && matchesCategory;
  });

  const getCategories = () => {
    return Array.from(new Set(functions.map(f => f.class)));
  };

  const getBadgeColor = (badgeClass: string) => {
    switch (badgeClass) {
      case 'DAX-Iterator': return 'bg-yellow-100 text-yellow-800';
      case 'DAX-Logical': return 'bg-orange-100 text-orange-800';
      case 'DAX-Math': return 'bg-blue-100 text-blue-800';
      case 'DAX-Statistical': return 'bg-purple-100 text-purple-800';
      case 'DAX-Time': return 'bg-green-100 text-green-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  if (loading) {
    return <div className="flex justify-center items-center h-64">Loading DAX functions...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col space-y-4">
        <h1 className="text-3xl font-bold">DAX Function Reference</h1>
        <p className="text-gray-600">
          Comprehensive guide to all DAX functions available in your semantic layer
        </p>

        {/* Filters */}
        <div className="flex space-x-4">
          <Input
            placeholder="Search functions..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="max-w-sm"
          />

          <Select value={selectedCategory} onValueChange={setSelectedCategory}>
            <SelectTrigger className="max-w-sm">
              <SelectValue placeholder="All Categories" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Categories</SelectItem>
              {getCategories().map(category => (
                <SelectItem key={category} value={category}>{category}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Function Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredFunctions.map(fn => (
          <Card key={fn.name} className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex justify-between items-start">
                <CardTitle className="text-lg font-mono">{fn.name}</CardTitle>
                <Badge className={getBadgeColor(fn.class)}>
                  {fn.badge}
                </Badge>
              </div>
              <Badge variant="outline" className="w-fit">
                {fn.class}
              </Badge>
            </CardHeader>

            <CardContent>
              <div className="space-y-4">
                <p className="text-sm text-gray-600">{fn.description}</p>

                {/* Usage Statistics */}
                {functionUsage[fn.name] && (
                  <div>
                    <h4 className="text-sm font-medium mb-2">Usage</h4>
                    <div className="space-y-2">
                      {functionUsage[fn.name].map(usage => (
                        <div key={usage.bundle_id} className="flex justify-between items-center text-xs">
                          <span className="text-gray-600">{usage.domain}</span>
                          <span className="font-medium">{usage.metrics_count} metrics</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* Sample Metrics */}
                {functionUsage[fn.name] && functionUsage[fn.name].some(u => u.sample_metrics.length > 0) && (
                  <div>
                    <h4 className="text-sm font-medium mb-2">Sample Metrics</h4>
                    <div className="flex flex-wrap gap-1">
                      {functionUsage[fn.name]
                        .flatMap(u => u.sample_metrics.slice(0, 2))
                        .slice(0, 3)
                        .map(metric => (
                          <Badge key={metric} variant="secondary" className="text-xs">
                            {metric}
                          </Badge>
                        ))}
                    </div>
                  </div>
                )}

                <Button variant="outline" size="sm" className="w-full">
                  View Examples
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredFunctions.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">No functions match your search criteria.</p>
        </div>
      )}

      {/* Summary Stats */}
      <Card>
        <CardHeader>
          <CardTitle>Function Library Summary</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center">
              <div className="text-2xl font-bold text-blue-600">{functions.length}</div>
              <div className="text-sm text-gray-600">Total Functions</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-green-600">
                {functions.filter(f => f.class === 'DAX-Iterator').length}
              </div>
              <div className="text-sm text-gray-600">Iterator Functions</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-orange-600">
                {functions.filter(f => f.class === 'DAX-Logical').length}
              </div>
              <div className="text-sm text-gray-600">Logical Functions</div>
            </div>
            <div className="text-center">
              <div className="text-2xl font-bold text-purple-600">
                {functions.filter(f => f.class === 'DAX-Statistical').length}
              </div>
              <div className="text-sm text-gray-600">Statistical Functions</div>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
