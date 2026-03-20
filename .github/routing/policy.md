# Agent Policy

## Architect agent
- may analyze any service and package
- may create plans and task decomposition
- may not push production code

## Developer agent
- may modify only paths explicitly listed in the approved task packet
- may update tests under the same scope
- may not change contracts without explicit `contracts_impact=true`
- may not auto-merge

## Reviewer agent
- may read repository and CI results
- may comment on PRs
- may not modify code in the same flow

## Mandatory human gates
- any protobuf contract change
- any new NATS subject
- any semantics change to existing subject payload or ownership
- any cross-service flow affecting more than two bounded contexts
- any schema migration with data backfill or replay impact
