# V1GitRepoVolumeSource


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**directory** | **str** | directory is the target directory name. Must not contain or start with &#39;..&#39;.  If &#39;.&#39; is supplied, the volume directory will be the git repository.  Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name. +optional | [optional] 
**repository** | **str** | repository is the URL | [optional] 
**revision** | **str** | revision is the commit hash for the specified revision. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_git_repo_volume_source import V1GitRepoVolumeSource

# TODO update the JSON string below
json = "{}"
# create an instance of V1GitRepoVolumeSource from a JSON string
v1_git_repo_volume_source_instance = V1GitRepoVolumeSource.from_json(json)
# print the JSON string representation of the object
print(V1GitRepoVolumeSource.to_json())

# convert the object into a dict
v1_git_repo_volume_source_dict = v1_git_repo_volume_source_instance.to_dict()
# create an instance of V1GitRepoVolumeSource from a dict
v1_git_repo_volume_source_from_dict = V1GitRepoVolumeSource.from_dict(v1_git_repo_volume_source_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


