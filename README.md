# SevenBridges
Automate the creation of Docker network topologies.

## Sample YAML file
The following YAML file would be used for creating a network with the topology {{0, 1, 2}, {2, 3}, {1, 3}}.
We also bridge container 1 to the host network.
Each number in the topology represents a container, and each set is a network.
Notice there's a hole in this network (around {1, 2, 3}).
```yaml
name: "example"
image: "alpine"
networks:
  - containers: [0, 1, 2]
    internal: true
    label: "big"
  - containers: [2, 3]
    internal: true
    image: "ubuntu"
  - containers: [1, 3]
containers:
  - bridge: true
    cmd: ["sh"]
    image: "ubuntu"
    containers:
      - 1
```

Pass in the `-teardown` flag to teardown the topology described in the passed in YAML file.
