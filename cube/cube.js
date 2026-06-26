
module.exports = {
  driverFactory: (driverContext) => {
    // driverContext.dataSource comes from the cube definition in the schema file
    if (driverContext.dataSource === 'starrocks') {
      return {
        type: 'mysql',
        host: process.env.CUBEJS_DB_STARROCKS_HOST,
        port: process.env.CUBEJS_DB_STARROCKS_PORT,
        user: process.env.CUBEJS_DB_STARROCKS_USER,
        password: process.env.CUBEJS_DB_STARROCKS_PASS,
        database: process.env.CUBEJS_DB_STARROCKS_DB,
      };
    }
    if (driverContext.dataSource === 'risingwave') {
      return {
        type: 'postgres',
        host: process.env.CUBEJS_DB_RISINGWAVE_HOST,
        port: process.env.CUBEJS_DB_RISINGWAVE_PORT,
        user: process.env.CUBEJS_DB_RISINGWAVE_USER,
        password: process.env.CUBEJS_DB_RISINGWAVE_PASS,
        database: process.env.CUBEJS_DB_RISINGWAVE_DB,
      };
    }
    
    // Default fallback or error if no datasource specified in schema
    // Assuming default might be one or the other, or throwing error.
    // For now we throw an error to be strict.
    throw new Error(`Unknown data source: ${driverContext.dataSource}. Please specify 'starrocks' or 'risingwave' in your Cube definition.`);
  },
};
