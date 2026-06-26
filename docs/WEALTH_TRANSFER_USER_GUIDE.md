# Wealth Transfer Platform - User Guide

## Getting Started

### For Advisors

#### Creating a New Family

1. Navigate to **Wealth Transfer** → **Families**
2. Click **"Create New Family Office"**
3. Fill in required fields:
   - Family Name
   - Legal Entity Name
   - Primary State
   - Estimated Net Worth
4. Click **"Create"**

#### Adding Family Members

1. Select a family from the list
2. Click **"Add Family Member"**
3. Enter member details:
   - Name, Date of Birth
   - Generation (1 = Grandparents, 2 = Parents, 3 = Children, 4 = Grandchildren)
   - Net Worth
   - Relationship (spouse, child)
4. Save

---

## Generating Estate Plans

### Automatic Plan Generation

1. Open a family's dashboard
2. Click **"Generate Estate Plan"**
3. Wait 30-60 seconds for AI processing
4. Review generated scenarios

### Understanding Scenarios

Each scenario shows:
- **Tax Savings**: Estimated estate tax reduction
- **Complexity**: Scale of 1-10 (1=simple, 10=complex)
- **Implementation Time**: Weeks to fully implement
- **Annual Cost**: Ongoing maintenance expenses

**Example Scenarios:**
1. **Annual Exclusion Gifting** (Simple, Low Cost)
   - Gift $37K/year to each child
   - Removes future appreciation from estate
   - Complexity: 2/10

2. **SLAT** (Moderate, Medium Cost)
   - Spousal Lifetime Access Trust
   - Uses lifetime exemption ($13.99M)
   - Preserves indirect access
   - Complexity: 6/10

3. **Dynasty Trust** (Complex, High Cost)
   - Multi-generational wealth preservation
   - Eliminates estate tax at every generation
   - Complexity: 8/10

---

## Gift Tracking

### Recording a Gift

1. Navigate to **Gift Tracking** tab
2. Click **"Record New Gift"**
3. Enter gift details:
   - Donor (family member)
   - Recipient (family member or external)
   - Gift value
   - Date
   - Asset description
4. System automatically calculates:
   - Annual exclusion used
   - Lifetime exemption used
   - Whether Form 709 is required

### Exemption Summary

View real-time tracking of:
- **Annual Exclusion**: $18,500/recipient/year ($37K if spousal split)
- **Lifetime Exemption**: $13.99M (2025)
- **GST Exemption**: $13.99M (2025)

**Example:**
```
Annual Exclusion:
  Used: $74,000 (to 2 children × $37K)
  Remaining: Unlimited (resets annually)

Lifetime Exemption:
  Used: $2,500,000
  Remaining: $11,490,000
```

---

##Trust Management

### Creating a Trust

1. Go to **Trusts & Entities** tab
2. Select trust type:
   - SLAT (Spousal Lifetime Access)
   - GRAT (Grantor Retained Annuity)
   - ILIT (Irrevocable Life Insurance)
   - Dynasty (Multi-generational)
   - Others (17 types supported)
3. Fill in trust details:
   - Grantor, Trustees, Beneficiaries
   - Formation date and state
   - Trust terms (JSONB customizable)
4. Save

### Compliance Monitoring

System automatically validates:
- ✓ Has designated trustee
- ✓ Has beneficiaries
- ✓ Has Tax ID (EIN) if irrevocable
- ✓ Tax filings current (Form 1041)
- ⚠ Alerts for overdue filings

---

## Tax Calculator

### Interactive "What-If" Analysis

Use the tax calculator to model different scenarios:

**Example: Estate Tax Calculation**
1. Enter gross estate: $25,000,000
2. Select state: New York
3. Enter prior exem ption used: $0
4. Click **"Calculate"**

**Result:**
```
Federal Estate Tax:
  Taxable Amount: $11,010,000
  Tax Due: $4,404,000
  Effective Rate: 17.6%

State Estate Tax (NY):
  Taxable Amount: $18,060,000
  Tax Due: $2,809,600
  Effective Rate: 11.2%

Total Tax: $7,213,600 (28.9% effective rate)
```

---

## Workflow Automation

### Form 709 Filing

System automatically:
1. Monitors gifts throughout the year
2. Identifies Form 709 requirements
3. Prepares form 60 days before April 15
4. Calculates gift tax owed
5. Generates PDF for review
6. (Optional) E-files with IRS

### Annual Plan Reviews

Scheduled annually, the system:
1. Checks for tax law changes
2. Detects family changes (births, deaths, marriages)
3. Monitors asset value changes (>10%)
4. Regenerates plan if needed
5. Notifies advisor of updates

---

## Family Tree Visualization

### Interactive Tree

Features:
- **Color-Coded Generations**: Purple (Gen 1) → Deep Purple (Gen 2) → Pink (Gen 3) → Blue (Gen 4)
- **Zoom Controls**: Zoom in/out, reset view
- **Click Nodes**: View detailed member profile
- **Net Worth Display**: Shows individual net worth

### Reading the Tree

- **Circles**: Family members
- **Lines**: Parent-child relationships
- **Size**: Proportional to net worth (optional)
- **Color Intensity**: Engagement score (darker = more engaged)

---

## Best Practices

### Annual Review Checklist

- [ ] Update member net worth estimates
- [ ] Review asset valuations
- [ ] Check compliance status for all trusts
- [ ] Review gift history for current year
- [ ] Update estate plan if major life events
- [ ] Schedule advisor meeting

### Red Flags to Watch

⚠ **Compliance Issues:**
- Trust tax filing overdue
- Missing trustee designation
- No EIN for irrevocable trust

⚠ **Tax Issues:**
- Approaching lifetime exemption limit
- Gifts exceeding annual exclusion not filed
- Estate above exemption threshold with no planning

⚠ **Family Issues:**
- Outdated beneficiary designations
- Disproportionate asset distribution
- Low engagement scores

---

## Keyboard Shortcuts

| Action | Shortcut |
|--------|----------|
| Generate Plan | `Ctrl+G` |
| New Gift | `Ctrl+N` |
| Search Families | `Ctrl+F` |
| Toggle Tree View | `Ctrl+T` |
| Export PDF | `Ctrl+E` |

---

## FAQ

**Q: How often should estate plans be reviewed?**  
A: Annually, or when major life events occur (birth, death, marriage, significant wealth change).

**Q: What's the difference between SLAT and Dynasty Trust?**  
A: SLAT provides spousal access and uses one generation's exemption. Dynasty Trust spans multiple generations and avoids estate tax at each level.

**Q: Can I combine multiple strategies?**  
A: Yes! The optimizer automatically suggests compatible combinations (e.g., Annual Gifting + SLAT = 10% synergy bonus).

**Q: How accurate are the tax savings estimates?**  
A: Estimates use current tax law and projected growth rates. Actual savings depend on future law changes, asset performance, and implementation quality.

**Q: What happens if tax law changes?**  
A: The system tracks tax law changes in real-time and automatically triggers plan reviews when significant changes occur.

---

## Support

For technical support:
- Email: support@example.com
- Phone: 1-800-XXX-XXXX
- Help Center: https://help.example.com

For advisor assistance:
- Schedule consultation: /book-meeting
- Training videos: /training
- Best practices guide: /best-practices
