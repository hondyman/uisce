import { graphql } from 'msw';

const rules = [
  {
    id: '1',
    name: '401(k) Contribution Limit',
    status: 'Active',
    benefitPlan: '401(k) Savings Plan',
    triggerEvent: 'New Hire Enrollment',
  },
  {
    id: '2',
    name: 'HSA Eligibility Check',
    status: 'Active',
    benefitPlan: 'High Deductible Health Plan',
    triggerEvent: 'Life Event Change',
  },
  {
    id: '3',
    name: 'Dependent Age Verification',
    status: 'Inactive',
    benefitPlan: 'Medical Plan',
    triggerEvent: 'Open Enrollment',
  },
  {
    id: '4',
    name: 'Spousal Surcharge Validation',
    status: 'Active',
    benefitPlan: 'Medical Plan',
    triggerEvent: 'Open Enrollment',
  },
  {
    id: '5',
    name: 'Vision Plan Waiting Period',
    status: 'Draft',
    benefitPlan: 'Vision Plan',
    triggerEvent: 'New Hire Enrollment',
  },
];

export const handlers = [
  graphql.query('GetValidationRules', (req, res, ctx) => {
    return res(
      ctx.data({
        validation_rules: rules.map(rule => ({
          ...rule,
          __typename: 'validation_rules',
          criteria: [],
        })),
      })
    );
  }),
];