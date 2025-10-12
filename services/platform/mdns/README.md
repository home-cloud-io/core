# mdns

The `mdns` service monitors Kubernetes Service objects and hosts an mDNS server to advertise those services on the LAN. It will register mDNS entries for any Service with the `home-cloud.io/dns` annotation set.

<!-- TODO: read these from blueprint -->
It requires that the `HOST_NAME` and `HOST_IP` env vars be set. In Kubernetes that would look like the below:

```yaml
env:
- name: HOST_IP
    valueFrom:
    fieldRef:
        fieldPath: status.hostIP
- name: HOST_NAME
    value: home-cloud
```
