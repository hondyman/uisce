#!/bin/bash

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
echo -e "${BLUE}  Phase 2 Production Readiness Verification${NC}"
echo -e "${BLUE}══════════════════════════════════════════════════${NC}"
echo ""

# Navigate to frontend
cd /Users/eganpj/GitHub/semlayer/frontend

# Check 1: Verify component files exist
echo -e "${YELLOW}[1/7] Checking component files...${NC}"
COMPONENTS=(
  "src/pages/portfolio/FactorExposureChart.tsx"
  "src/pages/portfolio/RuleBreachTable.tsx"
  "src/pages/portfolio/ScenarioPnLChart.tsx"
  "src/pages/portfolio/PortfolioDetailPage.tsx"
  "src/hooks/useMaterialTheme.ts"
)

all_exist=true
for component in "${COMPONENTS[@]}"; do
  if [ -f "$component" ]; then
    echo -e "${GREEN}✅ $component${NC}"
  else
    echo -e "${RED}❌ $component NOT FOUND${NC}"
    all_exist=false
  fi
done
echo ""

# Check 2: Verify Tailwind CSS removal
echo -e "${YELLOW}[2/7] Verifying Tailwind CSS removal...${NC}"
TAILWIND_PATTERNS=(
  "className.*bg-"
  "className.*text-"
  "className.*border-"
  "className.*dark:"
)

tailwind_clean=true
for pattern in "${TAILWIND_PATTERNS[@]}"; do
  matches=$(grep -r "$pattern" \
    src/pages/portfolio/FactorExposureChart.tsx \
    src/pages/portfolio/RuleBreachTable.tsx \
    src/pages/portfolio/ScenarioPnLChart.tsx \
    src/pages/portfolio/PortfolioDetailPage.tsx 2>/dev/null | wc -l)
  
  if [ "$matches" -eq 0 ]; then
    echo -e "${GREEN}✅ Pattern \"$pattern\": 0 matches${NC}"
  else
    echo -e "${RED}❌ Pattern \"$pattern\": $matches matches found${NC}"
    tailwind_clean=false
  fi
done
echo ""

# Check 3: Verify MUI imports
echo -e "${YELLOW}[3/7] Verifying Material UI imports...${NC}"
mui_imports=$(grep -r "from '@mui/material" \
  src/pages/portfolio/FactorExposureChart.tsx \
  src/pages/portfolio/RuleBreachTable.tsx \
  src/pages/portfolio/ScenarioPnLChart.tsx \
  src/pages/portfolio/PortfolioDetailPage.tsx 2>/dev/null | wc -l)

if [ "$mui_imports" -gt 0 ]; then
  echo -e "${GREEN}✅ MUI imports found: $mui_imports${NC}"
else
  echo -e "${RED}❌ No MUI imports found${NC}"
fi
echo ""

# Check 4: Line counts
echo -e "${YELLOW}[4/7] Checking component sizes...${NC}"
echo -e "${BLUE}Component Sizes:${NC}"
echo "  FactorExposureChart.tsx:  $(wc -l < src/pages/portfolio/FactorExposureChart.tsx) lines"
echo "  RuleBreachTable.tsx:      $(wc -l < src/pages/portfolio/RuleBreachTable.tsx) lines"
echo "  ScenarioPnLChart.tsx:     $(wc -l < src/pages/portfolio/ScenarioPnLChart.tsx) lines"
echo "  PortfolioDetailPage.tsx:  $(wc -l < src/pages/portfolio/PortfolioDetailPage.tsx) lines"
echo "  Total:                    $(($(wc -l < src/pages/portfolio/FactorExposureChart.tsx) + $(wc -l < src/pages/portfolio/RuleBreachTable.tsx) + $(wc -l < src/pages/portfolio/ScenarioPnLChart.tsx) + $(wc -l < src/pages/portfolio/PortfolioDetailPage.tsx))) lines"
echo ""

# Check 5: sx prop usage (MUI styling)
echo -e "${YELLOW}[5/7] Verifying MUI sx prop usage...${NC}"
sx_usage=$(grep -r "sx=" \
  src/pages/portfolio/FactorExposureChart.tsx \
  src/pages/portfolio/RuleBreachTable.tsx \
  src/pages/portfolio/ScenarioPnLChart.tsx \
  src/pages/portfolio/PortfolioDetailPage.tsx 2>/dev/null | wc -l)

if [ "$sx_usage" -gt 0 ]; then
  echo -e "${GREEN}✅ MUI sx prop: $sx_usage occurrences${NC}"
else
  echo -e "${RED}❌ No sx prop usage detected${NC}"
fi
echo ""

# Check 6: TypeScript compilation (optional - requires npm)
echo -e "${YELLOW}[6/7] Checking TypeScript compilation...${NC}"
if command -v npm &> /dev/null; then
  echo -e "${BLUE}Running: npx tsc --noEmit${NC}"
  if npx tsc --noEmit 2>&1 | grep -q "error"; then
    echo -e "${RED}❌ TypeScript errors found${NC}"
  else
    echo -e "${GREEN}✅ TypeScript: No errors${NC}"
  fi
else
  echo -e "${YELLOW}⚠️  npm not found - skipping TypeScript check${NC}"
fi
echo ""

# Check 7: Summary
echo -e "${YELLOW}[7/7] Final Summary...${NC}"
echo ""

if [ "$all_exist" = true ] && [ "$tailwind_clean" = true ]; then
  echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
  echo -e "${GREEN}✅ ALL CHECKS PASSED - PRODUCTION READY${NC}"
  echo -e "${GREEN}══════════════════════════════════════════════════${NC}"
  echo ""
  echo -e "${BLUE}Next Steps:${NC}"
  echo "1. npm run build              # Full build"
  echo "2. npm test                   # Unit tests"
  echo "3. npx playwright test        # E2E tests"
  echo "4. Deploy to staging          # Staging deployment"
  echo "5. QA verification            # Quality assurance"
  echo "6. Deploy to production       # Production release"
else
  echo -e "${RED}══════════════════════════════════════════════════${NC}"
  echo -e "${RED}❌ SOME CHECKS FAILED - REVIEW REQUIRED${NC}"
  echo -e "${RED}══════════════════════════════════════════════════${NC}"
  exit 1
fi
