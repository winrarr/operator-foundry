# API Reference

## Packages
- [patterns.operator-foundry.example/v1alpha1](#patternsoperator-foundryexamplev1alpha1)


## patterns.operator-foundry.example/v1alpha1

Package v1alpha1 contains the reference custom resources used by Operator Foundry.

Package v1alpha1 contains the reference custom resources used by Operator Foundry.

### Resource Types
- [PatternConnection](#patternconnection)
- [PatternMembership](#patternmembership)
- [PatternResource](#patternresource)



#### CreationPolicy

_Underlying type:_ _string_

CreationPolicy controls how an external resource is acquired.

_Validation:_
- Enum: [Create Adopt CreateOrAdopt]

_Appears in:_
- [PatternResourceSpec](#patternresourcespec)

| Field | Description |
| --- | --- |
| `Create` |  |
| `Adopt` |  |
| `CreateOrAdopt` |  |


#### DeletionPolicy

_Underlying type:_ _string_

DeletionPolicy controls whether the external resource is deleted with the Kubernetes resource.

_Validation:_
- Enum: [Delete Orphan]

_Appears in:_
- [PatternResourceSpec](#patternresourcespec)

| Field | Description |
| --- | --- |
| `Delete` |  |
| `Orphan` |  |


#### LocalObjectReference



LocalObjectReference identifies a resource in the same namespace.



_Appears in:_
- [PatternMembershipSpec](#patternmembershipspec)
- [PatternResourceSpec](#patternresourcespec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the referenced resource name. |  | MinLength: 1 <br /> |


#### PatternConnection









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `patterns.operator-foundry.example/v1alpha1` | | |
| `kind` _string_ | `PatternConnection` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PatternConnectionSpec](#patternconnectionspec)_ |  |  |  |


#### PatternConnectionSpec



PatternConnectionSpec configures access to the example external API.



_Appears in:_
- [PatternConnection](#patternconnection)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `endpoint` _string_ | Endpoint is the external API base URL. |  | Pattern: `^https?://` <br /> |
| `authSecretRef` _[SecretKeyReference](#secretkeyreference)_ | AuthSecretRef references a same-namespace Secret containing the bearer token. |  |  |
| `requestTimeout` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | RequestTimeout bounds each external API request. |  | Optional: \{\} <br /> |


#### PatternMembership



PatternMembership claims one relationship edge between two PatternResources.





| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `patterns.operator-foundry.example/v1alpha1` | | |
| `kind` _string_ | `PatternMembership` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PatternMembershipSpec](#patternmembershipspec)_ |  |  |  |


#### PatternMembershipSpec



PatternMembershipSpec claims one relationship edge between two managed
PatternResources. Delete the PatternMembership to delete only this edge.



_Appears in:_
- [PatternMembership](#patternmembership)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[LocalObjectReference](#localobjectreference)_ | ConnectionRef selects the same-namespace PatternConnection used for the edge. |  |  |
| `parentRef` _[LocalObjectReference](#localobjectreference)_ | ParentRef identifies the resource that owns the relationship. |  |  |
| `memberRef` _[LocalObjectReference](#localobjectreference)_ | MemberRef identifies the resource connected to the parent. |  |  |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | DriftDetectionInterval controls periodic re-observation of the external edge.<br />When omitted, the controller reconciles on Kubernetes events only. |  | Optional: \{\} <br /> |


#### PatternResource









| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `apiVersion` _string_ | `patterns.operator-foundry.example/v1alpha1` | | |
| `kind` _string_ | `PatternResource` | | |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |  |  |
| `spec` _[PatternResourceSpec](#patternresourcespec)_ |  |  |  |


#### PatternResourceSpec



PatternResourceSpec demonstrates the common lifecycle fields of an external resource.



_Appears in:_
- [PatternResource](#patternresource)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `connectionRef` _[LocalObjectReference](#localobjectreference)_ | ConnectionRef identifies the connection used for external operations. |  |  |
| `externalName` _string_ | ExternalName is the external identity. When omitted, metadata.name is used. |  | Optional: \{\} <br /> |
| `value` _string_ | Value is the desired external value. |  |  |
| `creationPolicy` _[CreationPolicy](#creationpolicy)_ | CreationPolicy controls create versus adoption behavior.<br />Defaults to Create. |  | Enum: [Create Adopt CreateOrAdopt] <br />Optional: \{\} <br /> |
| `deletionPolicy` _[DeletionPolicy](#deletionpolicy)_ | DeletionPolicy controls whether deleting this object deletes its external resource.<br />Defaults to Orphan. |  | Enum: [Delete Orphan] <br />Optional: \{\} <br /> |
| `driftDetectionInterval` _[Duration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.37/#duration-v1-meta)_ | DriftDetectionInterval controls periodic external observations. |  | Optional: \{\} <br /> |


#### SecretKeyReference



SecretKeyReference identifies a key in a Secret in the same namespace.



_Appears in:_
- [PatternConnectionSpec](#patternconnectionspec)

| Field | Description | Default | Validation |
| --- | --- | --- | --- |
| `name` _string_ | Name is the Secret name. |  | MinLength: 1 <br /> |
| `key` _string_ | Key is the data key. It defaults to token when omitted. |  | Optional: \{\} <br /> |
