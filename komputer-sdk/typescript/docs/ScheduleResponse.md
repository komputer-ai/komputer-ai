
# ScheduleResponse


## Properties

Name | Type
------------ | -------------
`agent` | [CreateScheduleAgentSpec](CreateScheduleAgentSpec.md)
`agentName` | string
`autoDelete` | boolean
`createdAt` | string
`failedRuns` | number
`instructions` | string
`keepAgents` | boolean
`lastRunCostUSD` | string
`lastRunStatus` | string
`lastRunTime` | string
`lastRunTokens` | number
`name` | string
`namespace` | string
`nextRunTime` | string
`phase` | string
`runCount` | number
`schedule` | string
`successfulRuns` | number
`suspended` | boolean
`timezone` | string
`totalCostUSD` | string
`totalTokens` | number

## Example

```typescript
import type { ScheduleResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agent": null,
  "agentName": null,
  "autoDelete": null,
  "createdAt": null,
  "failedRuns": null,
  "instructions": null,
  "keepAgents": null,
  "lastRunCostUSD": null,
  "lastRunStatus": null,
  "lastRunTime": null,
  "lastRunTokens": null,
  "name": null,
  "namespace": null,
  "nextRunTime": null,
  "phase": null,
  "runCount": null,
  "schedule": null,
  "successfulRuns": null,
  "suspended": null,
  "timezone": null,
  "totalCostUSD": null,
  "totalTokens": null,
} satisfies ScheduleResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as ScheduleResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


