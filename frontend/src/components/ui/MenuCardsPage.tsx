import React from 'react';
import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  Typography,
  Grid,
  IconButton,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useNavigate } from 'react-router-dom';
import { useTheme } from '@mui/material/styles';

export interface MenuCardItem {
  id: string;
  label: string;
  description?: string;
  icon?: React.ReactNode;
  to?: string;
}

interface MenuCardsPageProps {
  title: string;
  items: MenuCardItem[];
  onClose: () => void;
  onSelect?: (item: MenuCardItem) => void;
  icon?: React.ReactNode;
}

export const MenuCardsPage: React.FC<MenuCardsPageProps> = ({
  title,
  items,
  onClose,
  onSelect,
  icon,
}) => {
  const navigate = useNavigate();
  const theme = useTheme();

  const handleSelect = (item: MenuCardItem) => {
    if (onSelect) {
      onSelect(item);
    } else if (item.to) {
      navigate(item.to);
    }
    onClose();
  };

  return (
    <Box
      sx={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        bottom: 0,
        bgcolor: 'background.default',
        zIndex: 1299,
        overflow: 'auto',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {/* Header */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          bgcolor: 'background.paper',
        }}
      >
        <IconButton onClick={onClose} size="small" aria-label="Go back">
          <ArrowBackIcon />
        </IconButton>
        {icon && <Box sx={{ display: 'flex', alignItems: 'center', mr: 1 }}>{icon}</Box>}
        <Typography variant="h6" component="span" sx={{ fontWeight: 600 }}>
          {title}
        </Typography>
      </Box>

      {/* Cards Grid */}
      <Box sx={{ flex: 1, p: 3 }}>
        <Grid container spacing={2}>
          {items.map((item) => (
            <Grid size={{ xs: 12, sm: 6, md: 4 }} key={item.id}>
              <Card
                variant="outlined"
                sx={{
                  height: '100%',
                  transition: 'box-shadow 0.2s, border-color 0.2s',
                  '&:hover': {
                    boxShadow: 3,
                    borderColor: theme.palette.primary.main,
                  },
                }}
              >
                <CardActionArea
                  onClick={() => handleSelect(item)}
                  sx={{
                    height: '100%',
                    display: 'flex',
                    flexDirection: 'column',
                    alignItems: 'flex-start',
                    justifyContent: 'flex-start',
                    p: 2,
                  }}
                >
                  <CardContent sx={{ width: '100%', p: 0 }}>
                    <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                      {item.icon && (
                        <Box
                          sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            width: 40,
                            height: 40,
                            borderRadius: 1,
                            bgcolor: theme.palette.mode === 'dark' ? 'rgba(255,255,255,0.05)' : 'rgba(0,0,0,0.03)',
                            color: 'primary.main',
                            fontSize: '1.25rem',
                            flexShrink: 0,
                          }}
                        >
                          {item.icon}
                        </Box>
                      )}
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Typography
                          variant="subtitle1"
                          sx={{
                            fontWeight: 600,
                            mb: 0.5,
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                          }}
                        >
                          {item.label}
                        </Typography>
                        {item.description && (
                          <Typography
                            variant="body2"
                            color="text.secondary"
                            sx={{
                              overflow: 'hidden',
                              textOverflow: 'ellipsis',
                              display: '-webkit-box',
                              WebkitLineClamp: 2,
                              WebkitBoxOrient: 'vertical',
                            }}
                          >
                            {item.description}
                          </Typography>
                        )}
                      </Box>
                    </Box>
                  </CardContent>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>

        {items.length === 0 && (
          <Box sx={{ textAlign: 'center', py: 8 }}>
            <Typography color="text.secondary">No items available</Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default MenuCardsPage;
