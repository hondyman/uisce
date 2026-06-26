import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  InputBase,
  IconButton,
  Button,
  Avatar,
  Grid,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  CircularProgress,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterIcon,
  Add as AddIcon,
  Folder as FolderIcon,
  MoreVert as MoreVertIcon,
  StarBorder as StarIcon,
  AccessTime as AccessTimeIcon,
  ShowChart as ShowChartIcon,
  Storage as StorageIcon,
  GridView as GridViewIcon,
  List as ListViewIcon,
  Sort as SortIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { useFolders } from '../../api/explorer';
import { useReportTemplates } from '../../api/reporting';

// --- Types ---
interface QueryItem {
  id: string;
  name: string;
  author: string;
  updatedAt: string;
  folderId?: string;
  tags?: string[];
  type: 'query';
}

interface FolderItem {
  id: string;
  name: string;
  queryCount: number;
  updatedAt: string;
  type: 'folder';
}

export const QueryLibraryDashboard: React.FC = () => {
  const navigate = useNavigate();

  // --- Real Data Fetching ---
  const { data: apiFolders, isLoading: isLoadingFolders } = useFolders();
  const { data: apiQueries, isLoading: isLoadingQueries } = useReportTemplates();

  // --- State ---
  const [searchQuery, setSearchQuery] = useState('');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('list');

  // --- Derived Data ---
  
  // Transform API Folders to component format
  const folders = useMemo<FolderItem[]>(() => {
    if (!apiFolders) return [];
    return apiFolders.map(f => ({
      id: f.id,
      name: f.name,
      queryCount: f.items ? f.items.filter(i => i.itemType === 'query').length : 0,
      updatedAt: 'Recently', // TODO: Add updatedAt to ExplorerFolder API
      type: 'folder'
    }));
  }, [apiFolders]);

  // Transform API Queries to component format
  const queries = useMemo<QueryItem[]>(() => {
    if (!apiQueries) return [];
    return apiQueries.map(q => ({
      id: q.id,
      name: q.name,
      author: 'User', // TODO: Get author from metadata
      updatedAt: q.updatedAt || 'Unknown',
      folderId: undefined, // TODO: Map query to folder if backend supports it
      tags: [],
      type: 'query'
    }));
  }, [apiQueries]);


  // Filtered Lists
  const filteredFolders = useMemo(() => 
    folders.filter((f) => f.name.toLowerCase().includes(searchQuery.toLowerCase())),
  [folders, searchQuery]);

  const filteredQueries = useMemo(() => 
    queries.filter((q) => q.name.toLowerCase().includes(searchQuery.toLowerCase())),
  [queries, searchQuery]);

  const recentActivity = useMemo(() => {
    return [...queries].sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()).slice(0, 4);
  }, [queries]);

  const isLoading = isLoadingFolders || isLoadingQueries;

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', bgcolor: '#f8f8f5' }}>
        <CircularProgress sx={{ color: '#f9f506' }} />
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', height: '100vh', bgcolor: '#f8f8f5', overflow: 'hidden' }}>
      {/* Sidebar */}
      <Box sx={{ 
        width: 260, 
        bgcolor: '#ffffff', 
        borderRight: '1px solid #e6e6db', 
        display: 'flex', 
        flexDirection: 'column',
        flexShrink: 0 
      }}>
        {/* Logo Area */}
        <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1.5 }}>
           <Box sx={{
              width: 32,
              height: 32,
              bgcolor: '#f9f506',
              borderRadius: '6px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center'
            }}>
              <StorageIcon sx={{ fontSize: 20, color: '#181811' }} />
           </Box>
           <Typography variant="subtitle1" fontWeight={700} sx={{ color: '#181811' }}>
             Data Explorer
           </Typography>
        </Box>

        {/* Navigation */}
        <Box sx={{ px: 2, py: 2, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Button 
            startIcon={<AccessTimeIcon />} 
            sx={{ 
              justifyContent: 'flex-start', 
              color: '#181811', 
              textTransform: 'none', 
              bgcolor: '#f4f4ec', 
              fontWeight: 600,
              '&:hover': { bgcolor: '#ecece4' }
            }}
          >
            Recent
          </Button>
          <Button 
            startIcon={<StarIcon />} 
            sx={{ 
              justifyContent: 'flex-start', 
              color: '#666660', 
              textTransform: 'none',
              '&:hover': { bgcolor: '#f4f4ec', color: '#181811' }
            }}
          >
            Favorites
          </Button>
          <Button 
            startIcon={<FolderIcon />} 
            sx={{ 
              justifyContent: 'flex-start', 
              color: '#666660', 
              textTransform: 'none',
              '&:hover': { bgcolor: '#f4f4ec', color: '#181811' }
            }}
          >
            All Folders
          </Button>
        </Box>

        {/* Categories / Tags (Mocked for now as backend support pending) */}
        <Box sx={{ px: 2, pt: 2 }}>
          <Typography variant="caption" sx={{ color: '#888880', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.5, mb: 1, display: 'block' }}>
            Categories
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
            {['Sales', 'Marketing', 'Product', 'Finance'].map((cat) => (
              <Box key={cat} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5, cursor: 'pointer', '&:hover span': { color: '#181811' } }}>
                <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: '#e6e6db' }} />
                <Typography variant="body2" sx={{ color: '#666660' }}>{cat}</Typography>
              </Box>
            ))}
          </Box>
        </Box>
      </Box>

      {/* Main Content */}
      <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        
        {/* Header */}
        <Box sx={{ p: 3, borderBottom: '1px solid #e6e6db', bgcolor: '#ffffff' }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
            <Typography variant="h5" fontWeight={700} sx={{ color: '#181811' }}>
              Library
            </Typography>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={() => navigate('/reports/explorer')}
              sx={{
                bgcolor: '#181811',
                color: '#ffffff',
                textTransform: 'none',
                fontWeight: 600,
                borderRadius: '8px',
                boxShadow: 'none',
                '&:hover': { bgcolor: '#000000', boxShadow: 'none' }
              }}
            >
              New Query
            </Button>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Paper
              elevation={0}
              sx={{
                p: '2px 4px',
                display: 'flex',
                alignItems: 'center',
                width: 400,
                bgcolor: '#f4f4ec',
                border: '1px solid transparent',
                borderRadius: '8px',
                transition: 'all 0.2s',
                '&:hover': { bgcolor: '#ffffff', borderColor: '#e6e6db' },
                '&:focus-within': { bgcolor: '#ffffff', borderColor: '#f9f506', boxShadow: '0 0 0 2px rgba(249, 245, 6, 0.2)' }
              }}
            >
              <IconButton sx={{ p: '10px', color: '#888880' }} aria-label="search">
                <SearchIcon />
              </IconButton>
              <InputBase
                sx={{ ml: 1, flex: 1, color: '#181811', fontWeight: 500 }}
                placeholder="Search queries and folders..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
              <IconButton sx={{ p: '10px', color: '#888880' }} aria-label="filter">
                <FilterIcon />
              </IconButton>
            </Paper>

            <Box sx={{ ml: 'auto', display: 'flex', gap: 1 }}>
              <IconButton 
                 size="small" 
                 onClick={() => setViewMode('grid')}
                 sx={{ 
                   color: viewMode === 'grid' ? '#181811' : '#888880', 
                   bgcolor: viewMode === 'grid' ? '#e6e6db' : 'transparent' 
                 }}
              >
                <GridViewIcon fontSize="small" />
              </IconButton>
              <IconButton 
                size="small" 
                onClick={() => setViewMode('list')}
                sx={{ 
                   color: viewMode === 'list' ? '#181811' : '#888880', 
                   bgcolor: viewMode === 'list' ? '#e6e6db' : 'transparent' 
                 }}
              >
                <ListViewIcon fontSize="small" />
              </IconButton>
            </Box>
          </Box>
        </Box>

        {/* Content Area */}
        <Box sx={{ flex: 1, overflowY: 'auto', p: 3 }}>
          
          {/* Recent Activity */}
          {recentActivity.length > 0 && (
            <Box sx={{ mb: 4 }}>
              <Typography variant="subtitle2" sx={{ color: '#888880', fontWeight: 600, mb: 2, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                Recent Activity
              </Typography>
              <Grid container spacing={2}>
                {recentActivity.map((query) => (
                  <Grid item xs={12} sm={6} md={3} key={query.id}>
                    <Paper
                      elevation={0}
                      sx={{
                        p: 2,
                        border: '1px solid #e6e6db',
                        borderRadius: '12px',
                        cursor: 'pointer',
                        transition: 'all 0.2s',
                        '&:hover': { borderColor: '#f9f506', transform: 'translateY(-2px)', boxShadow: '0 4px 12px rgba(0,0,0,0.05)' }
                      }}
                      onClick={() => navigate('/reports/explorer')}
                    >
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1.5 }}>
                        <Box sx={{ p: 0.75, borderRadius: '6px', bgcolor: '#f4f4ec', color: '#181811' }}>
                          <ShowChartIcon fontSize="small" />
                        </Box>
                        <IconButton size="small" sx={{ color: '#888880', mt: -1, mr: -1 }}>
                          <MoreVertIcon fontSize="small" />
                        </IconButton>
                      </Box>
                      <Typography variant="subtitle1" fontWeight={600} noWrap sx={{ color: '#181811', mb: 0.5 }}>
                        {query.name}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#888880' }}>
                        Opened {query.updatedAt}
                      </Typography>
                    </Paper>
                  </Grid>
                ))}
              </Grid>
            </Box>
          )}

          {/* Folders */}
          {filteredFolders.length > 0 && (
            <Box sx={{ mb: 4 }}>
              <Typography variant="subtitle2" sx={{ color: '#888880', fontWeight: 600, mb: 2, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                Folders
              </Typography>
              <Grid container spacing={2}>
                {filteredFolders.map((folder) => (
                  <Grid item xs={12} sm={6} md={3} lg={2.4} key={folder.id}>
                     <Paper
                      elevation={0}
                      sx={{
                        p: 2,
                        display: 'flex',
                        alignItems: 'center',
                        gap: 2,
                        border: '1px solid #e6e6db',
                        borderRadius: '12px',
                        cursor: 'pointer',
                        transition: 'all 0.2s',
                        '&:hover': { borderColor: '#f9f506', bgcolor: '#ffffff' }
                      }}
                    >
                      <FolderIcon sx={{ color: '#f9f506', fontSize: 28 }} />
                      <Box sx={{ minWidth: 0 }}>
                         <Typography variant="body2" fontWeight={600} noWrap sx={{ color: '#181811' }}>
                           {folder.name}
                         </Typography>
                         <Typography variant="caption" sx={{ color: '#888880' }}>
                           {folder.queryCount} items
                         </Typography>
                      </Box>
                    </Paper>
                  </Grid>
                ))}
              </Grid>
            </Box>
          )}

          {/* All Queries List */}
          <Box>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
              <Typography variant="subtitle2" sx={{ color: '#888880', fontWeight: 600, textTransform: 'uppercase', letterSpacing: 0.5 }}>
                All Queries
              </Typography>
              <Button startIcon={<SortIcon />} size="small" sx={{ color: '#666660', textTransform: 'none' }}>
                Sort by Name
              </Button>
            </Box>
            
            <TableContainer component={Paper} elevation={0} sx={{ border: '1px solid #e6e6db', borderRadius: '12px' }}>
              <Table>
                <TableHead sx={{ bgcolor: '#fbfbf9' }}>
                  <TableRow>
                     <TableCell sx={{ color: '#666660', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', py: 1.5 }}>Name</TableCell>
                     <TableCell sx={{ color: '#666660', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', py: 1.5 }}>Author</TableCell>
                     <TableCell sx={{ color: '#666660', fontWeight: 600, fontSize: '0.75rem', textTransform: 'uppercase', py: 1.5 }}>Last Modified</TableCell>
                     <TableCell align="right" sx={{ py: 1.5 }}></TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredQueries.map((query) => (
                    <TableRow 
                      key={query.id}
                      hover
                      sx={{ 
                        cursor: 'pointer',
                        '&:last-child td, &:last-child th': { border: 0 },
                        '&:hover': { bgcolor: '#fbfbf9' }
                      }}
                      onClick={() => navigate('/reports/explorer')}
                    >
                      <TableCell component="th" scope="row">
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                          <Box sx={{ p: 0.5, borderRadius: '4px', bgcolor: '#e6e6db', color: '#666660' }}>
                            <ShowChartIcon sx={{ fontSize: 16 }} />
                          </Box>
                          <Typography variant="body2" fontWeight={500} sx={{ color: '#181811' }}>
                            {query.name}
                          </Typography>
                          {query.tags && query.tags.map(tag => (
                            <Chip key={tag} label={tag} size="small" sx={{ height: 20, fontSize: '0.625rem', bgcolor: '#f4f4ec' }} />
                          ))}
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Avatar sx={{ width: 20, height: 20, fontSize: '0.75rem', bgcolor: '#181811' }}>{query.author[0]}</Avatar>
                          <Typography variant="body2" sx={{ color: '#444440' }}>{query.author}</Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" sx={{ color: '#666660' }}>{query.updatedAt}</Typography>
                      </TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={(e) => { e.stopPropagation(); /* Menu logic */ }}>
                          <MoreVertIcon fontSize="small" sx={{ color: '#888880' }} />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                  {filteredQueries.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={4} align="center" sx={{ py: 6 }}>
                        <Typography variant="body2" color="text.secondary">
                          No queries found matching "{searchQuery}"
                        </Typography>
                        <Button 
                          variant="text" 
                          startIcon={<AddIcon />} 
                          sx={{ mt: 1, color: '#181811' }}
                          onClick={() => navigate('/reports/explorer')}
                        >
                          Create New Query
                        </Button>
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>

        </Box>
      </Box>
    </Box>
  );
};


