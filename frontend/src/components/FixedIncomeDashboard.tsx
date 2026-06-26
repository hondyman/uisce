import { useState, useEffect } from 'react';
import { devError } from '../utils/devLogger';
import { useAuthFetch } from '../utils/authFetch';
import { Card, CardContent, CardHeader, CardTitle } from './ui/card';
import { Badge } from './ui/badge';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from './ui/select';
import { Tabs, TabsContent, TabsList, TabsTrigger } from './ui/tabs';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, BarChart, Bar } from 'recharts';

interface FixedIncomeMetric {
  node_id: string;
  category: string;
  description: string;
  badge?: string;
  function_class?: string;
  functions_used?: string[];
  governance: {
    status: string;
  };
  value?: number;
  last_refresh?: string;
}

interface BondData {
  security_id: string;
  clean_price: number;
  dirty_price: number;
  yield_to_maturity: number;
  duration: number;
  convexity: number;
  oas: number;
  total_return: number;
  date: string;
}

export default function FixedIncomeDashboard() {
  const { authFetch } = useAuthFetch();
  const [metrics, setMetrics] = useState<FixedIncomeMetric[]>([]);
  const [bondData, setBondData] = useState<BondData[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedSecurity, setSelectedSecurity] = useState<string>('all');
  const [timeRange, setTimeRange] = useState<string>('30d');

  useEffect(() => {
    loadFixedIncomeData();
  }, []);

  useEffect(() => {
    loadBondData();
  }, [selectedSecurity, timeRange]);

  const loadFixedIncomeData = async () => {
    try {
      // Load fixed income metrics from the bundle
      const response = await authFetch('/api/semantic/bundles/fixed_income');
      if (response.ok) {
        const bundle = (response && (response as any).data !== undefined) ? (response as any).data : await (response as any).json?.();
        setMetrics(bundle.metrics || []);
      }
    } catch (error) {
      devError('Failed to load fixed income metrics:', error);
    }
  };

  const loadBondData = async () => {
    try {
      // Simulate loading bond data - in real implementation, this would come from your API
      const mockData: BondData[] = [
        {
          security_id: 'US_TREASURY_10Y',
          clean_price: 98.50,
          dirty_price: 99.25,
          yield_to_maturity: 4.25,
          duration: 8.92,
          convexity: 85.6,
          oas: 0.15,
          total_return: 2.8,
          date: '2025-09-13'
        },
        {
          security_id: 'CORP_BOND_ABC',
          clean_price: 95.75,
          dirty_price: 96.80,
          yield_to_maturity: 5.10,
          duration: 6.45,
          convexity: 52.3,
          oas: 0.85,
          total_return: 3.2,
          date: '2025-09-13'
        },
        {
          security_id: 'MUNI_BOND_XYZ',
          clean_price: 101.25,
          dirty_price: 102.10,
          yield_to_maturity: 3.75,
          duration: 9.85,
          convexity: 105.2,
          oas: -0.25,
          total_return: 1.9,
          date: '2025-09-13'
        }
      ];

      setBondData(mockData);
    } catch (error) {
      devError('Failed to load bond data:', error);
    } finally {
      setLoading(false);
    }
  };



  const getGovernanceBadge = (status: string) => {
    return status === 'golden' ?
      <Badge className="bg-yellow-500">Golden</Badge> :
      <Badge variant="secondary">Draft</Badge>;
  };

  const getFunctionBadge = (badge: string) => {
    const badgeColors: { [key: string]: string } = {
      '🟩': 'bg-green-500',
      '🟨': 'bg-yellow-500',
      '🟧': 'bg-orange-500',
      '🟦': 'bg-blue-500',
      '🟪': 'bg-purple-500'
    };

    return (
      <Badge className={badgeColors[badge] || 'bg-gray-500'}>
        {badge}
      </Badge>
    );
  };

  if (loading) {
    return <div className="flex justify-center items-center h-64">Loading fixed income analytics...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col space-y-4">
        <h1 className="text-3xl font-bold">Fixed Income Analytics</h1>
        <p className="text-gray-600">
          Comprehensive fixed income metrics and analytics powered by DAX formulas
        </p>

        {/* Controls */}
        <div className="flex space-x-4">
          <Select value={selectedSecurity} onValueChange={setSelectedSecurity}>
            <SelectTrigger className="w-48">
              <SelectValue placeholder="Select Security" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Securities</SelectItem>
              <SelectItem value="US_TREASURY_10Y">US Treasury 10Y</SelectItem>
              <SelectItem value="CORP_BOND_ABC">Corp Bond ABC</SelectItem>
              <SelectItem value="MUNI_BOND_XYZ">Muni Bond XYZ</SelectItem>
            </SelectContent>
          </Select>

          <Select value={timeRange} onValueChange={setTimeRange}>
            <SelectTrigger className="w-32">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="7d">7 Days</SelectItem>
              <SelectItem value="30d">30 Days</SelectItem>
              <SelectItem value="90d">90 Days</SelectItem>
              <SelectItem value="1y">1 Year</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList>
          <TabsTrigger value="overview">Overview</TabsTrigger>
          <TabsTrigger value="pricing">Pricing & Yield</TabsTrigger>
          <TabsTrigger value="risk">Risk Metrics</TabsTrigger>
          <TabsTrigger value="performance">Performance</TabsTrigger>
          <TabsTrigger value="metrics">All Metrics</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Average YTM</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(bondData.reduce((sum, bond) => sum + bond.yield_to_maturity, 0) / bondData.length).toFixed(2)}%
                </div>
                <p className="text-xs text-gray-600">Across all securities</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Average Duration</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(bondData.reduce((sum, bond) => sum + bond.duration, 0) / bondData.length).toFixed(1)} years
                </div>
                <p className="text-xs text-gray-600">Modified duration</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Total Return</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(bondData.reduce((sum, bond) => sum + bond.total_return, 0) / bondData.length).toFixed(1)}%
                </div>
                <p className="text-xs text-gray-600">YTD performance</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">OAS Spread</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {(bondData.reduce((sum, bond) => sum + bond.oas, 0) / bondData.length * 100).toFixed(1)} bps
                </div>
                <p className="text-xs text-gray-600">Option-adjusted spread</p>
              </CardContent>
            </Card>
          </div>

          {/* Yield Curve Chart */}
          <Card>
            <CardHeader>
              <CardTitle>Yield to Maturity by Security</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={bondData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="security_id" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="yield_to_maturity" fill="#8884d8" name="YTM (%)" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="pricing" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Clean vs Dirty Price</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={bondData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="security_id" />
                    <YAxis />
                    <Tooltip />
                    <Legend />
                    <Line type="monotone" dataKey="clean_price" stroke="#8884d8" name="Clean Price" />
                    <Line type="monotone" dataKey="dirty_price" stroke="#82ca9d" name="Dirty Price" />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>Accrued Interest Impact</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  {bondData.map((bond) => (
                    <div key={bond.security_id} className="flex justify-between items-center">
                      <span className="text-sm">{bond.security_id}</span>
                      <span className="font-medium">
                        ${(bond.dirty_price - bond.clean_price).toFixed(2)}
                      </span>
                    </div>
                  ))}
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="risk" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <Card>
              <CardHeader>
                <CardTitle>Duration & Convexity</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={bondData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="security_id" />
                    <YAxis yAxisId="left" />
                    <YAxis yAxisId="right" orientation="right" />
                    <Tooltip />
                    <Legend />
                    <Line yAxisId="left" type="monotone" dataKey="duration" stroke="#8884d8" name="Duration" />
                    <Line yAxisId="right" type="monotone" dataKey="convexity" stroke="#82ca9d" name="Convexity" />
                  </LineChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>OAS Spread Analysis</CardTitle>
              </CardHeader>
              <CardContent>
                <ResponsiveContainer width="100%" height={250}>
                  <BarChart data={bondData}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="security_id" />
                    <YAxis />
                    <Tooltip />
                    <Bar dataKey="oas" fill="#ffc658" name="OAS (bps)" />
                  </BarChart>
                </ResponsiveContainer>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="performance" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Total Return Comparison</CardTitle>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={300}>
                <BarChart data={bondData}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="security_id" />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <Bar dataKey="total_return" fill="#8884d8" name="Total Return (%)" />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="metrics" className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {metrics.map((metric) => (
              <Card key={metric.node_id}>
                <CardHeader className="pb-2">
                  <div className="flex justify-between items-start">
                    <CardTitle className="text-sm">{metric.node_id}</CardTitle>
                    <div className="flex space-x-1">
                      {metric.badge && getFunctionBadge(metric.badge)}
                      {getGovernanceBadge(metric.governance.status)}
                    </div>
                  </div>
                </CardHeader>
                <CardContent>
                  <p className="text-xs text-gray-600 mb-2">{metric.description}</p>
                  <div className="text-xs">
                    <span className="font-medium">Category:</span> {metric.category}
                  </div>
                  {metric.functions_used && metric.functions_used.length > 0 && (
                    <div className="text-xs mt-1">
                      <span className="font-medium">Functions:</span> {metric.functions_used.join(', ')}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
}
