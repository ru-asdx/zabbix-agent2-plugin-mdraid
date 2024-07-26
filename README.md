# MDRaid Plugin for Zabbix Agent 2

## Description
mdadm monitoring plugin for zabbix-agent2.
plugin read the data from sysfs, instead of parsing "/proc/mdstat" and "mdadm -D". 

## Building

``` go
go build ./zabbix-agent2-plugin-mdraid.go
```

## Installation

Copy the binary to `/usr/sbin/zabbix-agent2-plugin/`. 
Copy the configuration file to `/etc/zabbix/zabbix_agent2.d/plugins.d/` and restart the agent.

## Supported items

* `mdraid.dev.discovery` - List of known MD devices in LLD format
    * `{#DEVICE}`
* `mdraid.dev.get[<device>]` - The state of a device as a json object
    * `Device` - Device name
    * `Level` - The raid level (e.g. raid0, raid1, raid5, linear, multipath, faulty).
    * `ArrayState`- Describes the current state of the array. 
    * `RaidDisks` - The number of devices in a fully functional array.
    * `DegradedDisks` - Contains a count of the number of devices by which the arrays is degraded.
    * `SyncAction`
    * `SyncCompleted`

