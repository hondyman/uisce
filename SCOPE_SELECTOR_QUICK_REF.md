# Quick Reference: Operating Scope Selection

## The Problem ❌
- You logged in as `global_ops`
- Went to Glossary page
- Saw no semantic terms
- No way to select a tenant/datasource

**Why?** Semantic terms require both a tenant AND datasource to be selected.

---

## The Solution ✅
We added a **scope selector** that prompts you to choose your operating scope.

---

## User Steps

```
1. Refresh browser               → Cmd+R or Ctrl+R
2. Go to Glossary page          → Click "Glossary" in sidebar
3. See empty state message      → "Select Operating Scope"
4. Click button                 → "Select Tenant & Datasource"
5. Choose scope:
   - Tenant: Uiscé
   - Instance: (select one)
   - Product: (select one)  
   - Datasource: Northwinds
6. View semantic terms          → ✨ All terms now visible!
```

---

## What You Can Do Now

With a scope selected:
- ✅ View all semantic terms for that tenant+datasource
- ✅ Create new semantic terms
- ✅ Edit existing terms
- ✅ Delete terms
- ✅ View business terms
- ✅ View calculation terms
- ✅ Switch to a different scope anytime

---

## Key Points

| Item | Details |
|------|---------|
| **Button Location** | Glossary page, when no scope selected |
| **Scope Persistence** | Your choice is saved in browser storage |
| **Scope Visibility** | Applies to all pages, not just Glossary |
| **Scope Change** | Click the scope badge at top-right to change |
| **Who Can Use** | Global ops and platform operators |

---

## Error States

| Error | Solution |
|-------|----------|
| Dialog won't open | Refresh page (`Cmd+R`) |
| No tenants in list | Check backend: `./docker-mac-local.sh logs backend` |
| Terms still empty | Try hard refresh: `Shift+Cmd+R` |
| Scope resets | Clear localStorage then re-select |

---

## Files Changed

- `frontend/src/pages/glossary/SemanticTermsTab.tsx` ← Only file modified

---

## Testing Checklist

- [ ] Refresh browser
- [ ] Navigate to Glossary
- [ ] See "Select Operating Scope" message
- [ ] Click "Select Tenant & Datasource" button
- [ ] Dialog opens showing tenants
- [ ] Select uisce → instance → product → northwinds
- [ ] Dialog closes automatically
- [ ] Semantic terms now visible
- [ ] Statistics show: Total, Mapped, Unmapped counts
- [ ] Can filter by mapping status
- [ ] Can create/edit/delete terms

---

Done! 🎉 Your scope selector is ready to use.
