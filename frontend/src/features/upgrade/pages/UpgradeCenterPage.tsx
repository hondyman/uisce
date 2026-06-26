import type { FC } from 'react';
import { Box, Breadcrumbs, Link as MUILink, Typography, Grid, Card, CardContent, Button } from '@mui/material';
// replaced Link usages with BlockableLink
import BlockableLink from '../../../components/RouteBlocker/BlockableLink';
import SystemUpdateAltIcon from '@mui/icons-material/SystemUpdateAlt';
import AssessmentIcon from '@mui/icons-material/Assessment';

const UpgradeCenterPage: FC = () => {
  return (
    <Box sx={{ p: 3 }}>
      <Breadcrumbs aria-label="breadcrumb" sx={{ mb: 2 }}>
  <MUILink component={BlockableLink} color="inherit" to="/">
          Home
        </MUILink>
        <Typography color="text.primary">Upgrade Center</Typography>
      </Breadcrumbs>

      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2, gap: 1 }}>
        <SystemUpdateAltIcon color="primary" />
        <Typography variant="h4">Upgrade Center</Typography>
      </Box>
      <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
        Plan, review, and apply upgrades safely. Compare versions, review diffs, and manage generated artifacts.
      </Typography>

      <Grid container spacing={2}>
        <Grid item xs={12} sm={6} md={4}>
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <AssessmentIcon color="primary" />
                <Typography variant="h6">Versions</Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Manage versions, canary, activation, and rollback.
              </Typography>
              <Button variant="outlined" component={BlockableLink} to="/upgrade/versions">
                Open
              </Button>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={4}>
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <AssessmentIcon color="primary" />
                <Typography variant="h6">Upgrade Compare</Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Compare current vs. target versions and review semantic diffs.
              </Typography>
              <Button variant="contained" component={BlockableLink} to="/upgrade-compare">
                Open
              </Button>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={4}>
          <Card variant="outlined">
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <AssessmentIcon color="primary" />
                <Typography variant="h6">Views Catalog</Typography>
              </Box>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Browse generated and resolved views, search, and export.
              </Typography>
              <Button variant="outlined" component={BlockableLink} to="/views">
                Browse
              </Button>
            </CardContent>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

export default UpgradeCenterPage;
