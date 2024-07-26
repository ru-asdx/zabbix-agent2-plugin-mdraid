package plugin

import (
    "fmt"
    "strconv"
    "strings"
    "os"
    "syscall"
    "bytes"

    "encoding/json"
    "path/filepath"

    "golang.zabbix.com/sdk/plugin"

)

const Name = "MDRaid"

const SYSFS_PATH = "/sys/block/"

// Plugin must define structure and embed plugin.Base structure.
type Plugin struct {
    plugin.Base
}

type Mdraid struct {
    Device     string `json:"{#DEVICE}"`
}

type MdraidState struct {
    Device	      string  `json:"Device"`
    Level	      string  `json:"Level"`
    ArrayState    string  `json:"ArrayState"`
    RaidDisks     uint64  `json:"RaidDisks"`
    DegradedDisks uint64  `json:"DegradedDisks"`
    SyncAction	  string  `json:"SyncAction"`
    SyncCompleted float64 `json:"SyncCompleted"`
}

// Create a new instance of the defined plugin structure
var Impl Plugin

func SysReadFile(file string) (string, error) {
    f, err := os.Open(file)
    if err != nil {
        return "", err
    }
    defer f.Close()

    // On some machines, hwmon drivers are broken and return EAGAIN.  This causes
    // Go's os.ReadFile implementation to poll forever.
    //
    // Since we either want to read data or bail immediately, do the simplest
    // possible read using syscall directly.
    const sysFileBufferSize = 128
    b := make([]byte, sysFileBufferSize)
    n, err := syscall.Read(int(f.Fd()), b)
    if err != nil {
        return "", err
    }

    return string(bytes.TrimSpace(b[:n])), nil
}

func ReadUintFromFile(file string) (uint64, error) {
    data, err := SysReadFile(file)
    if err != nil {
        return 0, err
    }
    return strconv.ParseUint(strings.TrimSpace(data), 10, 64)
}


func (p *Plugin) MdraidDiscovery() (jsonArray []byte, err error) {
    devices, err := filepath.Glob( filepath.Join(SYSFS_PATH, "md*") )
    if err != nil {
        return nil, fmt.Errorf("Discovery failed: cannot fetch data. (%s)", err)
    }

    out := []Mdraid{}

    for _, dev := range devices {
        out = append(out, Mdraid{ Device: strings.TrimPrefix(dev, SYSFS_PATH) })
    }

    jsonArray, err = json.Marshal(out)
    if err != nil {
        return nil, fmt.Errorf("Discovery failed: cannot marshal JSON. (%s)", err)
    }

    return
}

func (p *Plugin) MdraidGet(dev string) (jsonArray []byte, err error) {

    md := MdraidState{Device: dev}
    path := filepath.Join(SYSFS_PATH, md.Device, "md")

    if val, err := SysReadFile(filepath.Join(path, "level")); err == nil {
        md.Level = val
    } else {
        return nil, err
    }

    // Array state can be one of: clear, inactive, readonly, read-auto, clean, active,
    // write-pending, active-idle.
    if val, err := SysReadFile(filepath.Join(path, "array_state")); err == nil {
        md.ArrayState = val
    } else {
        return nil, err
    }

    if val, err := ReadUintFromFile(filepath.Join(path, "raid_disks")); err == nil {
        md.RaidDisks = val
    } else {
        return nil, err
    }


    switch md.Level {
    case "raid1", "raid4", "raid5", "raid6", "raid10":
        if val, err := ReadUintFromFile(filepath.Join(path, "degraded")); err == nil {
            md.DegradedDisks = val
        } else {
            return nil, err
        }

        // Array sync action can be one of: resync, recover, idle, check, repair.
        if val, err := SysReadFile(filepath.Join(path, "sync_action")); err == nil {
            md.SyncAction = val
        } else {
            return nil, err
        }

        if val, err := SysReadFile(filepath.Join(path, "sync_completed")); err == nil {
            if val != "none" {
                var a, b uint64

                // File contains two values representing the fraction of number of completed
                // sectors divided by number of total sectors to process.
                if _, err := fmt.Sscanf(val, "%d / %d", &a, &b); err == nil {
                    md.SyncCompleted = float64(a) / float64(b)
                } else {
                    return nil, err
                }
            }
        } else {
            return nil, err
        }
    }

    jsonArray, err = json.Marshal(md)
    if err != nil {
        return nil, fmt.Errorf("Getdata failed: cannot marshal JSON. (%s)", err)
    }

    return
}

// Plugin must implement one or several plugin interfaces.
func (p *Plugin) Export(key string, params []string, ctx plugin.ContextProvider) (result interface{}, err error) {

    var jsonArray []byte

    switch key {
    case "mdraid.dev.discovery":
        jsonArray, err = p.MdraidDiscovery()
        if err != nil {
            return nil, err
        }

    case "mdraid.dev.get":
        if len(params) == 0 {
            return nil, fmt.Errorf("Getdata failed: too few parameters.")
        }

        jsonArray, err = p.MdraidGet(params[0])
        if err != nil {
            return nil, err
        }

    default:
        return nil, plugin.UnsupportedMetricError

    }

    return string(jsonArray), nil
}

func init() {
    // Register the metric, specifying the plugin and metric details.
    // 1 - a pointer to plugin implementation
    // 2 - plugin name
    // 3 - metric name (item key)
    // 4 - metric description
    //
    // NB! The metric description must end with a period, otherwise Zabbix agent 2 will return an error and won't start!
    // Metric name (item key) and metric description can be repeated in a loop to register additional metrics.
    plugin.RegisterMetrics(
        &Impl, Name,
        "mdraid.dev.discovery", "Return JSON array of MDraid devices.",
        "mdraid.dev.get", "Return JSON data of MDraid device.",
    )
}
