import { useEffect, useState } from 'react';
import { Box, Tooltip, IconButton } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';
import type { Product, TenantInstance } from '../types';

interface ProductGridProps {
  instance: TenantInstance;
  products: Array<Record<string, any>>; // small shape prepared by parent
  selectedProductId: string | null;
  onSelectProduct: (id: string | null) => void;
  onAddProduct: (instance: TenantInstance) => void;
  onEditProduct: (product: Product) => void;
  onDeleteProduct: (productId: string) => void;
}

const ProductGrid: React.FC<ProductGridProps> = (props) => {
  const { instance: _instance, products, selectedProductId, onSelectProduct, onAddProduct: _onAddProduct, onEditProduct, onDeleteProduct } = props;
  const [DataGridModule, setDataGridModule] = useState<unknown>(null);

  useEffect(() => {
    let mounted = true;
    import('@mui/x-data-grid').then((mod) => {
      if (mounted) setDataGridModule(mod);
    });
    return () => { mounted = false; };
  }, []);

  const injectedLoading = `
    .product-grid-loading{ height:100%; display:flex; align-items:center; justify-content:center }
  `;

  if (!DataGridModule) {
    return <><style dangerouslySetInnerHTML={{ __html: injectedLoading }} /><div className="product-grid-loading">Loading products...</div></>;
  }

  const DataGrid = (DataGridModule as any)?.DataGrid as React.ComponentType<any> | undefined;

  if (!DataGrid) return <><style dangerouslySetInnerHTML={{ __html: injectedLoading }} /><div className="product-grid-loading">Loading products...</div></>;

  const rows = products.map(p => ({
    id: p.id != null && typeof p.id !== 'object' ? String(p.id) : '',
    name: p.alpha_product?.product_name,
    code: p.alpha_product?.product_code,
    version: p.version,
    status: p.alpha_product?.is_active ? 'Active' : 'Inactive',
    fullProduct: p
  }));

  const columns = [
  { field: 'code', headerName: 'Product Code', flex: 1, renderCell: (params: any) => (<Tooltip title={params.row.name}><Box component="span" sx={{ width: '100%' }}>{params.value}</Box></Tooltip>) },
    { field: 'version', headerName: 'Version', width: 100 },
    { field: 'status', headerName: 'Status', width: 120, renderCell: (params: any) => (
      <Tooltip title={params.value}>
        <Box component="span">
          {params.value === 'Active' ? <CheckCircleIcon color="success" /> : <WarningAmberIcon color="warning" />}
        </Box>
      </Tooltip>
    ) },
    { field: 'actions', headerName: 'Actions', width: 140, renderCell: (params: any) => (
      <Box sx={{ display: 'flex', gap: 1 }}>
        <Tooltip title="Edit">
          <IconButton size="small" onClick={() => onEditProduct(params.row.fullProduct)}>
            <EditIcon fontSize="small" />
          </IconButton>
        </Tooltip>
        <Tooltip title="Unassign">
          <IconButton size="small" color="error" onClick={() => onDeleteProduct(params.row.id)}>
            <DeleteIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>
    )},
  ];

  return (
    <Box sx={{ height: '100%', position: 'relative' }}>
      <DataGrid
        rows={rows}
        columns={columns}
        hideFooter
        density="compact"
        rowSelectionModel={selectedProductId ? [selectedProductId] : []}
        onRowSelectionModelChange={(selectionModel: string[] | number[] | readonly string[] | readonly number[]) => {
          const selectedId = Array.isArray(selectionModel) ? selectionModel[0] : undefined;
          const idStr = selectedId == null ? null : String(selectedId);
          onSelectProduct(idStr);
        }}
        checkboxSelection={false}
      />
    </Box>
  );
};

export default ProductGrid;
