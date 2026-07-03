# Catalog Service IAM

**Repository:** `bfstore`  
**Status:** Draft v0.1  
**Last updated:** 2026-07-03

## Purpose

Define AWS IAM and runtime identity requirements for the `catalog` service.

## Scope

Applies to `catalog` deployment, Kubernetes service account, AWS workload role and secret access.

## bfstore position

bfstore uses a production-shaped AWS platform model: multi-account separation, federated workforce access, short-lived automation credentials, explicit workload identity, policy-as-code guardrails, central audit, and restore-tested backup strategies. The project deliberately self-manages selected platform components where that demonstrates operational competence, while documenting managed-service alternatives and trade-offs.

## Implementation guidance

- Read catalog database secret.
- Access product media/object storage later if adopted.
- Publish catalog events only if required.

## Required controls

- Service has its own Kubernetes service account.
- Service has its own IAM role if AWS API access is needed.
- Service can read only its own secrets.
- Access is environment-scoped.

## Validation and evidence

- Deploy in dev and test required AWS API calls.
- Verify access denied for unrelated secrets/resources.



## References

- [EKS Pod Identity](https://docs.aws.amazon.com/eks/latest/userguide/pod-identities.html)
- [IAM roles](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles.html)
- [KMS key policies](https://docs.aws.amazon.com/kms/latest/developerguide/key-policies.html)
