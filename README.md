![GitHub Release](https://img.shields.io/github/v/release/nebn/unraid-simple-monitoring-api?display_name=tag&style=for-the-badge)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/nebn/unraid-simple-monitoring-api/latest?style=for-the-badge)
![GitHub last commit](https://img.shields.io/github/last-commit/nebn/unraid-simple-monitoring-api?style=for-the-badge)

# Table of Contents
1. [What is this](#what)
2. [Utilization with Unraid](#unraid)
   1. [Installation](#unraid-install)
   2. [Configuration](#unraid-conf)
3. [Integration with Homepage](#homepage)
   1. [Configuration](#homepage-conf)
4. [How reliable are the measurements?](#caveat)

## What is this? <a id="what"></a> 
Simple rest API to monitor basic metrics, currently supports:
- Disk utilization
- Network traffic
- CPU load
- Memory utilization

Originally created for use with [Homepage](https://gethomepage.dev/latest/widgets/services/customapi/).

## Utilization with Unraid <a id="unraid"></a> 
### Installation <a id="unraid-install"></a>
Install from the Unraid community apps

### Configuration <a id="unraid-conf"></a>
By default the application expects a configuration file in 
```
/mnt/user/appdata/unraid-simple-monitoring-api/conf.yml
```

You can find an example file [here](https://github.com/NebN/unraid-simple-monitoring-api/blob/master/conf/conf.yml). It should look like this

```yaml
networks:
  - eth0
  - anotherNetwork
disks:
  cache:
    - /mnt/cache
    - /another/cache/mount
  array:
    - /mnt/disk1
    - /mnt/disk2
```

### Utilization <a id="unraid-use"></a>
Make a request to 
```
http://your-unraid-ip:24940
```

The response will be formatted this way.

```json
{
   "array":[
      {
         "mount":"/mnt/disk1",
         "total":3724,
         "used":1864,
         "free":1860,
         "used_percent":50.05,
         "free_percent":49.95
      },
      {
         "mount":"/mnt/disk2",
         "total":3724,
         "used":1366,
         "free":2358,
         "used_percent":36.68,
         "free_percent":63.32
      },
      {
         "mount":"/mnt/disk5",
         "total":2793,
         "used":20,
         "free":2773,
         "used_percent":0.72,
         "free_percent":99.28
      },
      {
         "mount":"/mnt/disk6",
         "total":1862,
         "used":85,
         "free":1777,
         "used_percent":4.56,
         "free_percent":95.44
      },
      {
         "mount":"/mnt/disk7",
         "total":931,
         "used":7,
         "free":924,
         "used_percent":0.75,
         "free_percent":99.25
      }
   ],
   "cache":[
      {
         "mount":"/mnt/cache",
         "total":465,
         "used":210,
         "free":255,
         "used_percent":45.16,
         "free_percent":54.84
      }
   ],
   "network":[
      {
         "interface":"docker0",
         "rx_MiBs":0,
         "tx_MiBs":0,
         "rx_Mbps":0,
         "tx_Mbps":0
      },
      {
         "interface":"eth0",
         "rx_MiBs":0.02,
         "tx_MiBs":5.22,
         "rx_Mbps":0.13,
         "tx_Mbps":43.8
      }
   ],
   "array_total":{
      "mount":"total",
      "total":13034,
      "used":3342,
      "free":9692,
      "used_percent":25.64,
      "free_percent":74.36
   },
   "cache_total":{
      "mount":"total",
      "total":465,
      "used":210,
      "free":255,
      "used_percent":45.16,
      "free_percent":54.84
   },
   "network_total":{
      "interface":"total",
      "rx_MiBs":0.02,
      "tx_MiBs":5.22,
      "rx_Mbps":0.13,
      "tx_Mbps":43.8
   },
   "cpu":{
      "load_percent":10.6
   },
   "memory":{
      "total":15788,
      "used":1288,
      "free":14500,
      "used_percent":8.16,
      "free_percent":91.84
   }
}
```

## Integration with Homepage <a id="homepage"></a> 
![image](https://github.com/NebN/unraid-simple-monitoring-api/assets/57036949/0175ffbd-fe84-494c-a29f-264f09aae6f3)
### Configuration <a id="homepage-conf"></a>
Official homepage custom API widget documentation: https://gethomepage.dev/latest/widgets/services/customapi/.  
Your homepage `services.yml` should look like this if you want for example cache and network data. Homepage limits the widget to 4 items.

```yml
- Category:
   - Unraid:
        icon: unraid.png
        href: http://<your-unraid-ip>
        widget:
          type: customapi
          url: http://<your-unraid-ip>:24940
          method: GET # this doesn't matter
          mappings:
            - field:
                cache_total: free
              label: cache free
              format: number
              suffix: "GiB"
            - field:
                cache_total: free_percent
              label: percent
              format: percent
            - field:
                network_total: rx_MiBs
              label: rx
              format: float
              suffix: MiB/s
            - field:
                network_total: tx_MiBs
              label: tx
              format: float
              suffix: MiB/s
```

## How reliable are the measurements? <a id="caveat"></a>
The goal of this API is to be simple, fast, and lightweight.  
For these reasons, the measurements provided are not as accurate as they could be.

### Disk
Disk utilization is rounded down to the nearest GiB.

### Network and CPU
Both Network and CPU usage need to be measured for some time interval. Typically, to get an accurate measurement, you would monitor these for a few seconds before providing a response.  
To avoid having to either:
- wait for the measurement to be completed before responding
- continuosly measure them to have a recent measurement ready to respond with

A different approach has been taken: a snapshot of Network and CPU usage is taken every time the API is called, and the response is the average Network and CPU usage between the current and last API call.
This ensures that the response is quick and reasonably accurate, without having the process continuously read Network and CPU data even when not required.
