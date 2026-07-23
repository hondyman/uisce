package bundles

import (
	"sort"
	"time"
)

// MineCandidates performs a tiny in-memory mining run over usage events and entitlements.
// This is a deterministic, simple PoC: it groups by co-occurrence across users.
func MineCandidates(ents map[string]Entitlement, events []UsageEvent, tenantID string, minUsage int) []CandidateBundle {
	// Build user -> set(entitlement)
	userMap := map[string]map[string]int{}
	for _, e := range events {
		if tenantID != "" && e.TenantID != tenantID {
			continue
		}
		if _, ok := ents[e.EntitlementID]; !ok {
			continue
		}
		if _, ok := userMap[e.UserID]; !ok {
			userMap[e.UserID] = map[string]int{}
		}
		userMap[e.UserID][e.EntitlementID] += e.Count
	}

	// cooccurrence counts
	co := map[string]map[string]int{}
	usageCount := map[string]int{}
	for _, u := range userMap {
		// filter low usage per user
		if len(u) == 0 {
			continue
		}
		// for each pair
		ids := []string{}
		for id, cnt := range u {
			if cnt <= 0 {
				continue
			}
			usageCount[id] += cnt
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for i := 0; i < len(ids); i++ {
			for j := i + 1; j < len(ids); j++ {
				a, b := ids[i], ids[j]
				if _, ok := co[a]; !ok {
					co[a] = map[string]int{}
				}
				co[a][b]++
				if _, ok := co[b]; !ok {
					co[b] = map[string]int{}
				}
				co[b][a]++
			}
		}
	}

	// naive clustering: for each entitlement, find top co-occurring partners to form bundle
	candidates := []CandidateBundle{}
	now := time.Now()
	used := map[string]bool{}
	for id, total := range usageCount {
		if total < minUsage {
			continue
		}
		// gather partners
		partners := []struct {
			id string
			c  int
		}{}
		if m, ok := co[id]; ok {
			for k, v := range m {
				partners = append(partners, struct {
					id string
					c  int
				}{id: k, c: v})
			}
		}
		sort.Slice(partners, func(i, j int) bool { return partners[i].c > partners[j].c })
		// include top N partners where cooccurrence >= 1
		bundleClaims := []string{id}
		for i := 0; i < len(partners) && i < 5; i++ {
			if partners[i].c <= 0 {
				break
			}
			bundleClaims = append(bundleClaims, partners[i].id)
		}

		// compute simple score
		score := 0.0
		for _, c := range bundleClaims {
			score += float64(usageCount[c])
		}

		// simple name/description generation
		name := "Bundle: " + ents[id].Name
		desc := "Access to " + ents[id].Resource + " / " + ents[id].Action

		cb := CandidateBundle{
			ID:          "cb_" + id,
			TenantID:    tenantID,
			Name:        name,
			Description: desc,
			Claims:      bundleClaims,
			Scope:       "tenant",
			Score:       score,
			Risk:        0.0,
			Explanations: map[string]string{
				"why": "Co-occurrence and usage-based grouping",
			},
			Status:    "candidate",
			CreatedAt: now,
		}
		// mark used to avoid near-duplicates
		dup := false
		for _, c := range bundleClaims {
			if used[c] {
				dup = true
				break
			}
		}
		if dup {
			// still include but lower score
			cb.Score = cb.Score * 0.6
		}
		candidates = append(candidates, cb)
		for _, c := range bundleClaims {
			used[c] = true
		}
	}

	// sort candidates by score desc
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })
	return candidates
}
