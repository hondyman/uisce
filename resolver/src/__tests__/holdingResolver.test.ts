import { resolveHoldingMarketValue } from '../holdingResolver';

// Mock pg
jest.mock('pg', () => {
    const mockQuery = jest.fn();
    return {
        Pool: jest.fn().mockImplementation(() => ({
            query: mockQuery.mockResolvedValue({
                rows: [{ total_market_value: 123456 }]
            })
        }))
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
