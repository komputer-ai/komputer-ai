
# SkillResponse


## Properties

Name | Type
------------ | -------------
`agentNames` | Array&lt;string&gt;
`attachedAgents` | number
`content` | string
`createdAt` | string
`description` | string
`isDefault` | boolean
`name` | string
`namespace` | string
`updatedAt` | string

## Example

```typescript
import type { SkillResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agentNames": null,
  "attachedAgents": null,
  "content": null,
  "createdAt": null,
  "description": null,
  "isDefault": null,
  "name": null,
  "namespace": null,
  "updatedAt": null,
} satisfies SkillResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as SkillResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


