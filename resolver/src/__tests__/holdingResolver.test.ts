import { resolveHoldingMarketValue } from '../holdingResolver';

// Mock trino-client
jest.mock('trino-client', () => {
    return {
        Trino: {
            create: jest.fn().mockReturnValue({
                query: jest.fn().mockImplementation(async ({ query }) => {
                    // Mock async iterator return
                    return {
                        [Symbol.asyncIterator]: async function* () {
                            if (query.includes('holdings_preagg')) {
                                yield { data: [{ total_market_value: 123456 }] };
                            } else {
                                yield { data: [{ id: 'row1', market_value: 100, market_value_resolved: 100 }] };
                            }
                        }
                    };
                })
            })
        }
    };
});

// Mock Kafka
jest.mock('kafkajs', () => {
    return {
        Kafka: jest.fn().mockImplementation(() => ({
            producer: jest.fn().mockReturnValue({
                connect: jest.fn(),
                send: jest.fn(),
                disconnect: jest.fn()
            })
        }))
    };
});

jest.setTimeout(20000);

test('preagg mode returns a numeric value', async () => {
    const res = await resolveHoldingMarketValue({
        termId: 'holding.market_value_resolved',
        entityType: 'Account',
        accountId: 'A-001',
        valuationDate: '2026-01-02',
        mode: 'preagg'
    });
    expect(res).toHaveProperty('value');
    expect(res.value).toBe(123456);
});

test('row mode returns rows_sample', async () => {
    const res = await resolveHoldingMarketValue({
        termId: 'holding.market_value_resolved',
        entityType: 'Account',
        accountId: 'A-001',
        valuationDate: '2026-01-02',
        mode: 'row'
    });
    expect(res).toHaveProperty('rows_sample');
    expect(res.rows_sample).toHaveLength(1);
});
