const fs = require('fs');
const { FileRepository } = require('@cubejs-backend/server-core');
const path = require('path');

module.exports = {
  // Explicitly configure file repository to point to model directory
  repositoryFactory: () => {
    const modelPath = path.join(process.cwd(), 'model');
    
    // Debug: Check directory exists and contents
    console.log('=== CUBE DEBUG START ===');
    console.log('Model path:', modelPath);
    console.log('Directory exists:', fs.existsSync(modelPath));
    
    if (fs.existsSync(modelPath)) {
      const files = fs.readdirSync(modelPath);
      console.log('Files in model directory:', files);
      
      files.forEach(file => {
        const filePath = path.join(modelPath, file);
        try {
           const stats = fs.statSync(filePath);
           fs.accessSync(filePath, fs.constants.R_OK); 
           console.log(`File: ${file}, Size: ${stats.size}, Readable: true`);
        } catch (e) {
           console.log(`File: ${file}, Readable: false (${e.message})`);
        }
      });
    } else {
      console.log('Model directory does not exist!');
    }
    
    console.log('=== CUBE DEBUG END ===');
    
    const repo = new FileRepository(modelPath);
    
    // Bypass FileRepository logic and manually read files
    repo.dataSchemaFiles = async () => {
      console.log('=== CALLING MANUAL dataSchemaFiles ===');
      try {
        const files = fs.readdirSync(modelPath)
          .filter(f => f.endsWith('.js') || f.endsWith('.yaml') || f.endsWith('.yml'))
          .map(file => {
             const content = fs.readFileSync(path.join(modelPath, file), 'utf-8');
             console.log(`Loading file: ${file} (${content.length} bytes)`);
             return {
               fileName: file,
               content: content
             };
          });
        return files;
      } catch (e) {
        console.log('=== MANUAL dataSchemaFiles ERROR ===', e);
        throw e;
      }
    };

    return repo;
  },

  // Force schema recompilation on every request (dev mode)
  schemaVersion: () => `${Date.now()}`,

  // Multi-source configuration
  driverFactory: ({ dataSource }) => {
    if (dataSource === 'trino' || dataSource === 'default') {
      const { TrinoDriver } = require('@cubejs-backend/trino-driver');
      return new TrinoDriver({
        host: process.env.CUBEJS_DS_TRINO_HOST,
        port: process.env.CUBEJS_DS_TRINO_PORT,
        catalog: process.env.CUBEJS_DS_TRINO_CATALOG,
        schema: process.env.CUBEJS_DS_TRINO_SCHEMA,
        user: process.env.CUBEJS_DS_TRINO_USER || 'admin',
        password: process.env.CUBEJS_DS_TRINO_PASSWORD,
      });
    }

    throw new Error(`Unknown data source: ${dataSource}`);
  },

  // Optional: Add logging to see which files are being picked up
  logger: (msg, params) => {
    if (msg === 'Schema Management') {
      console.log(`[Cube Schema] ${JSON.stringify(params)}`);
    } else if (process.env.DEBUG_LOG === 'true') {
       console.log(`[Cube Log] ${msg}:`, params);
    }
  },

  // Enforce Tenant Isolation
  checkAuth: (req, auth) => {
     if (!auth) {
       throw new Error('Authentication required');
     }
     const ctx = auth.u || auth;
     // Support both snake_case and camelCase
     const tenantId = ctx.tenant_id || ctx.tenantId;
     
     if (!tenantId) {
        throw new Error('Tenant Isolation Error: Tenant ID is missing in security context');
     }
  },

  queryRewrite: (query, { securityContext }) => {
    const ctx = securityContext.u || securityContext;
    
    // Apply dynamic row filters provided by backend SecurityService
    if (ctx.rowFilters) {
      if (!query.filters) {
        query.filters = [];
      }
      
      // rowFilters is map<CubeName, Filter[]>
      Object.keys(ctx.rowFilters).forEach(cubeName => {
        const filters = ctx.rowFilters[cubeName];
        filters.forEach(f => {
           // If query targets this cube or wildcard, apply filter
           // Note: This is a simplified check. In strict mode, we should check query.measures/dimensions
           
           // Convert backend RowFilter to Cube.js filter format
           query.filters.push({
             member: f.dimension, // Assumes fully qualified name like "Cube.dim"
             operator: f.operator,
             values: f.values
           });
        });
      });
      console.log(`[Isolation] Applied ${Object.keys(ctx.rowFilters).length} context filters`);
    }
    
    return query;
  }
};
