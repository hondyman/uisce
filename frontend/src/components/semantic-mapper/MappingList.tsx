import { Box, Card, Typography, Stack, Button } from '@mui/material';
import { Database } from 'lucide-react';
import { MappingRow } from './MappingRow';
import type { Mapping, SemanticTerm } from './types';
import { getMappingUniqueId } from '../../utils/mappingId';
import { useState, useEffect, useRef, useCallback } from 'react';

interface MappingListProps {
  loading: boolean;
  mappings: Mapping[];
  // MappingRow props
  savedRows: Set<string>;
  compactRows: boolean;
  keyboardExpanded: boolean;
  setKeyboardExpanded: (expanded: boolean) => void;
  toggleMapping: (id: string) => void;
  confirmEditing: (id: string, term?: string) => void;
  searchSemanticTerms: (query: string) => Promise<SemanticTerm[]>;
  selectSemanticTerm: (term: SemanticTerm, mappingId: string) => void;
  handleCreateAndSelectTerm: (mappingId: string, termName: string) => Promise<SemanticTerm | null>;
  setOverride: (id: string, value: boolean) => void;
  setIgnored: (id: string, value: boolean) => void;
  openReplaceConfirm: (index: number) => void;
  openLineageModal: (mapping: any) => void;
  onAutoMap?: () => void;
}

export function MappingList(props: MappingListProps) {
  const [visibleCount, setVisibleCount] = useState(25); // Start with 25 items
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const observerRef = useRef<HTMLDivElement>(null);
  const ITEMS_PER_LOAD = 25;

  // Reset visible count when mappings change (e.g., filtering)
  useEffect(() => {
    setVisibleCount(25);
  }, [props.mappings.length]);

  const loadMore = useCallback(() => {
    if (isLoadingMore || visibleCount >= props.mappings.length) return;
    
    setIsLoadingMore(true);
    // Simulate async loading (in a real app, this might fetch from API)
    setTimeout(() => {
      setVisibleCount(prev => Math.min(prev + ITEMS_PER_LOAD, props.mappings.length));
      setIsLoadingMore(false);
    }, 100);
  }, [isLoadingMore, visibleCount, props.mappings.length]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && visibleCount < props.mappings.length) {
          loadMore();
        }
      },
      { threshold: 0.1 }
    );

    if (observerRef.current) {
      observer.observe(observerRef.current);
    }

    return () => observer.disconnect();
  }, [loadMore, visibleCount, props.mappings.length]);

  const visibleMappings = props.mappings.slice(0, visibleCount);

  if (props.loading && props.mappings.length === 0) {
    return null; // Let parent handle initial loading state
  }

  if (props.mappings.length === 0) {
    return (
      <Card sx={{ 
        p: 8, 
        textAlign: 'center', 
        borderRadius: 4,
        background: 'linear-gradient(135deg, rgba(248, 250, 252, 0.5) 0%, rgba(241, 245, 249, 0.5) 100%)',
        border: '1px dashed rgba(148, 163, 184, 0.4)',
        boxShadow: 'none'
      }}>
        <Box sx={{ 
          width: 80, 
          height: 80, 
          borderRadius: '50%', 
          bgcolor: 'white', 
          display: 'flex', 
          alignItems: 'center', 
          justifyContent: 'center',
          margin: '0 auto 24px',
          boxShadow: '0 4px 12px rgba(0,0,0,0.05)'
        }}>
          <Database width={40} height={40} style={{ color: '#94a3b8' }} />
        </Box>
        <Typography variant="h5" sx={{ mb: 1.5, fontWeight: 600, color: '#1e293b' }}>
          No Mappings Found
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 400, mx: 'auto', mb: 4 }}>
          No database columns found in this view. Try adjusting your filters or use the Auto-Mapper to discover new terms.
        </Typography>
        {props.onAutoMap && (
          <Box>
             <Button 
               variant="contained" 
               onClick={props.onAutoMap}
               sx={{ 
                 borderRadius: 2,
                 textTransform: 'none',
                 fontWeight: 600,
                 background: 'linear-gradient(135deg, #8b5cf6 0%, #7c3aed 100%)',
                 boxShadow: '0 4px 12px rgba(139, 92, 246, 0.3)',
                 px: 4,
                 py: 1.5
               }}
             >
               Launch Auto-Mapper
             </Button>
          </Box>
        )}
      </Card>
    );
  }

  return (
    <Stack spacing={2}>
      {visibleMappings.map((mapping, idx) => (
        <MappingRow key={getMappingUniqueId(mapping)} mapping={mapping} idx={idx} {...props} />
      ))}
      
      {/* Loading indicator and intersection observer trigger */}
      {visibleCount < props.mappings.length && (
        <Box ref={observerRef} sx={{ p: 2, textAlign: 'center' }}>
          {isLoadingMore ? (
            <Typography variant="body2" color="text.secondary">
              Loading more mappings...
            </Typography>
          ) : (
            <Typography variant="body2" color="text.secondary">
              Scroll for more ({props.mappings.length - visibleCount} remaining)
            </Typography>
          )}
        </Box>
      )}
      
      {/* Show total count at bottom */}
      <Box sx={{ textAlign: 'center', mt: 1 }}>
        <Typography variant="caption" color="text.secondary">
          Showing {visibleMappings.length} of {props.mappings.length} mappings
        </Typography>
      </Box>
    </Stack>
  );
}
