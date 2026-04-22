# V1FlockerVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**dataset_name** | **str** | datasetName is Name of the dataset stored as metadata -&gt; name on the dataset for Flocker should be considered as deprecated +optional | [optional] 
**dataset_uuid** | **str** | datasetUUID is the UUID of the dataset. This is unique identifier of a Flocker dataset +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_flocker_volume_source import V1FlockerVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1FlockerVolumeSource from a JSON string
v1_flocker_volume_source_instance = V1FlockerVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1FlockerVolumeSource.to_json())

# convert the object into a dict
v1_flocker_volume_source_dict = v1_flocker_volume_source_instance.to_dict()
# create an instance of V1FlockerVolumeSource from a dict
v1_flocker_volume_source_from_dict = V1FlockerVolumeSource.from_dict(v1_flocker_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


