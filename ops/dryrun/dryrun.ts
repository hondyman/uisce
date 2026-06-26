import { Trino } from 'trino-client';
import { Kafka } from 'kafkajs';
import AWS from 'aws-sdk';
import fetch from 'node-fetch';

const trino = Trino.create({
    server: `http://${process.env.TRINO_HOST || 'localhost'}:${process.env.TRINO_PORT || 8080}`,
    catalog: process.env.TRINO_CATALOG || 'iceberg',
    schema: process.env.TRINO_SCHEMA || 'default'
});

const kafka = new Kafka({ brokers: [(process.env.KAFKA_BROKER || 'localhost:9092')] });
const s3 = new AWS.S3({
    endpoint: process.env.S3_ENDPOINT || 'http://localhost:9000',
    accessKeyId: process.env.S3_KEY || 'minioadmin',
    secretAccessKey: process.env.S3_SECRET || 'minioadmin',
    s3ForcePathStyle: true
});

const DRYRUN_BASELINE = process.env.DRYRUN_BASELINE_STORE;
const THRESHOLD_PCT = Number(process.env.DRYRUN_THRESHOLD_PCT || '0.005');
const THRESHOLD_ABS = Number(process.env.DRYRUN_THRESHOLD_ABS || '1000');
const S3_BUCKET = process.env.S3_BUCKET || 'data';
const DRYRUN_VAL_DATE = process.env.DRYRUN_VAL_DATE || new Date().toISOString().split('T')[0];

async function runQuery(sql: string) {
    const iter = await trino.query({ query: sql });
    const rows: any[] = [];
    for await (const result of iter) {
        if (result.data) {
            rows.push(...result.data);
        }
    }
    return rows;
}

async function fetchBaseline(termId: string, accountId: string): Promise<{ value: number }> {
    const key = `baselines/${termId}/${accountId}.json`;
    try {
        const obj = await s3.getObject({ Bucket: S3_BUCKET, Key: key }).promise();
        return JSON.parse(obj.Body!.toString()) as { value: number };
    } catch {
        return { value: 0 };
    }
}

async function callResolver(termId: string, accountId: string, valuationDate?: string) {
    const resolverUrl = process.env.RESOLVER_URL || 'http://localhost:9003';
    const body = { termId, entityType: 'Account', accountId, valuationDate, mode: 'preagg' };
    const res = await fetch(resolverUrl + '/resolve/holding', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body)
    });
    return res.json() as Promise<any>;
}

async function emitReport(reportPath: string, flagged: any[]) {
    const producer = kafka.producer();
    await producer.connect();
    await producer.send({
        topic: 'dryrun.report',
        messages: [{ value: JSON.stringify({ reportPath, flaggedCount: flagged.length }) }]
    });
    await producer.disconnect();
}

async function main() {
    const accountsSql = process.env.DRYRUN_TOP_ACCOUNTS_QUERY || "SELECT account_id FROM iceberg.default.holdings_preagg GROUP BY account_id LIMIT 10";
    console.log(`Running dry-run for accounts from query: ${accountsSql}`);

    let accounts: any[] = [];
    try {
        accounts = await runQuery(accountsSql);
    } catch (e) {
        console.error("Failed to fetch accounts:", e);
        process.exit(1);
    }

    const semanticTerms = JSON.parse(process.env.DRYRUN_TERMS_JSON || '["holding.market_value_resolved"]');

    const flagged: any[] = [];
    const results: any[] = [];

    for (const a of accounts) {
        const accountId = a.account_id || a.accountId || a[0];
        console.log(`Processing account: ${accountId}`);

        for (const termId of semanticTerms) {
            const baseline = await fetchBaseline(termId, accountId);
            let res;
            try {
                res = await callResolver(termId, accountId, DRYRUN_VAL_DATE);
            } catch (e) {
                console.error(`Resolver failed for ${termId}/${accountId}`, e);
                continue;
            }

            const newValue = Number(res.value || 0);
            const delta = newValue - (baseline.value || 0);
            const pct = baseline.value ? delta / baseline.value : (newValue === 0 ? 0 : 1); // 100% diff if new value exists but no baseline

            const record = {
                termId,
                accountId,
                baseline: baseline.value || 0,
                newValue,
                delta,
                pct,
                sql: res.sql,
                traceId: res.traceId,
                evaluatedAt: new Date().toISOString()
            };

            results.push(record);
            if ((pct !== null && Math.abs(pct) > THRESHOLD_PCT) || Math.abs(delta) > THRESHOLD_ABS) {
                flagged.push(record);
            }
        }
    }

    const reportKey = `dryruns/${new Date().toISOString()}.json`;
    try {
        await s3.putObject({ Bucket: S3_BUCKET, Key: reportKey, Body: JSON.stringify({ results, flagged }), ContentType: 'application/json' }).promise();
        console.log(`Report saved to s3://${S3_BUCKET}/${reportKey}`);

        await emitReport(`s3://${S3_BUCKET}/${reportKey}`, flagged);
    } catch (e) {
        console.error("Failed to upload report or emit event:", e);
    }

    console.log('Dry-run complete', { resultsCount: results.length, flaggedCount: flagged.length });
    if (flagged.length > 0) {
        console.log("Flagged Diffs Found:", JSON.stringify(flagged, null, 2));
        // Optionally exit non-zero if running in CI to block merge
        if (process.env.CI) {
            process.exit(1);
        }
    }
}

main().catch(err => { console.error(err); process.exit(1); });
