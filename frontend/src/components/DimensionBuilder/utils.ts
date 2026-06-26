// src/components/DimensionBuilder/utils.ts
import { Dimension } from './types';

export const generateDimensionCode = (dimension: Dimension): string => {
  let code = `${dimension.name}: {
  sql: \`${dimension.sql}\`,
  type: \`${dimension.type}\``;

  // Add optional properties
  if (dimension.title) code += `,\n  title: \`${dimension.title}\``;
  if (dimension.description) code += `,\n  description: \`${dimension.description}\``;
  if (dimension.format) code += `,\n  format: \`${dimension.format}\``;
  if (dimension.primary_key) code += `,\n  primary_key: ${dimension.primary_key}`;
  if (dimension.public === false) code += `,\n  public: false`;
  if (dimension.sub_query) code += `,\n  sub_query: true`;
  if (dimension.propagate_filters_to_sub_query) code += `,\n  propagate_filters_to_sub_query: true`;

  // Add case statement
  if (dimension.case) {
    code += `,\n  case: {\n    when: [\n`;
    dimension.case.when.forEach(w => {
      const label = typeof w.label === 'string' ? `\`${w.label}\`` : `{ sql: \`${w.label.sql}\` }`;
      code += `      { sql: \`${w.sql}\`, label: ${label} },\n`;
    });
    code += `    ],\n`;
    const elseLabel = typeof dimension.case.else.label === 'string' 
      ? `\`${dimension.case.else.label}\`` 
      : `{ sql: \`${dimension.case.else.label.sql}\` }`;
    code += `    else: { label: ${elseLabel} }\n  }`;
  }

  // Add granularities
  if (dimension.granularities && dimension.granularities.length > 0) {
    code += `,\n  granularities: {\n`;
    dimension.granularities.forEach(g => {
      code += `    ${g.name}: {\n      interval: \`${g.interval}\``;
      if (g.offset) code += `,\n      offset: \`${g.offset}\``;
      if (g.origin) code += `,\n      origin: \`${g.origin}\``;
      if (g.title) code += `,\n      title: \`${g.title}\``;
      code += `\n    },\n`;
    });
    code += `  }`;
  }

  // Add meta information
  if (dimension.meta && Object.keys(dimension.meta).length > 0) {
    code += `,\n  meta: ${JSON.stringify(dimension.meta, null, 2)}`;
  }

  code += `\n}`;
  return code;
};

export const generateAllDimensionsCode = (dimensions: Dimension[]): string => {
  if (dimensions.length === 0) return '';
  
  const dimensionCodes = dimensions.map(generateDimensionCode);
  
  return `// Cube.js dimensions configuration
dimensions: {
${dimensionCodes.map(code => `  ${code.replace(/\n/g, '\n  ')}`).join(',\n\n')}
}`;
};
