package wealth.suitability

default suitable = false

# Inputs:
# client.risk_profile: "conservative" | "moderate" | "aggressive"
# proposal.exposures_after: map sector -> weight
# rules.max_concentration: { sector: max_weight }

suitable {
  not violates_concentration
}

violates_concentration {
  some sector
  w := input.proposal.exposures_after[sector]
  max := input.rules.max_concentration[sector]
  w > max
}
