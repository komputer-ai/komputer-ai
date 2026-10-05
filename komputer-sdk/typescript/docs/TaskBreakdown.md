
# TaskBreakdown


## Properties

Name | Type
------------ | -------------
`cacheCreateTokens` | number
`cacheReadTokens` | number
`completedAt` | string
`costUSD` | number
`durationMs` | number
`events` | [Array&lt;AgentEvent&gt;](AgentEvent.md)
`index` | number
`inputTokens` | number
`instruction` | string
`outputTokens` | number
`startedAt` | string
`steer` | boolean
`turns` | number

## Example

```typescript
import type { TaskBreakdown } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "cacheCreateTokens": null,
  "cacheReadTokens": null,
  "completedAt": null,
  "costUSD": null,
  "durationMs": null,
  "events": null,
  "index": null,
  "inputTokens": null,
  "instruction": null,
  "outputTokens": null,
  "startedAt": null,
  "steer": null,
  "turns": null,
} satisfies TaskBreakdown

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as TaskBreakdown
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


