import React, { useEffect, useMemo, useState } from "react";
import axios from "axios";
import {
  ScatterChart, Scatter, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer,
  BarChart, Bar, Cell, Brush
} from "recharts";
import { Line } from "react-chartjs-2";
import { computeHistogram, formatBinLabel, type Bin } from "../utils/histogram";
import {
  Box,
  Typography,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
  Alert,
  Card,
  CardContent,
  Button,
  TextField,
  Divider,
  FormControlLabel,
  Switch,
  Checkbox,
  FormGroup,
  FormControl,
  FormLabel
} from "@mui/material";
import {
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip as ChartJSTooltip,
  Legend as ChartJSLegend,
} from 'chart.js';
import Chart from 'chart.js/auto';
import styles from './FrontierExplorer.module.css';

// Register Chart.js components
Chart.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  ChartJSTooltip,
  ChartJSLegend
);

// Consistent asset color palette
const assetColors = ["#4E79A7", "#F28E2B", "#E15759", "#76B7B2", "#59A14F", "#EDC948", "#B07AA1", "#FF9DA7", "#9C755F", "#BAB0AC"];

type FrontierPoint = {
  weights: number[];
  exp_return: number;
  volatility: number;
  variance?: number;
  sharpe?: number;
};

type Tangency = FrontierPoint;

const assetNamesDefault = ["Asset A", "Asset B", "Asset C"];

function WeightsTooltip({ active, payload, label: _label, assetNames = assetNamesDefault, visibleAssets = [], hoveredAsset = null }: any) {
  if (!active || !payload || !payload.length) return null;
  const p = payload[0]?.payload as FrontierPoint;
  return (
    <div className={styles.tooltipContainer}>
      <div className={styles.tooltipHeader}><strong>Portfolio Details</strong></div>
      <div><strong>Volatility:</strong> {p.volatility.toFixed(4)}</div>
      <div><strong>Expected Return:</strong> {p.exp_return.toFixed(4)}</div>
      {typeof p.sharpe === "number" && (
        <div><strong>Sharpe Ratio:</strong> {p.sharpe?.toFixed(3)}</div>
      )}
      <Divider className={styles.tooltipDivider} />
      <div className={styles.weightsHeader}><strong>Weights:</strong></div>
      <div className={styles.weightsContainer}>
        {p.weights.map((w, i) => (
          visibleAssets[i] && (
            <div key={i} className={`${styles.weightItem} ${visibleAssets[i] ? styles.weightItemVisible : ''} ${hoveredAsset === i ? styles.weightItemHovered : ''}`}
              style={{ color: assetColors[i % assetColors.length] }}>
              {assetNames[i] || `Asset ${i + 1}`}: {(w * 100).toFixed(1)}%
            </div>
          )
        ))}
      </div>
    </div>
  );
}

function BinTooltip({ active, payload }: any) {
  if (!active || !payload || !payload.length) return null;
  const d = payload[0]?.payload as Bin & { label: string };
  return (
    <div className={styles.binTooltipContainer}>
      <div><strong>Range:</strong> {d.label}</div>
      <div><strong>Count:</strong> {d.count}</div>
    </div>
  );
}

interface FrontierExplorerProps {
  assetNames?: string[];
  benchmark?: number[];
}

export default function FrontierExplorer({
  assetNames = assetNamesDefault,
  benchmark = [0.33, 0.33, 0.34]
}: FrontierExplorerProps) {
  // Frontier and tangency
  const [frontier, setFrontier] = useState<FrontierPoint[]>([]);
  const [tangency, setTangency] = useState<Tangency | null>(null);

  // Selection and metrics
  const [selected, setSelected] = useState<FrontierPoint | null>(null);
  const [trackingError, setTrackingError] = useState<number | null>(null);
  const [infoRatio, setInfoRatio] = useState<number | null>(null);

  // Paths
  const [gbmPath, setGbmPath] = useState<number[]>([]);
  const [ouPath, setOuPath] = useState<number[]>([]);

  // Constraint toggles
  const [longOnly, setLongOnly] = useState(true);
  const [maxWeight, setMaxWeight] = useState<number | null>(null);

  // Asset visibility toggles
  const [visibleAssets, setVisibleAssets] = useState<boolean[]>(assetNames.map(() => true));
  const [hoveredAsset, setHoveredAsset] = useState<number | null>(null);

  // Monte Carlo samples and histogram
  const [mcSamples, setMcSamples] = useState<number[]>([]);
  const [mcBins, setMcBins] = useState(40);
  const [isRunningMC, setIsRunningMC] = useState(false);

  // Loading states
  const [loading, setLoading] = useState<{[key: string]: boolean}>({});
  const [error, setError] = useState<string | null>(null);

  // Demo input set (can be made configurable)
  const mu = React.useMemo(() => [0.08, 0.12, 0.10], []);
  const covariance = React.useMemo(() => [
    [0.04, 0.006, 0.004],
    [0.006, 0.09, 0.008],
    [0.004, 0.008, 0.0625]
  ], []);
  const riskFree = 0.02;

  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000';

  const makeApiCall = React.useCallback(async (endpoint: string, data: any, key: string) => {
    setLoading(prev => ({ ...prev, [key]: true }));
    setError(null);

    // Check cache first
    const cacheKey = `${key}_${JSON.stringify(data)}`;
    const cached = localStorage.getItem(cacheKey);
    if (cached) {
      const { data: cachedData, timestamp } = JSON.parse(cached);
      if (Date.now() - timestamp < 5 * 60 * 1000) { // 5 minutes cache
        setLoading(prev => ({ ...prev, [key]: false }));
        return cachedData;
      }
    }

  try {
      const response = await axios.post(`${apiBaseUrl}${endpoint}`, data);
      // Cache the response
      localStorage.setItem(cacheKey, JSON.stringify({
        data: response.data,
        timestamp: Date.now()
      }));
      return response.data;
    } catch (err: any) {
      setError(`Error fetching ${key}: ${err.message}`);
      return null;
    } finally {
      setLoading(prev => ({ ...prev, [key]: false }));
    }
  }, [apiBaseUrl]);

  useEffect(() => {
    const fetchData = async () => {
      // Efficient frontier
      const frontierData = await makeApiCall("/api/calc/run", {
        node_id: "frontier-explorer",
        node_type: "calculation",
        domain: "finance",
        category: "portfolio",
        subcategory: "optimization",
        version: "1.0",
        owner: "system",
        description: "Efficient frontier calculation",
        financial_calc: {
          type: "efficient_frontier",
          mu, covariance,
          points: 50, long_only: longOnly,
          max_weight: maxWeight,
          risk_free_rate: riskFree
        }
  }, "frontier");

      if (frontierData && frontierData.result) {
        const pts = Array.isArray(frontierData.result) ? frontierData.result : [];
        setFrontier(pts);
        // Auto-select the point with highest Sharpe ratio
        const best = [...pts].sort((a, b) => (b.sharpe || 0) - (a.sharpe || 0))[0];
        if (best) setSelected(best);
      }

      // Tangency
      const tangencyData = await makeApiCall("/api/calc/run", {
        node_id: "tangency-calc",
        node_type: "calculation",
        domain: "finance",
        category: "portfolio",
        subcategory: "optimization",
        version: "1.0",
        owner: "system",
        description: "Tangency portfolio calculation",
        financial_calc: {
          type: "tangency",
          mu, covariance,
          long_only: longOnly,
          max_weight: maxWeight,
          risk_free_rate: riskFree
        }
      }, "tangency");

      if (tangencyData && tangencyData.result) {
        setTangency(tangencyData.result);
      }

      // GBM path
      const gbmData = await makeApiCall("/api/calc/run", {
        node_id: "gbm-simulation",
        node_type: "calculation",
        domain: "finance",
        category: "stochastic",
        subcategory: "modeling",
        version: "1.0",
        owner: "system",
        description: "GBM path simulation",
        financial_calc: {
          type: "gbm",
          S0: [100],
          mu: [0.05],
          sigma: [0.2],
          T: 1.0,
          steps: 252,
          seed: 42
        }
      }, "gbm");

      if (gbmData && gbmData.result && gbmData.result.paths) {
        const paths = gbmData.result.paths;
        setGbmPath(paths.map((row: number[]) => row[0]));
      }

      // OU path
      const ouData = await makeApiCall("/api/calc/run", {
        node_id: "ou-simulation",
        node_type: "calculation",
        domain: "finance",
        category: "stochastic",
        subcategory: "modeling",
        version: "1.0",
        owner: "system",
        description: "OU process simulation",
        financial_calc: {
          type: "ou",
          x0: 0.03,
          theta: 1.5,
          mean: 0.05,
          sigma: 0.02,
          T: 1.0,
          steps: 252
        }
      }, "ou");

      if (ouData && ouData.result && ouData.result.path) {
        setOuPath(ouData.result.path);
      }
    };

    fetchData();
  }, [longOnly, maxWeight, makeApiCall, mu, covariance, riskFree]);

  // Update TE/IR when selection changes
  useEffect(() => {
    if (!selected) return;

    const updateMetrics = async () => {
      // Tracking Error
      const teData = await makeApiCall("/api/calc/run", {
        node_id: "tracking-error-calc",
        node_type: "calculation",
        domain: "finance",
        category: "portfolio",
        subcategory: "risk",
        version: "1.0",
        owner: "system",
        description: "Tracking error calculation",
        financial_calc: {
          type: "tracking_error",
          mu, covariance,
          weights: selected.weights,
          benchmark_weights: benchmark
        }
      }, "tracking_error");

      if (teData && teData.result) {
        setTrackingError(teData.result.tracking_error || teData.result);
      }

      // Information Ratio
      const irData = await makeApiCall("/api/calc/run", {
        node_id: "information-ratio-calc",
        node_type: "calculation",
        domain: "finance",
        category: "portfolio",
        subcategory: "risk",
        version: "1.0",
        owner: "system",
        description: "Information ratio calculation",
        financial_calc: {
          type: "information_ratio",
          mu, covariance,
          weights: selected.weights,
          benchmark_weights: benchmark
        }
      }, "information_ratio");

      if (irData && irData.result) {
        setInfoRatio(irData.result.information_ratio || irData.result);
      }
    };

    updateMetrics();
  }, [selected, selected?.weights, benchmark, makeApiCall, mu, covariance]);

  // Request Monte Carlo samples
  const fetchMonteCarlo = async () => {
    setIsRunningMC(true);
    try {
      const mcData = await makeApiCall("/api/calc/run", {
        node_id: "monte-carlo-simulation",
        node_type: "calculation",
        domain: "finance",
        category: "stochastic",
        subcategory: "simulation",
        version: "1.0",
        owner: "system",
        description: "Monte Carlo simulation",
        financial_calc: {
          type: "monte_carlo",
          sims: 20000,
          S0: 100,
          mu: 0.05,
          sigma: 0.2,
          strike: 100,
          T: 1.0,
          r: 0.02,
          return_samples: true
        }
      }, "monte_carlo");

      if (mcData && mcData.result && mcData.result.samples) {
        setMcSamples(mcData.result.samples);
      }
    } finally {
      setIsRunningMC(false);
    }
  };

  // Prepare histogram data with labels
  const hist = useMemo(() => {
    const bins = computeHistogram(mcSamples, mcBins);
    return bins.map(b => ({ ...b, label: formatBinLabel(b, 3) }));
  }, [mcSamples, mcBins]);

  return (
    <Box sx={{ padding: 3 }}>
      <Typography variant="h4" gutterBottom>
        🎨 Frontier Explorer
      </Typography>
      <Typography variant="body1" color="text.secondary" gutterBottom>
        Interactive portfolio analytics and stochastic simulations
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Efficient Frontier Chart */}
        <Grid item xs={12} lg={8}>
          <Card>
            <CardContent>
              <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <Typography variant="h6">
                  Efficient Frontier
                </Typography>
                <Box display="flex" gap={2} alignItems="center">
                  <FormControlLabel
                    control={
                      <Switch
                        checked={longOnly}
                        onChange={(e) => setLongOnly(e.target.checked)}
                        size="small"
                      />
                    }
                    label="Long Only"
                  />
                  <TextField
                    label="Max Weight"
                    type="number"
                    size="small"
                    value={maxWeight || ''}
                    onChange={(e) => setMaxWeight(e.target.value ? parseFloat(e.target.value) : null)}
                    inputProps={{ min: 0, max: 1, step: 0.1 }}
                    sx={{ width: 100 }}
                    placeholder="None"
                  />
                  <Button
                    variant="outlined"
                    size="small"
                    onClick={() => {
                      // Reset zoom by re-rendering the chart
                      setFrontier([...frontier]);
                    }}
                  >
                    Reset Zoom
                  </Button>
                </Box>
              </Box>

              {/* Asset Legend with Checkboxes */}
              <Box mb={2}>
                <FormControl component="fieldset">
                  <FormLabel component="legend" sx={{ mb: 1 }}>Asset Visibility</FormLabel>
                  <FormGroup row>
                    {assetNames.map((name, i) => (
                      <FormControlLabel
                        key={i}
                        control={
                          <Checkbox
                            checked={visibleAssets[i]}
                            onChange={(e) => {
                              const newVisible = [...visibleAssets];
                              newVisible[i] = e.target.checked;
                              setVisibleAssets(newVisible);
                            }}
                            sx={{
                              color: assetColors[i % assetColors.length],
                              '&.Mui-checked': {
                                color: assetColors[i % assetColors.length],
                              },
                            }}
                          />
                        }
                        label={
                          <Box display="flex" alignItems="center">
                            <Box
                              sx={{
                                width: 12,
                                height: 12,
                                backgroundColor: assetColors[i % assetColors.length],
                                borderRadius: 1,
                                mr: 1,
                                opacity: visibleAssets[i] ? 1 : 0.3
                              }}
                            />
                            {name}
                          </Box>
                        }
                      />
                    ))}
                  </FormGroup>
                </FormControl>
              </Box>

              <Typography variant="body2" color="text.secondary" gutterBottom>
                Click on any point to view portfolio weights and risk metrics
              </Typography>
              {loading.frontier ? (
                <Box display="flex" justifyContent="center" p={4}>
                  <CircularProgress />
                </Box>
              ) : (
                <ResponsiveContainer width="100%" height={400}>
                  <ScatterChart
                    onClick={(e: any) => {
                      const payload = e?.activePayload?.[0]?.payload as FrontierPoint;
                      if (payload?.weights) setSelected(payload);
                    }}
                    onMouseMove={(e: any) => {
                      const payload = e?.activePayload?.[0]?.payload as FrontierPoint;
                      if (payload?.weights) {
                        // Find which asset is being hovered based on weights
                        const maxWeightIndex = payload.weights.indexOf(Math.max(...payload.weights));
                        setHoveredAsset(maxWeightIndex);
                      }
                    }}
                    onMouseLeave={() => setHoveredAsset(null)}
                  >
                    <CartesianGrid />
                    <XAxis
                      type="number"
                      dataKey="volatility"
                      name="Volatility"
                      domain={['dataMin', 'dataMax']}
                    />
                    <YAxis
                      type="number"
                      dataKey="exp_return"
                      name="Expected Return"
                      domain={['dataMin', 'dataMax']}
                    />
                    <Tooltip content={<WeightsTooltip assetNames={assetNames} visibleAssets={visibleAssets} hoveredAsset={hoveredAsset} />} />
                    <Legend />
                    <Scatter
                      name="Efficient Frontier"
                      data={frontier}
                      fill="#4E79A7"
                      fillOpacity={0.8}
                      shape={(props: any) => {
                        const { payload } = props;
                        const sharpe = payload?.sharpe || 0;
                        const color = sharpe > 1.5 ? '#76B7B2' : sharpe > 1.0 ? '#4E79A7' : '#E15759';
                        return (
                          <circle
                            cx={props.cx}
                            cy={props.cy}
                            r={4}
                            fill={color}
                            fillOpacity={0.8}
                            stroke="#fff"
                            strokeWidth={1}
                          />
                        );
                      }}
                    />
                    {tangency && (
                      <Scatter
                        name="Tangency Portfolio"
                        data={[tangency]}
                        fill="#E15759"
                        shape="star"
                        r={8}
                      />
                    )}
                    {selected && (
                      <Scatter
                        name="Selected Portfolio"
                        data={[selected]}
                        fill="#76B7B2"
                        shape="diamond"
                        r={6}
                      />
                    )}
                    <Brush dataKey="volatility" height={30} stroke="#8884d8" />
                  </ScatterChart>
                </ResponsiveContainer>
              )}
              {selected && (
                <Box
                  sx={{
                    mt: 2,
                    p: 2,
                    bgcolor: 'grey.50',
                    borderRadius: 1,
                    transition: 'all 0.3s ease-in-out',
                    transform: selected ? 'scale(1)' : 'scale(0.95)',
                    opacity: selected ? 1 : 0.8
                  }}
                >
                  <Typography variant="subtitle2" gutterBottom>
                    Selected Portfolio Details
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="body2">
                        <strong>Volatility:</strong> {selected.volatility.toFixed(4)}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="body2">
                        <strong>Return:</strong> {selected.exp_return.toFixed(4)}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="body2">
                        <strong>Sharpe:</strong> {selected.sharpe?.toFixed(3) || 'N/A'}
                      </Typography>
                    </Grid>
                    <Grid item xs={6} sm={3}>
                      <Typography variant="body2">
                        <strong>Assets:</strong> {selected.weights.length}
                      </Typography>
                    </Grid>
                  </Grid>
                  <Box sx={{ mt: 2 }}>
                    <Typography variant="body2" gutterBottom>
                      <strong>Portfolio Weights:</strong>
                    </Typography>
                    <ResponsiveContainer width="100%" height={100}>
                      <BarChart
                        data={selected.weights.map((w, i) => ({
                          asset: assetNames[i] || `Asset ${i + 1}`,
                          weight: w * 100,
                          color: assetColors[i % assetColors.length],
                          visible: visibleAssets[i]
                        }))}
                        layout="horizontal"
                        onMouseMove={(e: any) => {
                          if (e?.activeLabel) {
                            const assetIndex = assetNames.findIndex(name => name === e.activeLabel);
                            if (assetIndex !== -1) {
                              setHoveredAsset(assetIndex);
                            }
                          }
                        }}
                        onMouseLeave={() => setHoveredAsset(null)}
                      >
                        <XAxis type="number" domain={[0, 100]} />
                        <YAxis dataKey="asset" type="category" width={60} />
                        <Tooltip formatter={(value: number) => [`${value.toFixed(1)}%`, 'Weight']} />
                        <Bar dataKey="weight" fill="#76B7B2">
                          {selected.weights.map((_, i) => (
                            <Cell
                              key={`cell-${i}`}
                              fill={assetColors[i % assetColors.length]}
                              opacity={visibleAssets[i] ? 1 : 0.3}
                              stroke={hoveredAsset === i ? "#000" : "none"}
                              strokeWidth={hoveredAsset === i ? 2 : 0}
                            />
                          ))}
                        </Bar>
                      </BarChart>
                    </ResponsiveContainer>
                  </Box>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Risk Metrics Table */}
        <Grid item xs={12} lg={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Risk Metrics
              </Typography>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Metric</TableCell>
                      <TableCell align="right">Value</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    <TableRow>
                      <TableCell>Tracking Error</TableCell>
                      <TableCell align="right">
                        {loading.tracking_error ? (
                          <CircularProgress size={16} />
                        ) : (
                          trackingError?.toFixed(4) || "N/A"
                        )}
                      </TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Information Ratio</TableCell>
                      <TableCell align="right">
                        {loading.information_ratio ? (
                          <CircularProgress size={16} />
                        ) : (
                          infoRatio?.toFixed(4) || "N/A"
                        )}
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </TableContainer>
            </CardContent>
          </Card>
        </Grid>

        {/* Stochastic Paths */}
        <Grid item xs={12} lg={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Geometric Brownian Motion (GBM)
              </Typography>
              {loading.gbm ? (
                <Box display="flex" justifyContent="center" p={4}>
                  <CircularProgress />
                </Box>
              ) : gbmPath.length > 0 ? (
                <Box sx={{ height: 300 }}>
                  <Line
                    data={{
                      labels: gbmPath.map((_, i) => i.toString()),
                      datasets: [{
                        label: "GBM Path",
                        data: gbmPath,
                        borderColor: "rgb(78, 121, 167)",
                        backgroundColor: "rgba(78, 121, 167, 0.1)",
                        fill: false,
                        pointRadius: 0,
                        borderWidth: 2
                      }]
                    }}
                    options={{
                      responsive: true,
                      maintainAspectRatio: false,
                      plugins: {
                        legend: { display: true },
                        tooltip: {
                          mode: 'index',
                          intersect: false,
                        }
                      },
                      scales: {
                        x: {
                          display: true,
                          title: {
                            display: true,
                            text: 'Time Steps'
                          }
                        },
                        y: {
                          display: true,
                          title: {
                            display: true,
                            text: 'Price'
                          }
                        }
                      }
                    }}
                  />
                </Box>
              ) : (
                <Typography color="text.secondary">No GBM data available</Typography>
              )}
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} lg={6}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                Ornstein-Uhlenbeck (OU) Process
              </Typography>
              {loading.ou ? (
                <Box display="flex" justifyContent="center" p={4}>
                  <CircularProgress />
                </Box>
              ) : ouPath.length > 0 ? (
                <Box sx={{ height: 300 }}>
                  <Line
                    data={{
                      labels: ouPath.map((_, i) => i.toString()),
                      datasets: [{
                        label: "OU Path",
                        data: ouPath,
                        borderColor: "rgb(89, 161, 79)",
                        backgroundColor: "rgba(89, 161, 79, 0.1)",
                        fill: false,
                        pointRadius: 0,
                        borderWidth: 2
                      }]
                    }}
                    options={{
                      responsive: true,
                      maintainAspectRatio: false,
                      plugins: {
                        legend: { display: true },
                        tooltip: {
                          mode: 'index',
                          intersect: false,
                        }
                      },
                      scales: {
                        x: {
                          display: true,
                          title: {
                            display: true,
                            text: 'Time Steps'
                          }
                        },
                        y: {
                          display: true,
                          title: {
                            display: true,
                            text: 'Value'
                          }
                        }
                      }
                    }}
                  />
                </Box>
              ) : (
                <Typography color="text.secondary">No OU data available</Typography>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Monte Carlo Histogram */}
        <Grid item xs={12}>
          <Card>
            <CardContent>
              <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <Typography variant="h6">
                  Monte Carlo Distribution
                </Typography>
                <Box display="flex" gap={2} alignItems="center">
                  <Button
                    variant="contained"
                    onClick={fetchMonteCarlo}
                    disabled={isRunningMC}
                    size="small"
                  >
                    {isRunningMC ? <CircularProgress size={16} /> : "Run 20k Simulations"}
                  </Button>
                  <TextField
                    label="Bins"
                    type="number"
                    size="small"
                    value={mcBins}
                    onChange={(e) => setMcBins(parseInt(e.target.value) || 40)}
                    inputProps={{ min: 10, max: 200 }}
                    sx={{ width: 80 }}
                  />
                </Box>
              </Box>
              {loading.monte_carlo ? (
                <Box display="flex" justifyContent="center" p={4}>
                  <CircularProgress />
                </Box>
              ) : hist.length > 0 ? (
                <ResponsiveContainer width="100%" height={320}>
                  <BarChart data={hist}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="label"
                      angle={-45}
                      textAnchor="end"
                      height={80}
                      interval={0}
                    />
                    <YAxis />
                    <Tooltip content={<BinTooltip />} />
                    <Bar dataKey="count" fill="#76B7B2">
                      {hist.map((_, i) => (
                        <Cell
                          key={`cell-${i}`}
                          fill={`hsl(${200 + (i / hist.length) * 60}, 70%, 50%)`}
                        />
                      ))}
                    </Bar>
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <Box display="flex" justifyContent="center" p={4}>
                  <Typography color="text.secondary">
                    Click "Run Simulations" to generate Monte Carlo distribution
                  </Typography>
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
}
