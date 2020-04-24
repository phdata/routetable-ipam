# Warning
Though functional, this software is still in an alpha state. Though written with compatibility in mind, this software has not been tested with IPv6.

# Description
This repository contains a CNI compatible IPAM plugin for use with container cluster systems. When an address is requested within a given network, the plugin will generate a random address within that network, and then consult the local routing table to determine whether that address already exists. If the host prefix represeting the generated address is not found, the address will be converted to a host prefix and then stored into the routing table using a custom protocol number. A host prefix is a "subnet" where the number of bits in the subnet mask is equal to the number of bits in the address. In other words, a /32 address in IPv4, or a /128 address in IPv6.

The real power of this plugin comes into play when you couple it with a routing protocol. Doing that allows you to get a cluster-wide view of available IP addresses on a given network. Care is taken to avoid race conditions where the same address is selected on multiple nodes simultaneously. There is a configurable amount of time to wait an expected propogation timeout before the routing table is consulted a second time to ensure that the address wasn't first selected somewhere else.

Some initial features include:
 * The ability to request a specific address in the network
 * The ability to exclude some number of addresses from the beginning or from the end of the range

Along with a routing protocol, this plugin is intended to be coupled with a CNI plugin that provides the ability to span layer 2 networks across multiple nodes, such as our [vxlan-cni](https://github.com/phdata/vxlan-cni) plugin.



The ipam concepts and some of this code were inspired by and are originally from [here](https://github.com/TrilliumIT/vxrouter)
