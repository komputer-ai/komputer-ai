# V1DownwardAPIProjection


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**items** | [**List[V1DownwardAPIVolumeFile]**](V1DownwardAPIVolumeFile.md) | Items is a list of DownwardAPIVolume file +optional +listType&#x3D;atomic | [optional] 

## Example

```python
from komputer_ai.models.v1_downward_api_projection import V1DownwardAPIProjection

# TODO update the JSON string below
json = "{}"
# create an instance of V1DownwardAPIProjection from a JSON string
v1_downward_api_projection_instance = V1DownwardAPIProjection.from_json(json)
# print the JSON string representation of the object
print(V1DownwardAPIProjection.to_json())

# convert the object into a dict
v1_downward_api_projection_dict = v1_downward_api_projection_instance.to_dict()
# create an instance of V1DownwardAPIProjection from a dict
v1_downward_api_projection_from_dict = V1DownwardAPIProjection.from_dict(v1_downward_api_projection_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


