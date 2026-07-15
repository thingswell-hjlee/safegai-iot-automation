# AWS Edge Security Review

## Scope

Security assessment of the SafeGAI AWS-First Edge-Ready implementation
covering cloud infrastructure, gateway software, and communication channels.

## Security Controls Implemented

### Infrastructure
| Control | Implementation | Rating |
|---------|---------------|--------|
| IAM least privilege | Per-service roles, scoped resources | Good |
| No long-lived credentials | OIDC federation for CI/CD | Good |
| Network segmentation | VPC with security groups | Good |
| Encryption at rest | S3 SSE, EBS encryption | Good |
| No SSH access | SSM-only management | Good |
| IMDSv2 required | Instance metadata hardened | Good |
| Auto-stop | Prevents runaway costs and exposure | Good |

### Application
| Control | Implementation | Rating |
|---------|---------------|--------|
| Session management | Secure cookies with expiry | Adequate |
| RBAC | Role-based access control | Adequate |
| Input validation | JSON schema validation | Adequate |
| Audit logging | All safety decisions logged | Good |
| Replay guard | Output commands not replayed | Good |
| Graceful shutdown | State preserved on signal | Good |

### Communication
| Control | Implementation | Rating |
|---------|---------------|--------|
| IoT Core TLS | Mutual TLS with X.509 certs | Good |
| API Gateway auth | Cognito + JWT tokens | Good |
| CloudFront HTTPS | Redirect HTTP to HTTPS | Good |
| Internal comms | VPC-scoped, not encrypted | Needs Improvement |

## Identified Risks

### HIGH
- No TLS on local gateway API (HTTP only on LAN)
- Session secret from environment variable (not secrets manager)

### MEDIUM
- Internal simulator traffic is unencrypted (acceptable in sim)
- No rate limiting on gateway API endpoints
- Cognito user pool allows admin-created users only (intended)

### LOW
- Safety rules are read-only but not cryptographically signed
- Audit log integrity relies on file system permissions
- No WAF on API Gateway (cost consideration for sim)

## Recommendations

1. **Before pilot**: Add TLS to gateway API (self-signed or Let's Encrypt)
2. **Before pilot**: Move secrets to AWS Secrets Manager or local vault
3. **Future**: Add API rate limiting to prevent DoS
4. **Future**: Implement audit log signing for tamper detection
5. **Future**: Add WAF rules for API Gateway in production

## Compliance Notes

- No PII stored in events (only sensor data and safety decisions)
- Audit trail meets traceability requirements
- Data retention limited (7 days in sim, configurable in production)
- No camera image storage (only metadata/events)

## Credential Management

| Credential | Storage | Rotation |
|------------|---------|----------|
| OIDC token | GitHub (short-lived) | Per-workflow (1h) |
| IoT cert | Instance profile | Manual |
| Session secret | Env variable | Manual |
| Cognito users | User pool | Password policy enforced |

## Conclusion

The security posture is appropriate for a simulation/development environment.
Before pilot deployment, TLS and secrets management should be addressed.
The architecture supports these additions without code changes (config only).
