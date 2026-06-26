# ATR Deployment Checklist

## Pre-Deployment (Dev Environment)

- [ ] XAI API key validated (test call successful)
- [ ] PostgreSQL 14+ running locally
- [ ] Temporal server running (`temporal server start-dev`)
- [ ] Go 1.24+ installed
- [ ] Node.js 18+ installed
- [ ] Database migrations applied
- [ ] Backend server starts without errors
- [ ] Frontend builds successfully
- [ ] API health check passes: `curl http://localhost:8080/health`

## Local Testing

- [ ] Create test trade data
- [ ] Create test confirmations
- [ ] Run manual reconciliation (API call)
- [ ] Verify results in database
- [ ] Check discrepancies are correctly identified
- [ ] Test low-code rules application
- [ ] Test task creation and assignment
- [ ] Test dashboard loads and displays results

## Staging Deployment

- [ ] Docker images built
- [ ] Environment variables configured
- [ ] Database backups created
- [ ] docker-compose.yml validated
- [ ] All services start correctly
- [ ] Load test reconciliation (1000+ trades/confirms)
- [ ] Verify performance metrics
- [ ] Test error handling and recovery
- [ ] Security scan completed

## Security Review

- [ ] ABAC policies configured correctly
- [ ] Audit logging enabled
- [ ] API authentication/authorization in place
- [ ] Database encryption at rest
- [ ] HTTPS configured (if production)
- [ ] Rate limiting enabled
- [ ] SQL injection prevention verified
- [ ] XOR AI API key securely stored

## Production Deployment

- [ ] Kubernetes manifests created (if applicable)
- [ ] Persistent volume claims configured
- [ ] Health checks and readiness probes set
- [ ] Resource limits defined
- [ ] Monitoring and alerting set up
- [ ] Log aggregation configured
- [ ] Backup and disaster recovery tested
- [ ] On-call runbook created
- [ ] Stakeholders notified

## Post-Deployment

- [ ] First reconciliation run successful
- [ ] Discrepancy notifications working
- [ ] Tasks created and assigned
- [ ] PDF reports generating
- [ ] Dashboard accessible and performant
- [ ] No errors in application logs
- [ ] Database backups running
- [ ] Monitoring alerts firing as expected

## Success Criteria

- ✅ Match rate >98%
- ✅ Processing time <5 minutes
- ✅ API latency <500ms
- ✅ Zero critical bugs
- ✅ All tests passing
- ✅ Documentation complete
- ✅ Team trained on operations

---

**Deployment Date:** ____________  
**Deployed By:** ____________  
**Sign-Off:** ____________
