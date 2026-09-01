import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen } from '@testing-library/react';
import { TotalsEditor } from '../../../../../src/components/reporting/tableStyling/TotalsEditor';
import type { ColumnConfig, TotalsConfig } from '../../../../../src/components/reporting/tableColumnModel';

const defaultTotals: TotalsConfig = {
  grandTotal: { enabled: false, position: 'bottom', label: 'Grand Total' },
  subtotals: { enabled: false, position: 'bottom', label: 'Total {groupValue}' },
};

function makeCol(id: string, formatType: ColumnConfig['formatType'] = 'Auto', aggregate?: ColumnConfig['aggregate']): ColumnConfig {
  return {
    id,
    field: `field_${id}`,
    headerText: `Field ${id}`,
    widthPx: 120,
    visible: true,
    headerStyle: {},
    bodyStyle: {},
    align: 'left',
    verticalAlign: 'middle',
    wrap: false,
    formatType,
    formatMask: '',
    formatPrefix: '',
    formatSuffix: '',
    aggregate,
  };
}

function getGrandTotalSwitch(): HTMLElement {
  return screen.getAllByRole('switch')[0];
}

describe('TotalsEditor', () => {
  describe('Grand Total toggle auto-aggregates numeric columns', () => {
    it('does not call onColumnsChange when grandTotal is toggled off', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const enabledTotals: TotalsConfig = {
        ...defaultTotals,
        grandTotal: { ...defaultTotals.grandTotal, enabled: true },
      };
      render(
        <TotalsEditor
          totals={enabledTotals}
          onChange={onChange}
          columns={[makeCol('a', 'Currency', { enabled: false, function: 'SUM', scope: 'column' })]}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());
      expect(onChange).toHaveBeenCalled();
      expect(onColumnsChange).not.toHaveBeenCalled();
    });

    it('auto-enables SUM aggregate on Currency column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('amount', 'Currency', { enabled: false, function: 'AVG', scope: 'column' })];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toEqual({ enabled: true, function: 'SUM', scope: 'column' });
    });

    it('auto-enables SUM aggregate on Integer column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('qty', 'Integer')];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toEqual({ enabled: true, function: 'SUM', scope: 'column' });
    });

    it('auto-enables SUM aggregate on Auto column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('amount', 'Auto')];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toEqual({ enabled: true, function: 'SUM', scope: 'column' });
    });

    it('auto-enables SUM aggregate on Percent column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('pct', 'Percent')];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toEqual({ enabled: true, function: 'SUM', scope: 'column' });
    });

    it('does NOT auto-enable aggregate on Date column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('date', 'Date')];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toBeUndefined();
    });

    it('does NOT auto-enable aggregate on Text column when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('name', 'Text')];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toBeUndefined();
    });

    it('does NOT overwrite already-enabled aggregate when Grand Total is toggled on', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const cols = [makeCol('amount', 'Currency', { enabled: true, function: 'AVG', scope: 'column' })];
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated[0].aggregate).toEqual({ enabled: true, function: 'AVG', scope: 'column' });
    });

    it('handles empty columns array gracefully', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      render(
        <TotalsEditor
          totals={defaultTotals}
          onChange={onChange}
          columns={[]}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onColumnsChange).toHaveBeenCalledTimes(1);
      const updated = onColumnsChange.mock.calls[0][0];
      expect(updated).toEqual([]);
    });

    it('toggles grandTotal off does NOT strip aggregates from columns', () => {
      const onChange = vi.fn();
      const onColumnsChange = vi.fn();
      const alreadyEnabledTotals: TotalsConfig = {
        ...defaultTotals,
        grandTotal: { ...defaultTotals.grandTotal, enabled: true },
      };
      const cols = [makeCol('amount', 'Currency', { enabled: true, function: 'SUM', scope: 'column' })];
      render(
        <TotalsEditor
          totals={alreadyEnabledTotals}
          onChange={onChange}
          columns={cols}
          onColumnsChange={onColumnsChange}
        />
      );
      fireEvent.click(getGrandTotalSwitch());

      expect(onChange).toHaveBeenCalled();
      expect(onColumnsChange).not.toHaveBeenCalled();
    });
  });
});
