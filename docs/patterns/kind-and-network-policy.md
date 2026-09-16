# Kind and network-policy validation

The default local Kind workflow uses Kind's default CNI for speed and broad compatibility. Set `KIND_CNI=cilium` to create a cluster without the default CNI, install the pinned Cilium chart, enable Hubble and Hubble Relay, and run the same operator installation checks.

The Cilium path is useful when an operator adds or documents NetworkPolicy. Use `make kind-hubble-check` to verify Cilium and Hubble are healthy, then use focused traffic checks for the policy under test. Presence of Hubble alone does not prove that a policy allows or denies the intended traffic.

Always delete the cluster before switching between `default` and `cilium`; the CNI is a cluster-creation choice.
