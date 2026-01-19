# mdns

The `mdns` service monitors Kubernetes Service objects and hosts an mDNS server to advertise those services on the LAN. It will register mDNS entries for any Service with the `home-cloud.io/dns` annotation set.

It requires that the `HOST_IP` env var be set. In Kubernetes that would look like the below:

```yaml
env:
- name: HOST_IP
    valueFrom:
    fieldRef:
        fieldPath: status.hostIP
```
