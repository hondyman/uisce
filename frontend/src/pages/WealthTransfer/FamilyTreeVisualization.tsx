import React, { useEffect, useRef, useState } from 'react';
import { Box, Typography, Paper, Tooltip, IconButton, Chip } from '@mui/material';
import { ZoomIn, ZoomOut, CenterFocusStrong } from '@mui/icons-material';
import * as d3 from 'd3';

interface FamilyMember {
  member_id: string;
  legal_first_name: string;
  legal_last_name: string;
  date_of_birth: string;
  generation: number;
  spouse_member_id: string | null;
  children_member_ids: string[];
  separate_networth: number;
  engagement_score?: number;
}

interface FamilyTreeVisualizationProps {
  familyId: string;
}

export const FamilyTreeVisualization: React.FC<FamilyTreeVisualizationProps> = ({ familyId }) => {
  const svgRef = useRef<SVGSVGElement>(null);
  const [members, setMembers] = useState<FamilyMember[]>([]);
  const [selectedMember, setSelectedMember] = useState<FamilyMember | null>(null);
  const [zoom, setZoom] = useState(1);

  useEffect(() => {
    // Fetch family members
    fetch(`/api/wealth-transfer/families/${familyId}/members`)
      .then(res => res.json())
      .then(data => {
        setMembers(data);
        renderTree(data);
      })
      .catch(err => console.error('Failed to load family members:', err));
  }, [familyId]);

  const renderTree = (data: FamilyMember[]) => {
    if (!svgRef.current || data.length === 0) return;

    const svg = d3.select(svgRef.current);
    svg.selectAll('*').remove();

    const width = 1200;
    const height = 800;
    const margin = { top: 40, right: 120, bottom: 40, left: 120 };

    // Build hierarchy
    const root = buildHierarchy(data);
    const treeLayout = d3.tree<FamilyMember>().size([height - margin.top - margin.bottom, width - margin.left - margin.right]);
    const treeData = treeLayout(d3.hierarchy(root));

    const g = svg
      .attr('width', width)
      .attr('height', height)
      .append('g')
      .attr('transform', `translate(${margin.left},${margin.top})`);

    // Links
    g.selectAll('.link')
      .data(treeData.links())
      .enter()
      .append('path')
      .attr('class', 'link')
      .attr('d', d3.linkHorizontal<any, any>()
        .x(d => d.y)
        .y(d => d.x)
      )
      .style('fill', 'none')
      .style('stroke', '#ccc')
      .style('stroke-width', 2);

    // Nodes
    const node = g.selectAll('.node')
      .data(treeData.descendants())
      .enter()
      .append('g')
      .attr('class', 'node')
      .attr('transform', d => `translate(${d.y},${d.x})`)
      .style('cursor', 'pointer')
      .on('click', (event, d) => {
        setSelectedMember(d.data as any);
      });

    // Node circles
    node.append('circle')
      .attr('r', 25)
      .style('fill', d => getGenerationColor((d.data as any).generation))
      .style('stroke', '#333')
      .style('stroke-width', 2)
      .style('opacity', 0.9);

    // Node labels
    node.append('text')
      .attr('dy', -35)
      .attr('text-anchor', 'middle')
      .style('font-size', '12px')
      .style('font-weight', 'bold')
      .text(d => {
        const member = d.data as any;
        return `${member.legal_first_name} ${member.legal_last_name}`;
      });

    // Generation label
    node.append('text')
      .attr('dy', 40)
      .attr('text-anchor', 'middle')
      .style('font-size', '10px')
      .style('fill', '#666')
      .text(d => `Gen ${(d.data as any).generation}`);

    // Net worth label
    node.append('text')
      .attr('dy', 52)
      .attr('text-anchor', 'middle')
      .style('font-size', '9px')
      .style('fill', '#999')
      .text(d => {
        const member = d.data as any;
        const networth = (member.separate_networth / 1000000).toFixed(1);
        return `$${networth}M`;
      });
  };

  const buildHierarchy = (members: FamilyMember[]): any => {
    // Find root (generation 1, oldest)
    const gen1 = members.filter(m => m.generation === 1);
    if (gen1.length === 0) return { member_id: 'root', legal_first_name: 'Family', legal_last_name: '', children: [] };

    const root = gen1[0];
    
    const buildNode = (member: FamilyMember): any => {
      const children = member.children_member_ids
        .map(childId => members.find(m => m.member_id === childId))
        .filter(Boolean)
        .map(child => buildNode(child!));

      return {
        ...member,
        children: children.length > 0 ? children : undefined,
      };
    };

    return buildNode(root);
  };

  const getGenerationColor = (gen: number): string => {
    const colors = [
      '#667eea', // Gen 1 - Purple
      '#764ba2', // Gen 2 - Deep Purple
      '#f093fb', // Gen 3 - Pink
      '#4facfe', // Gen 4 - Blue
    ];
    return colors[gen - 1] || '#ccc';
  };

  const handleZoomIn = () => setZoom(z => Math.min(z + 0.2, 3));
  const handleZoomOut = () => setZoom(z => Math.max(z - 0.2, 0.5));
  const handleResetZoom = () => setZoom(1);

  return (
    <Box>
      <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
        <Typography variant="h6">Family Tree</Typography>
        <Box sx={{ flexGrow: 1 }} />
        <IconButton onClick={handleZoomOut} size="small">
          <ZoomOut />
        </IconButton>
        <Typography variant="body2">{Math.round(zoom * 100)}%</Typography>
        <IconButton onClick={handleZoomIn} size="small">
          <ZoomIn />
        </IconButton>
        <IconButton onClick={handleResetZoom} size="small">
          <CenterFocusStrong />
        </IconButton>
      </Box>

      <Paper elevation={2} sx={{ p: 2, overflow: 'auto', height: 600 }}>
        <Box sx={{ transform: `scale(${zoom})`, transformOrigin: 'top left' }}>
          <svg ref={svgRef}></svg>
        </Box>
      </Paper>

      {/* Selected Member Details */}
      {selectedMember && (
        <Paper elevation={3} sx={{ mt: 2, p: 3 }}>
          <Typography variant="h6" gutterBottom>
            {selectedMember.legal_first_name} {selectedMember.legal_last_name}
          </Typography>
          <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
            <Chip label={`Generation ${selectedMember.generation}`} color="primary" size="small" />
            <Chip label={`Age ${calculateAge(selectedMember.date_of_birth)}`} size="small" />
            {selectedMember.engagement_score && (
              <Chip 
                label={`Engagement: ${Math.round(selectedMember.engagement_score * 100)}%`} 
                color={selectedMember.engagement_score > 0.7 ? 'success' : 'default'}
                size="small" 
              />
            )}
          </Box>
          <Typography variant="body2" color="text.secondary">
            <strong>Net Worth:</strong> ${(selectedMember.separate_networth / 1000000).toFixed(2)}M
          </Typography>
          <Typography variant="body2" color="text.secondary">
            <strong>Children:</strong> {selectedMember.children_member_ids.length}
          </Typography>
        </Paper>
      )}

      {/* Legend */}
      <Box sx={{ mt: 2, display: 'flex', gap: 2 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ width: 16, height: 16, borderRadius: '50%', bgcolor: '#667eea' }} />
          <Typography variant="caption">Generation 1</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ width: 16, height: 16, borderRadius: '50%', bgcolor: '#764ba2' }} />
          <Typography variant="caption">Generation 2</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ width: 16, height: 16, borderRadius: '50%', bgcolor: '#f093fb' }} />
          <Typography variant="caption">Generation 3</Typography>
        </Box>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ width: 16, height: 16, borderRadius: '50%', bgcolor: '#4facfe' }} />
          <Typography variant="caption">Generation 4</Typography>
        </Box>
      </Box>
    </Box>
  );
};

const calculateAge = (dob: string): number => {
  const birthDate = new Date(dob);
  const today = new Date();
  let age = today.getFullYear() - birthDate.getFullYear();
  const monthDiff = today.getMonth() - birthDate.getMonth();
  if (monthDiff < 0 || (monthDiff === 0 && today.getDate() < birthDate.getDate())) {
    age--;
  }
  return age;
};
