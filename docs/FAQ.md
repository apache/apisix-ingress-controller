# FAQ

1. How to bind between Service and Upstream?

All resource objects are uniquely determined by the namespace / name / port combination Id. If the combined Id is the same, the `service` and `upstream` will be considered as a binding relationship.

2. When modifying a CRD, how do other binding objects perceive it?

This is a cascading update problem, see for details [apisix-ingress-controller Design ideas](./design.md)

3. Can I mix CRDs and admin api to define routing rules?

No, currently we are implementing one-way synchronization, that is, CRDs file -> Apache AIPSIX. If the configuration is modified separately through admin api, it will not be synchronized to CRDs in Kubernetes.

This is because CRDs are generally declared in the file system, and Apply to enter Kubernetes etcd, we follow the definition of CRDs and synchronize to Apache Apisix Data Plane, but the reverse will make the situation more complicated.