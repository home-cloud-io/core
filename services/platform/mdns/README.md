# mdns

The `mdns` service monitors Kubernetes Service objects and hosts an mDNS server to advertise those services on the LAN. It will ignore services that:
- Are outside of the `home-cloud`
- Are not of the type `ExternalName`
- Do not have a `Spec.ExternalName` that matches the `HOST_IP` of the mdns server

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
