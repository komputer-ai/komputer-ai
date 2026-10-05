
# CostBreakdownResponse


## Properties

Name | Type
------------ | -------------
`agent` | string
`cachedAt` | string
`taskCount` | number
`tasks` | [Array&lt;TaskBreakdown&gt;](TaskBreakdown.md)
`totalCost` | number

## Example

```typescript
import type { CostBreakdownResponse } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "agent": null,
  "cachedAt": null,
  "taskCount": null,
  "tasks": null,
  "totalCost": null,
} satisfies CostBreakdownResponse

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as CostBreakdownResponse
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


