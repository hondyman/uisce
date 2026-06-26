import { JSX } from 'react';
import { Card, CardContent, Typography, List, ListItem, ListItemText, Divider, Box } from '@mui/material';
import { TenantInstance, Product } from '../types';

interface TenantInstanceDetailsProps {
  instance: TenantInstance | null;
}

export default function TenantInstanceDetails(props: TenantInstanceDetailsProps): JSX.Element {
  const { instance } = props;
  if (!instance) {
    return (
      <Card>
        <CardContent>
          <Typography variant="h6">Please select a tenant instance to see the details.</Typography>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h5" gutterBottom>
          Instance: {String(instance.display_name ?? instance.instance_name ?? instance.id)}
        </Typography>
        <Divider sx={{ my: 2 }} />
        <Typography variant="h6">Products</Typography>
        <List>
          {Array.isArray(instance.tenant_products) && instance.tenant_products.map((product: Product, _idx: number) => {
            return (
              <Box key={product.id != null && typeof product.id !== 'object' ? String(product.id) : ''} sx={{ mb: 2 }}>
                <ListItem>
                  <ListItemText primary={product.alpha_product?.product_name ?? product.id} primaryTypographyProps={{ fontWeight: 'bold' }} />
                </ListItem>
                <List component="div" disablePadding sx={{ pl: 4 }}>
                  {Array.isArray(product.tenant_product_datasources) && product.tenant_product_datasources.map((dataSource, _dsIdx) => {
                    return (
                      <ListItem key={dataSource.id != null && typeof dataSource.id !== 'object' ? String(dataSource.id) : ''}>
                        <ListItemText primary={dataSource.alpha_datasource?.datasource_name ?? dataSource.source_name ?? dataSource.id} />
                      </ListItem>
                    );
                  })}
                </List>
              </Box>
            );
          })}
        </List>
      </CardContent>
    </Card>
  );
}