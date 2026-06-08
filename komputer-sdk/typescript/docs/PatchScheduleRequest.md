
# PatchScheduleRequest


## Properties

Name | Type
------------ | -------------
`agent` | [CreateScheduleAgentSpec](CreateScheduleAgentSpec.md)
`agentName` | string
`autoDelete` | boolean
`instructions` | string
`keepAgents` | boolean
`schedule` | string
`suspended` | boolean
`timezone` | string

## Example

```typescript
import type { PatchScheduleRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agent": null,
  "agentName": null,
  "autoDelete": null,
  "instructions": null,
  "keepAgents": null,
  "schedule": null,
  "suspended": null,
  "timezone": null,
} satisfies PatchScheduleRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as PatchScheduleRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


