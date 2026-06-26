// Utility functions for semantic model pre-aggregation handling

export const replacePreAggregationPlaceholders = (
  template: any,
  semanticModel: any,
  selectedColumn: any
) => {
  const replaced = { ...template };

  if (replaced.measures) {
    replaced.measures = replaced.measures.map((measure: string) => {
      if (measure.includes('<replace_with_cashflow_measure>')) {
        const cashFlowMeasures = semanticModel.measures?.filter((m: any) =>
          m.name.toLowerCase().includes('cash') ||
          m.name.toLowerCase().includes('flow') ||
          m.name.toLowerCase().includes('amount') ||
          m.name.toLowerCase().includes('value')
        );
        return cashFlowMeasures?.length > 0 ? cashFlowMeasures[0].name : 'total_amount';
      }
      return measure;
    });
  }

  if (replaced.dimensions) {
    replaced.dimensions = replaced.dimensions.map((dimension: string) => {
      if (dimension.includes('<replace_with_grouping_dimension>')) {
        const groupingDimensions = semanticModel.dimensions?.filter((d: any) =>
          d.name.toLowerCase().includes('id') ||
          d.name.toLowerCase().includes('group') ||
          d.name.toLowerCase().includes('category') ||
          d.name.toLowerCase().includes('portfolio') ||
          d.name.toLowerCase().includes('investment')
        );
        return (
          selectedColumn?.column?.name ||
          (groupingDimensions?.length > 0 ? groupingDimensions[0].name : 'id')
        );
      }
      return dimension;
    });
  }

  if (replaced.timeDimension && replaced.timeDimension.includes('<replace_with_')) {
    const timeDimensions = semanticModel.dimensions?.filter((d: any) =>
      d.name.toLowerCase().includes('date') ||
      d.name.toLowerCase().includes('time') ||
      d.type === 'time'
    );
    replaced.timeDimension = timeDimensions?.length > 0 ? timeDimensions[0].name : 'created_at';
  }

  return replaced;
};

export const isPreAggregationNeeded = (preAgg: any, semanticModel: any) => {
  if (!preAgg.is_template) return true;
  const measuresUsingPreAgg = semanticModel.measures?.filter((measure: any) =>
    measure.sql?.includes(preAgg.name) ||
    measure.sql?.includes(preAgg.measures?.join('')) ||
    measure.sql?.includes(preAgg.dimensions?.join(''))
  );
  return measuresUsingPreAgg?.length > 0;
};

export const cleanupUnusedPreAggregations = (semanticModel: any) => {
  if (!semanticModel.pre_aggregations) return semanticModel;
  return {
    ...semanticModel,
    pre_aggregations: semanticModel.pre_aggregations.filter((preAgg: any) =>
      isPreAggregationNeeded(preAgg, semanticModel)
    ),
  };
};
