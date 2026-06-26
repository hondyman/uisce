import { useState, useEffect, useCallback, useMemo } from 'react';
import { devWarn, devError } from '../utils/devLogger';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Button } from './ui/button';
import { Input } from './ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';

import { RegistryBundle } from '../types/bundles';
type Bundle = RegistryBundle;

// Using centralized RegistryBundle/RegistryMetric types from ../types/bundles

export default function BundleExplorer() {
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [filteredBundles, setFilteredBundles] = useState<Bundle[]>([]);
  const [loading, setLoading] = useState(true);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedDomain, setSelectedDomain] = useState<string>('all');
  const [selectedAudience, setSelectedAudience] = useState<string>('all');
  const [selectedFunction, setSelectedFunction] = useState<string>('all');

  useEffect(() => {
    loadBundles();
  }, []);

  const filterBundles = useCallback(() => {
    let filtered = bundles;

    // Filter by search term
    if (searchTerm) {
      filtered = filtered.filter(bundle =>
        bundle.bundle_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        bundle.domain.toLowerCase().includes(searchTerm.toLowerCase()) ||
        bundle.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
        bundle.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase()))
      );
    }

    // Filter by domain
    if (selectedDomain !== 'all') {
      filtered = filtered.filter(bundle => bundle.domain === selectedDomain);
    }

    // Filter by audience
    if (selectedAudience !== 'all') {
      filtered = filtered.filter(bundle =>
        bundle.audience.includes(selectedAudience)
      );
    }

    // Filter by DAX function
    if (selectedFunction !== 'all') {
      filtered = filtered.filter(bundle =>
        bundle.functions?.some(fn => fn.name === selectedFunction) ||
        bundle.metrics.some(metric => metric.functions_used?.includes(selectedFunction))
      );
    }

    setFilteredBundles(filtered);
  }, [bundles, searchTerm, selectedDomain, selectedAudience, selectedFunction]);

  useEffect(() => {
    filterBundles();
  }, [filterBundles]);

  const loadBundles = async () => {
    try {
      // Load bundles from the semantic registry
      const domains = ['banking', 'insurance', 'capital_markets', 'regulatory', 'healthcare', 'retail', 'wealth_management', 'financial_services', 'fixed_income'];
      const loadedBundles: Bundle[] = [];

      for (const domain of domains) {
        try {
          const response = await fetch(`/api/semantic/bundles/${domain}`);
          if (response.ok) {
            const bundle = await response.json();
            loadedBundles.push(bundle);
          }
        } catch (error) {
          devWarn(`Failed to load ${domain} bundle:`, error);
        }
      }

      setBundles(loadedBundles);
    } catch (error) {
      devError('Failed to load bundles:', error);
    } finally {
      setLoading(false);
    }
  };

  // Memoize getters that derive dropdown options from bundles to avoid recalculation
  const getAllDomains = useMemo(() => {
    return () => Array.from(new Set(bundles.map(b => b.domain)));
  }, [bundles]);

  const getAllAudiences = useMemo(() => {
    return () => {
      const audiences = new Set<string>();
      bundles.forEach(bundle => {
        bundle.audience.forEach(audience => audiences.add(audience));
      });
      return Array.from(audiences);
    };
  }, [bundles]);

  const getAllFunctions = useMemo(() => {
    return () => {
      const functions = new Set<string>();
      bundles.forEach(bundle => {
        bundle.functions?.forEach(fn => functions.add(fn.name));
        bundle.metrics.forEach(metric => {
          metric.functions_used?.forEach(fn => functions.add(fn));
        });
      });
      return Array.from(functions).sort();
    };
  }, [bundles]);


  if (loading) {
    return <div className="flex justify-center items-center h-64">Loading bundles...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col space-y-4">
        <h1 className="text-3xl font-bold">Bundle Explorer</h1>
        <p className="text-gray-600">
          Discover and explore DAX-powered metric bundles across all domains
        </p>

        {/* Filters */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Input
            placeholder="Search bundles..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
          />

          <Select value={selectedDomain} onValueChange={setSelectedDomain}>
            <SelectTrigger>
              <SelectValue placeholder="All Domains" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Domains</SelectItem>
              {getAllDomains().filter(d => d && d.trim() !== '').map(domain => (
                <SelectItem key={domain} value={domain}>{domain}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={selectedAudience} onValueChange={setSelectedAudience}>
            <SelectTrigger>
              <SelectValue placeholder="All Audiences" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Audiences</SelectItem>
              {getAllAudiences().filter(a => a && a.trim() !== '').map(audience => (
                <SelectItem key={audience} value={audience}>{audience}</SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={selectedFunction} onValueChange={setSelectedFunction}>
            <SelectTrigger>
              <SelectValue placeholder="All Functions" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Functions</SelectItem>
              {getAllFunctions().filter(f => f && f.trim() !== '').map(func => (
                <SelectItem key={func} value={func}>{func}</SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      {/* Bundle Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        {filteredBundles.map(bundle => (
          <Card key={bundle.bundle_id} className="hover:shadow-lg transition-shadow">
            <CardHeader>
              <div className="flex justify-between items-start">
                <CardTitle className="text-lg">{bundle.domain}</CardTitle>
                <Badge variant="outline">{bundle.version}</Badge>
              </div>
              <p className="text-sm text-gray-600">{bundle.bundle_id}</p>
            </CardHeader>

            <CardContent>
              <div className="space-y-4">
                {/* Audience */}
                <div>
                  <h4 className="text-sm font-medium mb-2">Audience</h4>
                  <div className="flex flex-wrap gap-1">
                    {bundle.audience.map(aud => (
                      <Badge key={aud} variant="secondary" className="text-xs">
                        {aud}
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* Tags */}
                <div>
                  <h4 className="text-sm font-medium mb-2">Tags</h4>
                  <div className="flex flex-wrap gap-1">
                    {bundle.tags.map(tag => (
                      <Badge key={tag} variant="outline" className="text-xs">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </div>

                {/* DAX Functions */}
                {bundle.functions && bundle.functions.length > 0 && (
                  <div>
                    <h4 className="text-sm font-medium mb-2">DAX Functions ({bundle.functions.length})</h4>
                    <div className="flex flex-wrap gap-1">
                      {bundle.functions.slice(0, 5).map(fn => (
                        <Badge key={fn.name} className="text-xs">
                          {fn.badge} {fn.name}
                        </Badge>
                      ))}
                      {bundle.functions.length > 5 && (
                        <Badge variant="outline" className="text-xs">
                          +{bundle.functions.length - 5} more
                        </Badge>
                      )}
                    </div>
                  </div>
                )}

                {/* Metrics Summary */}
                <div className="flex justify-between items-center text-sm">
                  <span>{bundle.metrics.length} metrics</span>
                  <span className="text-green-600">
                {bundle.metrics.filter(m => m.governance && m.governance.status === 'golden').length} golden
                  </span>
                </div>

                <Button variant="outline" className="w-full">
                  Explore Metrics
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredBundles.length === 0 && (
        <div className="text-center py-12">
          <p className="text-gray-500">No bundles match your current filters.</p>
        </div>
      )}
    </div>
  );
}
