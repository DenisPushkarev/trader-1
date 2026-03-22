# Agent Policy

## Architect agent
- May analyze any service and package
- May create plans and task decomposition
- May NOT push production code

## Developer agent
- May modify only paths explicitly listed in the approved task packet
- May update tests under the same scope
- May NOT change contracts without explicit `contracts_impact=true`
- May NOT auto-merge

## Reviewer agent
- May read repository and CI results
- May comment on PRs
- May NOT modify code in the same flow

## Mandatory human gates
- Any contract/interface breaking change
- Any new messaging subject/topic
- Any semantics change to existing subject payload or ownership
- Any cross-service flow affecting more than two bounded contexts
- Any schema migration with data backfill or replay impact
