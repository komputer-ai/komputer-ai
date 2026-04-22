
# V1PodDNSConfig


## Properties

Name | Type
------------ | -------------
`nameservers` | Array&lt;string&gt;
`options` | [Array&lt;V1PodDNSConfigOption&gt;](V1PodDNSConfigOption.md)
`searches` | Array&lt;string&gt;

## Example

```typescript
import type { V1PodDNSConfig } from '@komputer-ai/sdk'

// TODO: Update the object below with actual values
const example = {
  "nameservers": null,
  "options": null,
  "searches": null,
} satisfies V1PodDNSConfig

console.log(example)

// Convert the instance to a JSON string
const exampleJSON: string = JSON.stringify(example)
console.log(exampleJSON)

// Parse the JSON string back to an object
const exampleParsed = JSON.parse(exampleJSON) as V1PodDNSConfig
console.log(exampleParsed)
```

[[Back to top]](#) [[Back to API list]](../README.md#api-endpoints) [[Back to Model list]](../README.md#models) [[Back to README]](../README.md)


