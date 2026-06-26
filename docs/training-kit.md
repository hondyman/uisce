# SemLayer User Training Kit

## Overview
This training kit provides materials for different user personas to effectively use SemLayer's governance and conversational features.

## Table of Contents
1. [Video Scripts](#video-scripts)
2. [Cheat Sheets](#cheat-sheets)
3. [Hands-on Labs](#hands-on-labs)
4. [Quick Reference Guides](#quick-reference-guides)

---

## Video Scripts

### Video 1: Requesting Access and JIT Add-ons (3-5 minutes)

#### Script Outline

**Introduction (0:00-0:30)**
- Welcome to SemLayer access management
- Overview: Self-service access with governance guardrails
- Learning objectives: How to request access, understand approvals, use JIT features

**Section 1: The Access Request Process (0:30-2:00)**

*[Visual: Login to SemLayer portal]*

"Let's walk through requesting access to data you need for your analysis.

**Step 1: Navigate to Access Requests**
- Click on 'Access' in the main navigation
- Select 'Request New Access'

**Step 2: Describe What You Need**
- Choose the data asset or category
- Specify the type of access (read, write, analyze)
- Provide business justification

*[Visual: Fill out request form]*
'Be specific about why you need this data. Good justifications include:
- "Analyzing Q4 sales performance for my region"
- "Building customer segmentation model"
- "Creating compliance reporting dashboard"

**Step 3: Review Automated Checks**
- System checks for policy compliance
- May suggest alternatives or micro-bundles
- Addresses potential SoD conflicts

**Step 4: Submit and Track**
- Submit request
- Receive immediate feedback
- Track status in 'My Requests'"

**Section 2: Understanding JIT Add-ons (2:00-3:00)**

*[Visual: JIT approval scenario]*

"Just-in-Time access provides temporary permissions for specific tasks.

**When JIT is Offered:**
- Time-sensitive analysis
- One-off reporting needs
- Emergency access requirements

**How JIT Works:**
1. Request triggers JIT evaluation
2. System creates temporary permission bundle
3. Access granted for limited time (hours/days)
4. Automatic expiration and audit trail

**Best Practices:**
- Use JIT for short-term needs
- Combine with regular access requests for ongoing work
- Always document the business purpose"

**Conclusion (3:00-3:30)**
- Summary of key points
- Resources for additional help
- Contact information for support

---

### Video 2: Reading Access Decisions and 'Why?' Explanations (4 minutes)

#### Script Outline

**Introduction (0:00-0:30)**
- Understanding access decisions
- The importance of transparency
- What you'll learn: Reading decisions, understanding 'why?', next steps

**Section 1: Access Request Outcomes (0:30-1:30)**

*[Visual: Different decision screens]*

**Approved Requests:**
- Green checkmark with approval details
- Access granted immediately or scheduled
- Next steps for using the access

**Modified Requests:**
- Yellow warning icon
- Explanation of changes made
- Alternative suggestions provided

**Rejected Requests:**
- Red X with detailed reasoning
- Alternative approaches suggested
- Appeal process explained

**Pending Requests:**
- Clock icon with estimated timeline
- Current review stage
- Contact information for follow-up

**Section 2: Understanding the 'Why?' (1:30-3:00)**

*[Visual: Expandable 'Why?' section]*

"Every decision includes detailed reasoning:

**Policy Information:**
- Which policy rule was applied
- Risk level assessment
- Compliance requirements

**Context Factors:**
- Your role and current permissions
- Data sensitivity classification
- Usage patterns and history

**Alternatives Suggested:**
- Different data sources
- Micro-bundle options
- JIT access possibilities

**Appeal Process:**
- When and how to appeal
- Required additional information
- Escalation paths"

**Section 3: Taking Action (3:00-3:45)**

*[Visual: Action buttons and next steps]*

**For Approvals:**
- Start using the access
- Bookmark for future use
- Set up notifications

**For Rejections:**
- Review suggested alternatives
- Modify and resubmit if appropriate
- Contact steward for clarification

**For All Decisions:**
- Provide feedback on the decision
- Save the explanation for reference
- Learn from the reasoning"

**Conclusion (3:45-4:00)**
- Key takeaways
- Additional resources
- Support contact information

---

## Cheat Sheets

### Access Request Checklist

#### Before Submitting
- [ ] **Clear Business Purpose**: Explain why you need this data
- [ ] **Right Scope**: Request only what you need
- [ ] **Time Frame**: Specify when you need access
- [ ] **Alternatives Considered**: Have you explored other options?

#### During Request
- [ ] **Accurate Information**: Double-check asset names and permissions
- [ ] **Complete Justification**: Provide specific use case
- [ ] **Contact Details**: Ensure correct contact information
- [ ] **Urgency Level**: Set appropriate priority

#### After Submission
- [ ] **Track Status**: Monitor request progress
- [ ] **Prepare Questions**: Note any clarification needs
- [ ] **Plan Next Steps**: What to do once approved

### Governance "Dos and Don'ts"

#### ✅ Dos
- **Do** provide detailed business justifications
- **Do** request minimal necessary access
- **Do** use JIT for temporary needs
- **Do** appeal decisions with additional context
- **Do** follow up on pending requests
- **Do** report suspicious access patterns

#### ❌ Don'ts
- **Don't** request access without business need
- **Don't** share credentials or access
- **Don't** bypass approval processes
- **Don't** ignore security warnings
- **Don't** use personal accounts for business data
- **Don't** store sensitive data inappropriately

### NL Query Phrasing Patterns

#### Basic Queries
```
Show me [metric] by [dimension]
What is the [metric] for [entity]?
How many [entities] have [condition]?
```

#### Time-based Queries
```
Show me [metric] over the last [time period]
Compare [metric] between [time1] and [time2]
What is the trend for [metric] in [time period]?
```

#### Comparative Queries
```
Compare [metric] between [group1] and [group2]
Show me the difference in [metric] by [dimension]
Which [entities] have the highest/lowest [metric]?
```

#### Conditional Queries
```
Show me [entities] where [condition]
Find [entities] with [metric] > [value]
List [entities] that [criteria]
```

#### Examples
- ✅ "Show me sales by region for the last quarter"
- ✅ "What is the customer churn rate by product category?"
- ✅ "Compare revenue growth between Q1 and Q2"
- ❌ "Give me all the data" (too broad)
- ❌ "Show me everything about customers" (insufficient specificity)

---

## Hands-on Labs

### Lab 1: Access Request Workflow

#### Objective
Complete a full access request cycle and understand the approval process.

#### Prerequisites
- SemLayer account with basic permissions
- Access to sample data environment

#### Steps

**Step 1: Prepare Your Request**
1. Identify a specific business need
2. Research available data assets
3. Determine appropriate permission level

**Step 2: Submit Request**
1. Navigate to Access → Request New Access
2. Fill out the request form:
   - Select target data asset
   - Choose permission type
   - Provide detailed justification
   - Set appropriate urgency

**Step 3: Handle Initial Response**
1. Review automated system response
2. Note any suggested modifications
3. Address any additional information requests

**Step 4: Monitor and Follow Up**
1. Check request status regularly
2. Respond to any steward questions
3. Prepare for access activation

**Step 5: Use Approved Access**
1. Access the approved data
2. Perform your intended analysis
3. Document results and insights

#### Success Criteria
- [ ] Request submitted with complete information
- [ ] Understood system feedback and suggestions
- [ ] Successfully used approved access
- [ ] Provided feedback on the process

### Lab 2: NL Query with Guardrails

#### Objective
Learn to write effective natural language queries and handle guardrail responses.

#### Prerequisites
- Approved access to sample datasets
- Understanding of basic query concepts

#### Steps

**Step 1: Start with Simple Queries**
1. Access the NL Query interface
2. Try basic queries:
   - "Show me total sales"
   - "What products do we have?"
   - "How many customers are there?"

**Step 2: Experiment with Complex Queries**
1. Try more sophisticated queries:
   - "Show me sales by region for the last 6 months"
   - "Compare customer satisfaction between products"
   - "Find customers with orders over $1000"

**Step 3: Handle Guardrail Responses**
1. Intentionally try restricted queries:
   - "Show me customer credit card information"
   - "Display all employee salaries"
   - "Give me access to sensitive PII data"

2. Study the guardrail responses:
   - Note the specific restrictions
   - Understand the reasoning provided
   - Learn from suggested alternatives

**Step 4: Refine and Optimize**
1. Use guardrail feedback to improve queries
2. Try suggested alternative approaches
3. Combine multiple queries for comprehensive analysis

**Step 5: Advanced Techniques**
1. Use conversational follow-ups
2. Build complex analysis workflows
3. Export and share results appropriately

#### Success Criteria
- [ ] Successfully executed multiple query types
- [ ] Understood and learned from guardrail responses
- [ ] Improved query effectiveness based on feedback
- [ ] Completed a multi-step analysis workflow

### Lab 3: Self-Service Debugging

#### Objective
Learn to troubleshoot access issues and resolve common problems independently.

#### Prerequisites
- Experience with basic access requests
- Understanding of common access patterns

#### Steps

**Step 1: Common Issue Scenarios**
1. **Expired Access**: Try accessing data after permission expiry
2. **Insufficient Scope**: Attempt analysis requiring broader permissions
3. **Policy Conflicts**: Trigger separation of duties violations
4. **Data Classification**: Access data above your clearance level

**Step 2: Reading Error Messages**
1. Study the specific error details
2. Identify the type of access issue
3. Note any suggested remediation steps
4. Understand the business reasoning

**Step 3: Self-Service Resolution**
1. **For Scope Issues**:
   - Review current permissions
   - Submit scope modification request
   - Use micro-bundle suggestions

2. **For Policy Conflicts**:
   - Understand the conflict reason
   - Request exception or alternative approach
   - Contact appropriate steward

3. **For Expired Access**:
   - Check renewal requirements
   - Submit renewal request
   - Use JIT for immediate needs

**Step 4: Effective Escalation**
1. When to contact support:
   - Unclear error messages
   - System appears to be malfunctioning
   - No self-service resolution available

2. How to provide context:
   - Include request IDs
   - Describe attempted solutions
   - Explain business impact

#### Success Criteria
- [ ] Identified and resolved multiple access issues
- [ ] Understood different types of access problems
- [ ] Used self-service tools effectively
- [ ] Escalated appropriately when needed

---

## Quick Reference Guides

### Access Request Status Guide

| Status | Icon | Meaning | Next Steps |
|--------|------|---------|------------|
| Approved | 🟢 | Access granted | Start using access |
| Modified | 🟡 | Partial approval | Review changes, use approved scope |
| Rejected | 🔴 | Access denied | Review alternatives, appeal if needed |
| Pending | 🕐 | Under review | Monitor status, provide additional info |
| JIT Ready | ⚡ | Temporary access available | Use immediately, expires automatically |

### Common Error Messages

#### "Access Denied: Insufficient Permissions"
**Meaning**: Your current role doesn't allow this access
**Solutions**:
- Request appropriate permissions
- Use suggested micro-bundle
- Contact data owner for alternatives

#### "Policy Violation: Separation of Duties"
**Meaning**: This access would create a conflict with your other permissions
**Solutions**:
- Request exception with business justification
- Use time-limited JIT access
- Work with steward to resolve conflict

#### "Data Classification: Access Restricted"
**Meaning**: Data sensitivity exceeds your clearance level
**Solutions**:
- Request security clearance upgrade
- Use aggregated/certified alternatives
- Work with data custodian for approved access

#### "Access Expired"
**Meaning**: Your permissions have reached their expiry date
**Solutions**:
- Submit renewal request
- Use JIT for immediate needs
- Review usage patterns for future planning

### NL Query Best Practices

#### Query Structure
1. **Be Specific**: Use concrete terms and time periods
2. **Use Business Language**: Frame queries in business terms
3. **Specify Scope**: Include relevant dimensions and filters
4. **Ask One Thing**: Focus each query on a single analysis

#### Effective Patterns
- ✅ "Show me sales performance by region for Q3 2024"
- ✅ "Compare customer acquisition costs between marketing channels"
- ✅ "Find products with declining sales over the last 6 months"
- ❌ "Show me everything about sales" (too broad)
- ❌ "Give me data" (insufficient specificity)

#### Handling Responses
- **Success**: Review results, ask follow-up questions
- **Guardrail**: Read explanation, try suggested alternatives
- **Clarification**: Rephrase query with more context
- **Error**: Check query syntax, verify permissions

### Contact Information

#### For Access Issues
- **Self-Service**: Access portal → Help → Troubleshooting
- **Steward Support**: steward@company.com
- **Emergency**: security@company.com (after hours)

#### For Technical Issues
- **NL Query Help**: data@company.com
- **System Issues**: it-support@company.com
- **Training**: training@company.com

#### Office Hours
- **Steward Drop-in**: Tuesdays 2-4 PM
- **Training Sessions**: Wednesdays 10-11 AM
- **Q&A Sessions**: Fridays 3-4 PM

Remember: Most issues can be resolved through self-service tools. When contacting support, include request IDs, error messages, and steps you've already tried.
