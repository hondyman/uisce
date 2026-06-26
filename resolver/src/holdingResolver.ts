import { v4 as uuidv4 } from 'uuid';
import { Trino } from 'trino-client';
import { Kafka } from 'kafkajs';

const trino = Trino.create({
    server: `http://${process.env.TRINO_HOST || 'localhost'}:${process.env.TRINO_PORT || 8080}`,
    catalog: process.env.TRINO_CATALOG || 'iceberg',
    schema: process.env.TRINO_SCHEMA || 'default'
});

const kafka = new Kafka({ brokers: [process.env.KAFKA_BROKER || 'localhost:9092'] });
const producer = kafka.producer();

export type ResolveMode = 'row' | 'preagg';

export async function resolveHoldingMarketValue(req: {
    termId: string;
    entityType: string;
    accountId?: string;
    valuationDate?: string;
    mode?: ResolveMode;
    traceId?: string;
}) {
    const traceId = req.traceId || `trc-${uuidv4()}`;
    const mode = req.mode || 'preagg';
    const filters: string[] = [];
    if (req.accountId) filters.push(`account_id = '${req.accountId}'`);
    if (req.valuationDate) filters.push(`valuation_date = DATE '${req.valuationDate}'`);
    const filterSql = filters.length ? filters.join(' AND ') : 'TRUE';

    if (mode === 'row') {
        const sql = `
      SELECT id, account_id, security_id, holding_type, market_value,
        CASE
          WHEN holding_type = 'SETTLED' THEN market_value
          WHEN holding_type = 'EOD' THEN market_value
          WHEN holding_type = 'SOD' THEN market_value
          ELSE market_value
        END AS market_value_resolved,
        valuation_date, settlement_date, currency, as_of_timestamp
      FROM iceberg.default.raw_holdings
      WHERE ${filterSql}
      LIMIT 200
    `;
        const rows = await runTrinoQuery(sql);
        const value = rows.reduce((s: number, r: any) => s + Number(r.market_value_resolved || 0), 0);
        await emitEvaluation({ termId: req.termId, entity: req.entityType, value, sql, rows_sample: rows.slice(0, 10), traceId });
        return { value, rows_sample: rows.slice(0, 10), sql, traceId };
    } else {
        const sql = `
      SELECT SUM(market_value_resolved) AS total_market_value
      FROM iceberg.default.holdings_preagg
      WHERE ${filterSql}
    `;
        const rows = await runTrinoQuery(sql);
        const value = rows.length ? Number(rows[0].total_market_value || 0) : 0;
        await emitEvaluation({ termId: req.termId, entity: req.entityType, value, sql, rows_sample: rows.slice(0, 10), traceId });
        return { value, sql, traceId };
    }
}

async function runTrinoQuery(sql: string): Promise<any[]> {
    const iter = await trino.query({
        query: sql,
        catalog: process.env.TRINO_CATALOG || 'iceberg',
        schema: process.env.TRINO_SCHEMA || 'default'
    });
    const rows: any[] = [];
    for await (const result of iter) {
        if (result.data) {
            rows.push(...result.data);
        }
    }
    return rows;
}

async function emitEvaluation(payload: any) {
    await producer.connect();
    await producer.send({
        topic: 'semantic.evaluations',
        messages: [{ value: JSON.stringify(payload) }]
    });
    await producer.disconnect();
}
