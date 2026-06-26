import React, { useEffect, useRef, useState } from 'react';
import { Box, Typography, Paper, Select, MenuItem, FormControl, InputLabel } from '@mui/material';
import * as d3 from 'd3';
import { sankey, sankeyLinkHorizontal } from 'd3-sankey';

interface WealthTransferFlowProps {
  familyId: string;
}

interface SankeyNode {
  name: string;
  category: string;
}

interface SankeyLink {
  source: number;
  target: number;
  value: number;
  method: string;
}

export const WealthTransferFlow: React.FC<WealthTransferFlowProps> = ({ familyId }) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const [scenario, setScenario] = useState('baseline');

  useEffect(() => {
    if (svgRef.current) {
      renderSankeyDiagram(scenario);
    }
  }, [scenario]);

  const renderSankeyDiagram = (scenarioType: string) => {
    if (!svgRef.current) return;

    // Clear previous diagram
    d3.select(svgRef.current).selectAll('*').remove();

    const width = 1000;
    const height = 600;
    const margin = { top: 20, right: 160, bottom: 20, left: 160 };

    // Define nodes representing generations and destinations
    const nodes: SankeyNode[] = [
      // Generation 1 (Current Wealth Holders)
      { name: 'Patriarch\n$15M', category: 'gen1' },
      { name: 'Matriarch\n$10M', category: 'gen1' },
      
      // Intermediary structures
      { name: 'SLAT\n$8M', category: 'trust' },
      { name: 'Annual Gifts\n$2M', category: 'gift' },
      { name: 'Estate\n$15M', category: 'estate' },
      
      // Generation 2
      { name: 'Child 1\n$7M', category: 'gen2' },
      { name: 'Child 2\n$6M', category: 'gen2' },
      
      // Generation 3
      { name: 'Grandchildren\n$10M', category: 'gen3' },
      
      // Tax impact
      { name: 'Estate Tax\n$3M', category: 'tax' },
    ];

    // Define wealth flow links
    const links: SankeyLink[] = getLinksForScenario(scenarioType);

    // Create Sankey layout
    const sankeyLayout = sankey<SankeyNode, SankeyLink>()
      .nodeWidth(15)
      .nodePadding(30)
      .extent([
        [margin.left, margin.top],
        [width - margin.right, height - margin.bottom],
      ]);

    const { nodes: sankeyNodes, links: sankeyLinks } = sankeyLayout({
      nodes: nodes.map(d => ({ ...d })),
      links: links.map(d => ({ ...d })),
    } as any);

    const svg = d3
      .select(svgRef.current)
      .attr('width', width)
      .attr('height', height)
      .attr('viewBox', [0, 0, width, height]);

    // Color scale
    const color = d3.scaleOrdinal<string>()
      .domain(['gen1', 'gen2', 'gen3', 'trust', 'gift', 'estate', 'tax'])
      .range(['#667eea', '#764ba2', '#f093fb', '#4facfe', '#43e97b', '#ffa726', '#ef5350']);

    // Draw links
    svg.append('g')
      .attr('class', 'links')
      .selectAll('path')
      .data(sankeyLinks)
      .join('path')
      .attr('d', sankeyLinkHorizontal() as any)
      .attr('stroke', d => color((d.source as any).category))
      .attr('stroke-width', d => Math.max(1, (d as any).width))
      .attr('fill', 'none')
      .attr('opacity', 0.5)
      .append('title')
      .text(d => `${(d.source as any).name} → ${(d.target as any).name}\n${formatCurrency(d.value)}`);

    // Draw nodes
    const nodeGroup = svg.append('g')
      .attr('class', 'nodes')
      .selectAll('g')
      .data(sankeyNodes)
      .join('g');

    nodeGroup
      .append('rect')
      .attr('x', d => (d as any).x0)
      .attr('y', d => (d as any).y0)
      .attr('height', d => (d as any).y1 - (d as any).y0)
      .attr('width', d => (d as any).x1 - (d as any).x0)
      .attr('fill', d => color(d.category))
      .attr('stroke', '#333')
      .attr('stroke-width', 1)
      .append('title')
      .text(d => `${d.name}\n${formatCurrency(d.value || 0)}`);

    // Add  labels
    nodeGroup
      .append('text')
      .attr('x', d => ((d as any).x0 < width / 2 ? (d as any).x1 + 6 : (d as any).x0 - 6))
      .attr('y', d => ((d as any).y1 + (d as any).y0) / 2)
      .attr('dy', '0.35em')
      .attr('text-anchor', d => ((d as any).x0 < width / 2 ? 'start' : 'end'))
      .style('font-size', '12px')
      .style('font-weight', 'bold')
      .text(d => d.name);
  };

  const getLinksForScenario = (scenarioType: string): SankeyLink[] => {
    if (scenarioType === 'baseline') {
      // No planning scenario - most wealth goes through estate
      return [
        { source: 0, target: 4, value: 15000000, method: 'inheritance' }, // Patriarch -> Estate
        { source: 1, target: 4, value: 10000000, method: 'inheritance' }, // Matriarch -> Estate
        { source: 4, target: 5, value: 11000000, method: 'inheritance' }, // Estate -> Child 1
        { source: 4, target: 6, value: 11000000, method: 'inheritance' }, // Estate -> Child 2
        { source: 4, target: 8, value: 3000000, method: 'tax' }, // Estate -> Tax
      ];
    } else if (scenarioType === 'gifting') {
      // Annual gifting strategy
      return [
        { source: 0, target: 3, value: 2000000, method: 'annual_gift' }, // Patriarch -> Annual Gifts
        { source: 1, target: 3, value: 2000000, method: 'annual_gift' }, // Matriarch -> Annual Gifts
        { source: 0, target: 4, value: 13000000, method: 'inheritance' }, // Patriarch -> Estate
        { source: 1, target: 4, value: 8000000, method: 'inheritance' }, // Matriarch -> Estate
        { source: 3, target: 5, value: 2000000, method: 'gift' }, // Gifts -> Child 1
        { source: 3, target: 6, value: 2000000, method: 'gift' }, // Gifts -> Child 2
        { source: 4, target: 5, value: 9000000, method: 'inheritance' }, // Estate -> Child 1
        { source: 4, target: 6, value: 9000000, method: 'inheritance' }, // Estate -> Child 2
        { source: 4, target: 8, value: 3000000, method: 'tax' }, // Estate -> Tax
      ];
    } else {
      // SLAT + Gifting comprehensive strategy
      return [
        { source: 0, target: 2, value: 8000000, method: 'slat_funding' }, // Patriarch -> SLAT
        { source: 1, target: 3, value: 2000000, method: 'annual_gift' }, // Matriarch -> Annual Gifts
        { source: 0, target: 4, value: 7000000, method: 'inheritance' }, // Patriarch -> Estate
        { source: 1, target: 4, value: 8000000, method: 'inheritance' }, // Matriarch -> Estate
        { source: 2, target: 5, value: 4000000, method: 'trust_distribution' }, // SLAT -> Child 1
        { source: 2, target: 6, value: 4000000, method: 'trust_distribution' }, // SLAT -> Child 2
        { source: 3, target: 7, value: 2000000, method: 'gift' }, // Gifts -> Grandchildren
        { source: 4, target: 5, value: 6000000, method: 'inheritance' }, // Estate -> Child 1
        { source: 4, target: 6, value: 6000000, method: 'inheritance' }, // Estate -> Child 2
        { source: 4, target: 8, value: 3000000, method: 'tax' }, // Estate -> Tax
        { source: 5, target: 7, value: 3000000, method: 'inheritance' }, // Child 1 -> Grandchildren
        { source: 6, target: 7, value: 3000000, method: 'inheritance' }, // Child 2 -> Grandchildren
      ];
    }
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 1,
      notation: 'compact',
      compactDisplay: 'short',
    }).format(value);
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', gap: 2, mb: 3, alignItems: 'center' }}>
        <Typography variant="h6">Wealth Transfer Flow Visualization</Typography>
        <Box sx={{ flexGrow: 1 }} />
        <FormControl sx={{ minWidth: 200 }}>
          <InputLabel>Scenario</InputLabel>
          <Select
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            label="Scenario"
          >
            <MenuItem value="baseline">Baseline (No Planning)</MenuItem>
            <MenuItem value="gifting">Annual Gifting Strategy</MenuItem>
            <MenuItem value="comprehensive">SLAT + Gifting (Comprehensive)</MenuItem>
          </Select>
        </FormControl>
      </Box>

      <Paper elevation={2} sx={{ p: 3 }}>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          This Sankey diagram visualizes how wealth flows across generations and through various estate planning strategies.
          Hover over nodes and links to see details.
        </Typography>

        <Box sx={{ mt: 2, overflow: 'auto' }}>
          <svg ref={svgRef}></svg>
        </Box>

        {/* Legend */}
        <Box sx={{ mt: 3, display: 'flex', gap: 3, flexWrap: 'wrap' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#667eea', borderRadius: 1 }} />
            <Typography variant="caption">Generation 1 (Current)</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#764ba2', borderRadius: 1 }} />
            <Typography variant="caption">Generation 2 (Children)</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#f093fb', borderRadius: 1 }} />
            <Typography variant="caption">Generation 3 (Grandchildren)</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#4facfe', borderRadius: 1 }} />
            <Typography variant="caption">Trusts</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#43e97b', borderRadius: 1 }} />
            <Typography variant="caption">Gifts</Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Box sx={{ width: 20, height: 20, bgcolor: '#ef5350', borderRadius: 1 }} />
            <Typography variant="caption">Estate Tax</Typography>
          </Box>
        </Box>
      </Paper>
    </Box>
  );
};
