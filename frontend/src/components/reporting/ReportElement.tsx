import React, { useMemo } from 'react';
import type { FC, KeyboardEvent } from 'react';
import { motion } from 'framer-motion';
import { Rnd } from 'react-rnd';
import {
  Box,
  Typography,
  IconButton,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  useTheme,
} from '@mui/material';
import DeleteIcon from '@mui/icons-material/Delete';
import {
  ELEMENT_TYPES,
  evaluateReportExpression,
  formatReportValue,
  computeAggregate,
} from './reportingUtils';
import { evaluateDynamicProperty, DynamicProperty } from './evaluateDynamicProperty';
import { evaluateCondition } from '../ExpressionBuilder/AdvancedConditionBuilder';
import { ColumnConfig, TotalsConfig, BandingConfig, ConditionalRule, SparklineConfig, PaginationConfig, FormatType, createDefaultBandingConfig, createDefaultTotalsConfig, createDefaultPaginationConfig } from './tableColumnModel';
import { SparklineInline } from './tableStyling/SparklinePicker';
import { FormEmbedShell } from './FormEmbedShell';
import type { FormTemplateSpec } from './form/FormManagerTypes';

interface ReportElementProps {
  id: string;
  type: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
  properties: Record<string, any>;
  onUpdate: (id: string, updates: Partial<any>) => void;
  onDelete: (id: string) => void;
  onSelect: (id: string) => void;
  isSelected: boolean;
  isLivePreview?: boolean;
  previewData?: any[] | null;
  formRegistry?: Record<string, FormTemplateSpec>;
}

const ReportElement: FC<ReportElementProps> = ({
  id,
  type,
  position,
  size,
  properties,
  onUpdate,
  onDelete,
  onSelect,
  isSelected,
  isLivePreview = false,
  previewData = null,
  formRegistry = {},
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  
  // Check conditional visibility rule
  const isHidden = useMemo(() => {
    if (properties.visibilityCondition && isLivePreview) {
      const sampleRow = previewData && previewData.length > 0 ? previewData[0] : {};
      return evaluateCondition(properties.visibilityCondition, sampleRow);
    }
    return false;
  }, [properties.visibilityCondition, isLivePreview, previewData]);

  if (isHidden) {
    return null;
  }

  // --- Table data computations lifted to top level (hooks must not be inside nested functions) ---
  const rows = isLivePreview && previewData && previewData.length > 0
    ? previewData.slice(0, 15)
    : Array.from({ length: 4 }, (_, i) => ({
        id: `ID-00${i + 1}`,
        name: `Sample Record ${i + 1}`,
        status: ['Active', 'Pending', 'Closed'][i % 3],
        amount: 1500 * (i + 1),
      }));

  const rawColumns = properties.columns || [];
  const columns: ColumnConfig[] = rawColumns.length > 0 && typeof rawColumns[0] === 'string'
    ? (rawColumns as string[]).map((field, i) => ({
        id: `col_${field}_${i}`,
        field,
        headerText: String(field).charAt(0).toUpperCase() + String(field).slice(1).replace(/_/g, ' '),
        widthPx: 120,
        visible: true,
        headerStyle: {},
        bodyStyle: {},
        align: 'left',
        verticalAlign: 'middle',
        wrap: false,
        formatType: 'Auto',
        formatMask: '',
        formatPrefix: '',
        formatSuffix: '',
      }))
    : (rawColumns as ColumnConfig[]);

  const visibleColumns = columns.filter(c => c.visible !== false);

  const totalsConfig: TotalsConfig = properties.totals
    ? (properties.totals as TotalsConfig)
    : createDefaultTotalsConfig();

  const allDataRows = isLivePreview && previewData && previewData.length > 0 ? previewData : rows;

  // Expensive: compute aggregate totals only when data/columns change
  const grandTotalRow = useMemo(() => {
    if (!totalsConfig.grandTotal?.enabled) return null;
    return visibleColumns.map(col => {
      let fn: string | { customExpression: string } | undefined;
      if (col.aggregate?.enabled) {
        fn = col.aggregate.function;
      } else if (col.aggregate === undefined || col.aggregate.enabled === false) {
        const numericTypes: FormatType[] = ['Auto', 'Currency', 'Percent', 'Decimal', 'Integer', 'Custom'];
        if (numericTypes.includes(col.formatType)) {
          fn = 'SUM';
        }
      }
      if (!fn) return null;
      const val = typeof fn === 'object' && 'customExpression' in fn
        ? computeAggregate(allDataRows, col.field, fn)
        : computeAggregate(allDataRows, col.field, fn as any);
      return { colId: col.id, value: val, col };
    }).filter(Boolean);
  }, [visibleColumns, totalsConfig, allDataRows]);

  const pagination: PaginationConfig = properties.pagination
    ? (properties.pagination as PaginationConfig)
    : createDefaultPaginationConfig();

  const pageTotalRows = useMemo(() => {
    if (!pagination.pageTotalEnabled || pagination.mode !== 'paginate') return [];
    const perPage = pagination.rowsPerPage || 20;
    const chunks: any[][] = [];
    for (let i = 0; i < allDataRows.length; i += perPage) {
      chunks.push(allDataRows.slice(i, i + perPage));
    }
    return chunks.map(chunk =>
      visibleColumns.map(col => {
        let fn: string | { customExpression: string } | undefined;
        if (col.aggregate?.enabled) {
          fn = col.aggregate.function;
        } else {
          const numericTypes: FormatType[] = ['Auto', 'Currency', 'Percent', 'Decimal', 'Integer', 'Custom'];
          if (numericTypes.includes(col.formatType)) {
            fn = 'SUM';
          }
        }
        if (!fn) return null;
        const val = typeof fn === 'object' && 'customExpression' in fn
          ? computeAggregate(chunk, col.field, fn)
          : computeAggregate(chunk, col.field, fn as any);
        return { colId: col.id, value: val, col };
      }).filter(Boolean)
    );
  }, [visibleColumns, allDataRows, pagination]);

  // Resolve dynamic styles via SSRS / Crystal expression evaluator
  const activeRow = previewData && previewData.length > 0 ? previewData[0] : {};
  const evaluatedTextColor = evaluateDynamicProperty(properties.textColor, activeRow) || '#111827';
  const evaluatedBgColor = evaluateDynamicProperty(properties.backgroundColor, activeRow) || 'transparent';

  const textStyles: React.CSSProperties = {
    fontFamily: properties.fontFamily || 'inherit',
    fontSize: Number(properties.fontSize) || 12,
    textAlign: properties.textAlign || 'left',
    fontWeight: properties.fontWeight || (properties.bold ? 700 : 500),
    fontStyle: properties.italic ? 'italic' : 'normal',
    textDecoration: [
      properties.underline ? 'underline' : '',
      properties.strikethrough ? 'line-through' : '',
    ].filter(Boolean).join(' ') || 'none',
    textTransform: properties.textTransform || 'none',
    color: evaluatedTextColor,
  };

  const containerStyles = {
    width: '100%',
    height: '100%',
    backgroundColor: evaluatedBgColor !== 'transparent' ? evaluatedBgColor : 'transparent',
    border: `${properties.borderWidth ?? 0}px ${properties.borderStyle || 'solid'} ${properties.borderColor || '#cccccc'}`,
    borderRadius: `${properties.borderRadius ?? 4}px`,
    padding: `${properties.padding ?? 4}px`,
    overflow: 'hidden',
    display: 'flex',
    flexDirection: 'column' as const,
  };

  const renderContent = () => {
    if (type === ELEMENT_TYPES.TABLE || type === ELEMENT_TYPES.MATRIX || type === ELEMENT_TYPES.LIST) {
      const bandingConfig: BandingConfig = properties.banding
        ? (properties.banding as BandingConfig)
        : createDefaultBandingConfig();

      const conditionalRules: ConditionalRule[] = properties.conditionalRules || [];

      // Empty Container Scaffolding: Active Drop Zone Target
      if (columns.length === 0) {
        return (
          <Box
            onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); }}
            onDrop={(e) => {
              e.preventDefault();
              e.stopPropagation();
              try {
                const data = JSON.parse(e.dataTransfer.getData('application/json'));
                if ((data.type === 'bo-field-bundle' || data.type === 'bo-field') && (data.fields || data.field)) {
                  const droppedList = data.fields || [data.field];
                  onUpdate(id, {
                    properties: {
                      ...properties,
                      columns: droppedList.map((f: any) => ({
                        id: `col_${f.technicalName || f.name}_${Date.now()}`,
                        field: f.technicalName || f.name,
                        headerText: f.label || f.technicalName || f.name,
                        widthPx: 120,
                        visible: true,
                        headerStyle: {},
                        bodyStyle: {},
                        align: 'left',
                        verticalAlign: 'middle',
                        wrap: false,
                        formatType: 'Auto',
                        formatMask: '',
                        formatPrefix: '',
                        formatSuffix: '',
                      })),
                      pagination: createDefaultPaginationConfig(),
                    },
                  });
                }
              } catch (err) {
                console.error('Drop error on table scaffolding:', err);
              }
            }}
            sx={{
              width: '100%', height: '100%', border: '2px dashed',
              borderColor: 'primary.main', borderRadius: '8px',
              bgcolor: 'rgba(99, 102, 241, 0.04)',
              display: 'flex', flexDirection: 'column', alignItems: 'center',
              justifyContent: 'center', p: 2, textAlign: 'center', cursor: 'pointer',
              '&:hover': { bgcolor: 'rgba(99, 102, 241, 0.08)', borderColor: 'primary.light' },
            }}
          >
            <Typography variant="subtitle2" sx={{ fontWeight: 700, color: 'primary.main', fontSize: '0.8rem', mb: 0.5 }}>
              📥 {type.toUpperCase()} Container Ready
            </Typography>
            <Typography variant="caption" color="text.secondary" sx={{ fontSize: '0.7rem' }}>
              Drag &amp; Drop selected fields here, or select columns in Properties
            </Typography>
          </Box>
        );
      }

      const getRowStyle = (rowIdx: number) => {
        const style: React.CSSProperties = {};
        if (bandingConfig.bandedRows) {
          style.backgroundColor = rowIdx % 2 === 0 ? 'transparent' : bandingConfig.bandColor;
        }
        return style;
      };

      const getCellStyle = (col: ColumnConfig, rowIdx: number) => {
        const style: React.CSSProperties = {
          textAlign: col.align || 'left',
          verticalAlign: col.verticalAlign || 'middle',
          whiteSpace: col.wrap ? 'normal' : 'nowrap',
          ...getRowStyle(rowIdx),
        };
        // Apply gridlines
        const gl = bandingConfig.gridlines;
        if (gl.horizontal) {
          style.borderBottom = `${gl.width}px ${gl.style} ${gl.color}`;
        }
        if (gl.vertical) {
          style.borderRight = `${gl.width}px ${gl.style} ${gl.color}`;
        }
        // Apply bodyStyle padding
        const pad = col.bodyStyle;
        if (pad) {
          style.padding = `${pad.paddingTop ?? 4}px ${pad.paddingRight ?? 8}px ${pad.paddingBottom ?? 4}px ${pad.paddingLeft ?? 8}px`;
          if (pad.fontFamily) style.fontFamily = pad.fontFamily;
          if (pad.fontSize) style.fontSize = `${pad.fontSize}px`;
          if (pad.fontWeight) style.fontWeight = String(pad.fontWeight);
          if (pad.fontStyle) style.fontStyle = pad.fontStyle;
          if (pad.color) style.color = pad.color;
          if (pad.backgroundColor && pad.backgroundColor !== 'transparent') {
            style.backgroundColor = pad.backgroundColor;
          }
        }
        return style;
      };

      const getHeaderCellStyle = (col: ColumnConfig) => {
        const style: React.CSSProperties = {
          textAlign: col.align || 'left',
          verticalAlign: col.verticalAlign || 'middle',
          whiteSpace: col.wrap ? 'normal' : 'nowrap',
        };
        const gl = bandingConfig.gridlines;
        if (gl.horizontal) {
          style.borderBottom = `${gl.width}px ${gl.style} ${gl.color}`;
        }
        if (gl.vertical) {
          style.borderRight = `${gl.width}px ${gl.style} ${gl.color}`;
        }
        const pad = col.headerStyle;
        if (pad) {
          style.padding = `${pad.paddingTop ?? 6}px ${pad.paddingRight ?? 8}px ${pad.paddingBottom ?? 6}px ${pad.paddingLeft ?? 8}px`;
          if (pad.fontFamily) style.fontFamily = pad.fontFamily;
          if (pad.fontSize) style.fontSize = `${pad.fontSize}px`;
          if (pad.fontWeight) style.fontWeight = String(pad.fontWeight);
          if (pad.fontStyle) style.fontStyle = pad.fontStyle;
          if (pad.color) style.color = pad.color;
          if (pad.backgroundColor) style.backgroundColor = pad.backgroundColor;
        } else {
          if (bandingConfig.headerFill) style.backgroundColor = bandingConfig.headerFill;
          if (bandingConfig.headerTextColor) style.color = bandingConfig.headerTextColor;
        }
        return style;
      };

      const formatCellValue = (col: ColumnConfig, rawVal: unknown) => {
        const formatted = formatReportValue(rawVal, col.formatType);
        return `${col.formatPrefix || ''}${formatted}${col.formatSuffix || ''}`;
      };

      const hasSparkline = (col: ColumnConfig) => !!col.sparkline;

      const renderTotalsRow = (label: string, values: Array<{ value: number; col: ColumnConfig } | null>) => (
        <TableRow sx={{
          bgcolor: bandingConfig.totalsFill || 'rgba(0,212,255,0.06)',
          '& td': { color: bandingConfig.totalsTextColor || '#00D4FF', fontWeight: 700 },
        }}>
          <TableCell sx={{ color: bandingConfig.totalsTextColor || '#00D4FF', fontWeight: 700, fontStyle: 'italic' }}>
            {label}
          </TableCell>
          {values.map((entry, cIdx) => {
            if (!entry) return <TableCell key={cIdx} />;
            const { value, col } = entry;
            return (
              <TableCell key={col.id} style={getCellStyle(col, -1)}>
                {col.sparkline ? (
                  <SparklineInline
                    type={col.sparkline.type}
                    color={col.sparkline.color}
                    highColor={col.sparkline.highColor}
                    lowColor={col.sparkline.lowColor}
                    negativeColor={col.sparkline.negativeColor}
                    data={[3, 5, 2, 8, value / 1000, 6, 5, 7, 3, value / 1000]}
                  />
                ) : (
                  formatCellValue(col, value)
                )}
              </TableCell>
            );
          })}
        </TableRow>
      );

      const totalRowLabel = totalsConfig.grandTotal?.label || 'Grand Total';
      const isGrandTop = totalsConfig.grandTotal?.position === 'top';

      return (
        <Box
          onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); }}
          onDrop={(e) => {
            e.preventDefault();
            e.stopPropagation();
            try {
              const data = JSON.parse(e.dataTransfer.getData('application/json'));
              if ((data.type === 'bo-field-bundle' || data.type === 'bo-field') && (data.fields || data.field)) {
                const droppedList = data.fields || [data.field];
                onUpdate(id, {
                  properties: {
                    ...properties,
                    columns: [
                      ...columns,
                      ...droppedList.map((f: any) => ({
                        id: `col_${f.technicalName || f.name}_${Date.now()}`,
                        field: f.technicalName || f.name,
                        headerText: f.label || f.technicalName || f.name,
                        widthPx: 120,
                        visible: true,
                        headerStyle: {},
                        bodyStyle: {},
                        align: 'left',
                        verticalAlign: 'middle',
                        wrap: false,
                        formatType: 'Auto',
                        formatMask: '',
                        formatPrefix: '',
                        formatSuffix: '',
                      })),
                    ],
                  },
                });
              }
            } catch (err) {
              console.error('Drop error on table container:', err);
            }
          }}
          sx={{
            width: '100%',
            height: '100%',
            ...(pagination.mode === 'paginate' ? {
              overflow: 'auto',
              maxHeight: properties.freezePane?.frozenHeaderRows
                ? `${(properties.freezePane.frozenHeaderRows * 32) + (allDataRows.length * 32)}px`
                : undefined,
            } : {
              overflow: 'visible',
              maxHeight: undefined,
            }),
          }}
        >
          {pagination.mode === 'expand' ? (
            <Table size="small" stickyHeader={!!(properties.freezePane?.frozenHeaderRows)} sx={{
              '& td, & th': { fontSize: `${Number(properties.fontSize) || 11}px` },
            }}>
              <colgroup>
                {visibleColumns.map(col => (
                  <col key={col.id} style={{ width: col.widthPx || 120, minWidth: col.widthPx || 120 }} />
                ))}
              </colgroup>
              <TableHead>
                <TableRow sx={{ bgcolor: bandingConfig.headerFill || 'action.hover' }}>
                  {visibleColumns.map(col => (
                    <TableCell key={col.id} style={getHeaderCellStyle(col)}>
                      {col.headerText || col.field}
                    </TableCell>
                  ))}
                </TableRow>
              </TableHead>
              <TableBody>
                {isGrandTop && grandTotalRow && grandTotalRow.some(Boolean) && (
                  renderTotalsRow(totalRowLabel, grandTotalRow)
                )}

                {allDataRows.map((row: any, idx: number) => (
                  <TableRow key={idx} hover style={getRowStyle(idx)}>
                    {visibleColumns.map(col => {
                      const rawVal = row[col.field] ?? row[col.field?.toLowerCase?.()] ?? '-';
                      const hasSp = hasSparkline(col);
                      return (
                        <TableCell key={col.id} style={getCellStyle(col, idx)}>
                          {hasSp ? (
                            <SparklineInline
                              type={col.sparkline!.type}
                              color={col.sparkline!.color}
                              data={allDataRows.slice(0, 12).map(r => Number(r[col.field] ?? r[col.field?.toLowerCase?.()]) || 0)}
                            />
                          ) : (
                            formatCellValue(col, rawVal)
                          )}
                        </TableCell>
                      );
                    })}
                  </TableRow>
                ))}

                {!isGrandTop && grandTotalRow && grandTotalRow.some(Boolean) && (
                  renderTotalsRow(totalRowLabel, grandTotalRow)
                )}
              </TableBody>
            </Table>
          ) : (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
              {(() => {
                const perPage = pagination.rowsPerPage || 20;
                const totalChunks = Math.ceil(allDataRows.length / perPage);
                const chunks: any[][] = [];
                for (let i = 0; i < allDataRows.length; i += perPage) {
                  chunks.push(allDataRows.slice(i, i + perPage));
                }
                return chunks.map((chunk, chunkIdx) => {
                  const chunkPageTotal = pageTotalRows[chunkIdx] || [];
                  const isFirst = chunkIdx === 0;
                  const isLast = chunkIdx === totalChunks - 1;
                  const pageLabel = `Page ${chunkIdx + 1} of ${totalChunks}`;
                  return (
                    <Box key={chunkIdx}>
                      {!isFirst && (
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5 }}>
                          <Box sx={{ flex: 1, borderTop: '1px dashed rgba(0,212,255,0.3)' }} />
                          <Typography variant="caption" sx={{ color: 'rgba(0,212,255,0.5)', fontSize: '0.65rem', whiteSpace: 'nowrap' }}>
                            {pageLabel}
                          </Typography>
                          <Box sx={{ flex: 1, borderTop: '1px dashed rgba(0,212,255,0.3)' }} />
                        </Box>
                      )}
                      <Table size="small" sx={{
                        '& td, & th': { fontSize: `${Number(properties.fontSize) || 11}px` },
                        pageBreakInside: 'avoid',
                        breakInside: 'avoid',
                      }}>
                        <colgroup>
                          {visibleColumns.map(col => (
                            <col key={col.id} style={{ width: col.widthPx || 120, minWidth: col.widthPx || 120 }} />
                          ))}
                        </colgroup>
                        <TableHead>
                          <TableRow sx={{ bgcolor: bandingConfig.headerFill || 'action.hover' }}>
                            {visibleColumns.map(col => (
                              <TableCell key={col.id} style={getHeaderCellStyle(col)}>
                                {col.headerText || col.field}
                              </TableCell>
                            ))}
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {pagination.pageTotalEnabled && pagination.pageTotalPosition === 'top' && chunkPageTotal.length > 0 && (
                            renderTotalsRow(pagination.pageTotalLabel, chunkPageTotal)
                          )}

                          {chunk.map((row: any, idx: number) => (
                            <TableRow key={idx} hover style={getRowStyle(idx)}>
                              {visibleColumns.map(col => {
                                const rawVal = row[col.field] ?? row[col.field?.toLowerCase?.()] ?? '-';
                                const hasSp = hasSparkline(col);
                                return (
                                  <TableCell key={col.id} style={getCellStyle(col, idx)}>
                                    {hasSp ? (
                                      <SparklineInline
                                        type={col.sparkline!.type}
                                        color={col.sparkline!.color}
                                        data={allDataRows.slice(0, 12).map(r => Number(r[col.field] ?? r[col.field?.toLowerCase?.()]) || 0)}
                                      />
                                    ) : (
                                      formatCellValue(col, rawVal)
                                    )}
                                  </TableCell>
                                );
                              })}
                            </TableRow>
                          ))}

                          {pagination.pageTotalEnabled && pagination.pageTotalPosition === 'bottom' && chunkPageTotal.length > 0 && (
                            renderTotalsRow(pagination.pageTotalLabel, chunkPageTotal)
                          )}

                          {!pagination.pageTotalEnabled && isGrandTop && grandTotalRow && grandTotalRow.some(Boolean) && chunkIdx === 0 && (
                            renderTotalsRow(totalRowLabel, grandTotalRow)
                          )}

                          {!pagination.pageTotalEnabled && !isGrandTop && grandTotalRow && grandTotalRow.some(Boolean) && isLast && (
                            renderTotalsRow(totalRowLabel, grandTotalRow)
                          )}
                        </TableBody>
                      </Table>
                    </Box>
                  );
                });
              })()}
            </Box>
          )}
        </Box>
      );
    }

    if (type === ELEMENT_TYPES.FORM_REFERENCE) {
      const rowData = previewData && previewData.length > 0 ? previewData[0] : {};
      return (
        <FormEmbedShell
          element={{
            id,
            type: 'formReference',
            sectionId: '',
            templateId: properties.templateId ?? { isExpression: false, value: '' },
            containerStyle: properties.containerStyle ?? {},
            visibilityCondition: properties.visibilityCondition,
          }}
          formRegistry={formRegistry}
          rowData={rowData}
        />
      );
    }

    // Default: Text box or single field element
    let displayText = String(properties.text ?? 'Sample Text');

    if (isLivePreview) {
      if (previewData && previewData.length > 0) {
        const firstRow = previewData[0];
        const fieldKey = properties.fieldName;
        if (fieldKey && (firstRow[fieldKey] !== undefined || firstRow[fieldKey.toLowerCase()] !== undefined)) {
          const rawVal = firstRow[fieldKey] ?? firstRow[fieldKey.toLowerCase()];
          displayText = formatReportValue(rawVal, properties.formatType, properties.formatPrefix, properties.formatSuffix);
        } else if (properties.valueExpression) {
          const evalVal = evaluateReportExpression(properties.valueExpression, firstRow, displayText);
          displayText = formatReportValue(evalVal, properties.formatType, properties.formatPrefix, properties.formatSuffix);
        } else {
          displayText = evaluateReportExpression(properties.text, firstRow, displayText);
        }
      }
    }

    return (
      <Typography
        variant="body2"
        sx={textStyles}
      >
        {displayText}
      </Typography>
    );
  };

  return (
    <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }} transition={{ duration: 0.2 }}>
      <Rnd
        size={size}
        position={position}
        onDragStop={(_, d) => onUpdate(id, { position: { x: d.x, y: d.y } })}
        onResizeStop={(_, _direction, ref, _delta, nextPosition) => {
          onUpdate(id, {
            size: { width: ref.offsetWidth, height: ref.offsetHeight },
            position: nextPosition,
          });
        }}
        bounds="parent"
        onClick={() => onSelect(id)}
        enableUserSelectHack={false}
        tabIndex={0}
        disableDragging={isLivePreview}
        enableResizing={!isLivePreview}
        onKeyDown={(e: KeyboardEvent) => {
          if (!isLivePreview && e.key === 'Delete') onDelete(id);
          if (e.key === 'Enter') onSelect(id);
        }}
        aria-label={`Report element: ${type}`}
      >
        <Box
          sx={{
            ...containerStyles,
            border: isSelected && !isLivePreview ? '2px solid #6366f1' : containerStyles.border,
            position: 'relative',
            transition: 'border 0.15s ease',
            '&:hover': {
              border: !isLivePreview ? '1px solid #6366f1' : containerStyles.border,
              boxShadow: !isLivePreview ? '0 4px 20px rgba(99, 102, 241, 0.3)' : 'none',
            },
          }}
          onClick={(e) => {
            if (!isLivePreview) {
              e.stopPropagation();
              onSelect(id);
            }
          }}
        >
          {renderContent()}
          {isSelected && !isLivePreview && (
            <motion.div initial={{ scale: 0 }} animate={{ scale: 1 }} transition={{ type: 'spring', stiffness: 500, damping: 30 }}>
              <IconButton
                size="small"
                sx={{
                  position: 'absolute',
                  top: -10,
                  right: -10,
                  bgcolor: isDark ? '#ef4444' : '#dc2626',
                  color: '#ffffff',
                  '&:hover': { bgcolor: isDark ? '#dc2626' : '#b91c1c' },
                  width: 22,
                  height: 22,
                  zIndex: 1000,
                }}
                onClick={(e) => {
                  e.stopPropagation();
                  onDelete(id);
                }}
                aria-label="Delete element"
              >
                <DeleteIcon sx={{ fontSize: 12 }} />
              </IconButton>
            </motion.div>
          )}
        </Box>
      </Rnd>
    </motion.div>
  );
};

export default ReportElement;
