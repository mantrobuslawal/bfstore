# Secrets Consumption

**Repository:** `bfstore`  
**Status:** Draft v0.1  
**Last updated:** 2026-07-03

## Purpose

Define how application services consume secrets.

## Scope

Application code, service runtime identity, secrets, customer identity and app-level authorisation.

## bfstore position

bfstore uses a production-shaped AWS platform model: multi-account separation, federated workforce access, short-lived automation credentials, explicit workload identity, policy-as-code guardrails, central audit, and restore-tested backup strategies. The project deliberately self-manages selected platform components where that demonstrates operational competence, while documenting managed-service alternatives and trade-offs.

## Implementation guidance

- Secrets are scoped by service and environment.
- Services read only the secrets they need.
- Secrets must not be logged or committed.

## Required controls

- No AWS static keys in app config.
- No shared app service role.
- No sensitive data in logs.

## Validation and evidence

- Unit/integration tests for authorisation paths.
- Secret access tests.
- Negative tests for unauthorised access.



## References

- [Cognito](https://docs.aws.amazon.com/cognito/latest/developerguide/what-is-amazon-cognito.html)
- [Verified Permissions](https://docs.aws.amazon.com/verifiedpermissions/latest/userguide/what-is-avp.html)
- [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
- [IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
