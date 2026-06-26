# Relative Dates: Quick Reference & Examples

## What Are Relative Dates?

Relative dates automatically calculate date ranges based on when the rule executes, not fixed calendar dates. This keeps rules relevant without manual updates.

---

## Quick Reference Table

### Single-Day References

| Expression | Meaning | When It Changes |
|------------|---------|-----------------|
| `today` | Current day | Every midnight |
| `yesterday` | Previous day | Every midnight |
| `tomorrow` | Next day | Every midnight |

### Week/Month/Year

| Expression | Meaning | Starts | Ends | Use Case |
|------------|---------|--------|------|----------|
| `this week` | Current week (Mon-Sun) | Monday | Sunday | Weekly reports |
| `last week` | Previous week | Mon-1w | Sun-1w | Week-ago comparison |
| `this month` | Current month (1st-last) | 1st | Last day | Month-to-date |
| `last month` | Previous month | 1st-1m | Last-1m | Previous month analysis |
| `this year` | Current year (Jan 1 - Dec 31) | Jan 1 | Dec 31 | Year-to-date |
| `last year` | Previous year | Jan 1-1y | Dec 31-1y | Year-ago comparison |

### Lookback Periods (Past N Days)

| Expression | Meaning | Example |
|------------|---------|---------|
| `last 1 day` | Past 24 hours | Last 24 hours |
| `last 7 days` | Past week | This week's data |
| `last 14 days` | Two weeks | Recent activity |
| `last 30 days` | Monthly lookback | This month's data |
| `last 60 days` | Two months | Recent trends |
| `last 90 days` | Quarter | Quarterly analysis |
| `last 365 days` | Annual lookback | Year-ago comparison |

### Specific Days in Past

| Expression | Meaning | Represents |
|------------|---------|------------|
| `today` | Current day | Today |
| `1 day ago` | Yesterday | Yesterday |
| `2 days ago` | Day before yesterday | 48 hours back |
| `7 days ago` | Exactly 1 week back | 1 week ago |
| `30 days ago` | Exactly 1 month back | 1 month ago |
| `90 days ago` | Exactly 3 months back | 3 months ago |
| `N days ago` | N days in the past | Flexible |

### Specific Weeks/Months in Past

| Expression | Meaning |
|------------|---------|
| `1 week ago` | Exactly 1 week back |
| `2 weeks ago` | Exactly 2 weeks back |
| `1 month ago` | Exactly 1 month back |
| `3 months ago` | Exactly 3 months back |
| `6 months ago` | Exactly 6 months back |
| `1 year ago` | Exactly 1 year back |

### Day of Week (Recurring)

| Expression | Meaning | Use Case |
|------------|---------|----------|
| `Monday` | All Mondays | Weekly rollup |
| `Tuesday` | All Tuesdays | Weekly pattern |
| `Wednesday` | All Wednesdays | Weekly pattern |
| `Thursday` | All Thursdays | Weekly pattern |
| `Friday` | All Fridays | End-of-week |
| `Saturday` | All Saturdays | Weekend data |
| `Sunday` | All Sundays | Weekend data |

### Time-Based Ranges (Advanced)

| Expression | Meaning | Example |
|------------|---------|---------|
| `N days ago for M days` | Range in past | `7 days ago for 7 days` = 2 weeks ago |
| `N weeks ago for M weeks` | Week range | `2 weeks ago for 4 weeks` = back to 6 weeks |
| `N months ago for M months` | Month range | `1 month ago for 3 months` = 3-1 months back |

---

## Real-World Examples

### Daily Reports
```
Field: transaction_date
Value: today
→ Only today's transactions
```

### Weekly Dashboards
```
Field: created_at
Value: this week
→ All transactions from Monday-Sunday this week
```

### Monthly Metrics
```
Field: order_date
Value: this month
→ Month-to-date sales
```

### Comparative Analysis
```
Condition 1: order_date = last 30 days
Condition 2: status = pending
→ Pending orders from last month
```

### Quarterly Business Reviews
```
Field: close_date
Value: last 90 days
→ Deals closed in last quarter
```

### Year-Over-Year Analysis
```
Condition 1: transaction_date = last 365 days
Condition 2: amount > [10000,999999]
→ Large transactions in past year
```

### Lookback with Department
```
Condition 1: hire_date = last 180 days
Condition 2: department = Engineering,Product
→ New hires in Engineering/Product from last 6 months
```

### Exclude Old Records
```
Condition 1: updated_at = last 90 days
Condition 2: status = -archived
→ Recently updated, non-archived records
```

---

## How Relative Dates Work

### Execution Time vs. Fixed Date

**Fixed date (❌ NOT recommended):**
```
created_at > 2024-01-01
→ After Jan 1, 2024 (becomes stale)
```

**Relative date (✅ RECOMMENDED):**
```
created_at in last 365 days
→ Past 365 days (always current)
```

### Automatic Recalculation

Relative dates recalculate every time a rule runs:

```
Today:    Dec 15, 2024
Rule:     last 7 days
Range:    Dec 9 - Dec 15, 2024

Tomorrow: Dec 16, 2024
Rule:     last 7 days (auto-updates)
Range:    Dec 10 - Dec 16, 2024 ← Automatically shifted
```

### Timezone Considerations

Relative dates use server timezone (usually UTC):
- `today` = UTC midnight to UTC midnight
- `this week` = UTC Monday to UTC Sunday
- `this month` = UTC 1st to UTC last day

---

## Common Patterns

### Weekly Recurring Task
```
Field: task_date
Operator: Relative Dates
Value: Monday
→ Every Monday (same day of week)
```

### Monthly Reconciliation
```
Field: reconciliation_date
Operator: Relative Dates
Value: this month
→ Month-to-date (first to last day)
```

### Quarterly Review
```
Field: review_date
Operator: Relative Dates
Value: last 90 days
→ Last 90 calendar days
```

### Seasonal Comparison
```
Field: sales_date
Operator: Relative Dates
Value: 3 months ago for 1 month
→ Same month 3 months ago
```

---

## Performance Tips

✓ **Use relative dates for:**
- Rules that run regularly (daily, weekly, monthly)
- Time-based analysis without manual updates
- Comparative analysis (now vs. same period ago)

✓ **Relative date ranges recalculate:**
- At query execution time (not storage time)
- Server-side by your backend
- Automatically, no manual intervention needed

⚠️ **Note:** Your backend must support relative date parsing. See [Backend Integration] for implementation details.

---

## Troubleshooting

### Issue: Relative Date Not Updating
**Check:** Does your rule execute daily? Relative dates only update on query.

**Solution:** If you need updates more frequently, have your system run checks hourly/4x daily.

### Issue: Yesterday's Data Included in "Last 7 Days"
**Check:** "Last 7 days" can mean different things:
- **Inclusive:** Last 7 calendar days (including today)
- **Exclusive:** Last 7 days (not including today)

**Solution:** Use "7 days ago" for specific day or check backend implementation.

### Issue: Sunday Included in "This Week"
**Check:** Different systems define week start:
- **US/UK:** Monday start (Mon-Sun)
- **Middle East:** Saturday start (Sat-Fri)

**Solution:** Be explicit or configure your system's week definition.

### Issue: Month Boundary Issues
**Check:** Months have different lengths (28-31 days)

**Example:** If you set a rule on Jan 30:
- Feb 28/29 is last day of month (not Feb 30)

**Solution:** Use "this month" for automatic handling or "30 days ago" for fixed periods.

---

## Edge Cases & Gotchas

### 1. Leap Years
```
2024 = leap year (366 days)
2025 = normal year (365 days)

Rule: last 365 days
→ Different ranges each year
```

**Solution:** Use "last 12 months" for fixed month count.

### 2. Daylight Saving Time
```
March: Clock moves forward (+1 hour)
November: Clock moves back (-1 hour)

Relative dates may shift by 1 hour
```

**Solution:** Use calendar-based (this month) instead of time-based (last 7 days) for DST periods.

### 3. Timezone Differences
```
Server: UTC
Client: PST (8 hours behind)

"today" in UTC ≠ "today" in PST
```

**Solution:** All relative dates calculated server-side in UTC; adjust if needed.

### 4. Month-End Data
```
Condition: this month
On Jan 31:
→ Jan 1 - Jan 31 (full month)

On Feb 28 (non-leap):
→ Feb 1 - Feb 28 (short month)
```

**Solution:** Use "last 30 days" for consistent 30-day windows.

---

## API Integration

### Sending Date Conditions
```typescript
const condition = {
  field: 'created_at',
  operator: 'relative_dates',
  value: 'last 7 days'  // Raw expression sent to backend
};
```

### Backend Must Parse
```python
# Pseudocode: Backend interpretation
if value == 'last 7 days':
    end_date = today
    start_date = today - 7 days
    filter: created_at BETWEEN start_date AND end_date

elif value == 'this month':
    start_date = first day of current month
    end_date = last day of current month
    filter: created_at BETWEEN start_date AND end_date
```

### Multiple Conditions
```typescript
conditions: [
  {
    field: 'hire_date',
    operator: 'relative_dates',
    value: 'last 90 days'
  },
  {
    field: 'salary',
    operator: 'expressions',
    value: '[75000,150000]'
  }
]
// Result: Hired in last 90 days AND salary 75k-150k
```

---

## Best Practices

### ✓ DO:
- Use relative dates for all time-based rules
- Document which timezone your system uses
- Test rule boundaries (month-end, year-end)
- Use "this month" for monthly data
- Use "this week" for weekly data
- Use "last N days" for lookback periods

### ✗ DON'T:
- Hardcode fixed dates (they become stale)
- Mix relative and absolute dates in one rule
- Assume specific timezone without checking
- Use relative dates without backend support
- Forget that months have different lengths

---

## Summary

Relative dates keep your rules **always current** without manual maintenance:
- ✓ Automatic date range calculation
- ✓ No manual updates needed
- ✓ Perfect for recurring rules
- ✓ Supports daily, weekly, monthly, quarterly analysis
- ✓ Real-time validation and feedback

**Start using relative dates in your rules today!**
