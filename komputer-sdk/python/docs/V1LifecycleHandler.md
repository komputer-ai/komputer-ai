# V1LifecycleHandler


## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**var_exec** | [**V1ExecAction**](V1ExecAction.md) | Exec specifies a command to execute in the container. +optional | [optional] 
**http_get** | [**V1HTTPGetAction**](V1HTTPGetAction.md) | HTTPGet specifies an HTTP GET request to perform. +optional | [optional] 
**sleep** | [**V1SleepAction**](V1SleepAction.md) | Sleep represents a duration that the container should sleep. +optional | [optional] 
**tcp_socket** | [**V1TCPSocketAction**](V1TCPSocketAction.md) | Deprecated. TCPSocket is NOT supported as a LifecycleHandler and kept for backward compatibility. There is no validation of this field and lifecycle hooks will fail at runtime when it is specified. +optional | [optional] 

## Example

```python
from komputer_ai.models.v1_lifecycle_handler import V1LifecycleHandler

# TODO update the JSON string below
json = "{}"
# create an instance of V1LifecycleHandler from a JSON string
v1_lifecycle_handler_instance = V1LifecycleHandler.from_json(json)
# print the JSON string representation of the object
print(V1LifecycleHandler.to_json())

# convert the object into a dict
v1_lifecycle_handler_dict = v1_lifecycle_handler_instance.to_dict()
# create an instance of V1LifecycleHandler from a dict
v1_lifecycle_handler_from_dict = V1LifecycleHandler.from_dict(v1_lifecycle_handler_dict)
```
[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


