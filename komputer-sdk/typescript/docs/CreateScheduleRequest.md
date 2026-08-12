
# CreateScheduleRequest


## Properties

Name | Type
------------ | -------------
`agent` | [V1alpha1ScheduleAgentSpec](V1alpha1ScheduleAgentSpec.md)
`agentName` | string
`autoDelete` | boolean
`instructions` | string
`keepAgents` | boolean
`name` | string
`namespace` | string
`schedule` | string
`timezone` | string

## Example

```typescript
import type { CreateScheduleRequest } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agent": null,
  "agentName": null,
  "autoDelete": null,
  "instructions": null,
  "keepAgents": null,
  "name": null,
  "namespace": null,
  "schedule": null,
  "timezone": null,
} satisfies CreateScheduleRequest

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as CreateScheduleRequest
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


