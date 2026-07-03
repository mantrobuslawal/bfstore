# Service-Owned Secrets

**Status:** Proposed  
**Date:** 2026-07-03  
**Repository:** `bfstore`

## Context

bfstore needs clear boundaries between AWS infrastructure identity, Kubernetes workload identity and application-level user/service authorisation.

## Decision

bfstore services will own and consume only their own environment-scoped secrets.

## Rationale

- Keeps infrastructure permissions separate from business permissions.
- Reduces blast radius if one service is compromised.
- Improves auditability and testability.
- Makes service boundaries explicit.

## Consequences

- More service-specific configuration to maintain.
- More test cases required.
- Stronger evidence for production-shaped service design.

## Validation

- Service-specific negative tests.
- Secret access tests.
- Runtime identity checks in dev/stage.

## References

- [Cognito](https://docs.aws.amazon.com/cognito/latest/developerguide/what-is-amazon-cognito.html)
- [Verified Permissions](https://docs.aws.amazon.com/verifiedpermissions/latest/userguide/what-is-avp.html)
- [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
- [IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
