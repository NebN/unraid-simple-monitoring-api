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
         "total":906,
         "used":515,
         "free":391,
         "free_percent":43.16,
         "used_percent":56.84
      }
   ],
   "cache":[
      {
         "mount":"/",
         "total":906,
         "used":515,
         "free":391,
         "free_percent":43.16,
         "used_percent":56.84
      }
   ],
   "network":[
      {
         "interface":"enp42s0",
         "rx_MiBs":0.81,
         "tx_MiBs":0.03,
         "rx_Mbps":6.84,
         "tx_Mbps":0.22
      },
      {
         "interface":"eth0",
         "rx_MiBs":0,
         "tx_MiBs":0,
         "rx_Mbps":0,
         "tx_Mbps":0
      }
   ],
   "array_total":{
      "mount":"total",
      "total":906,
      "used":515,
      "free":391,
      "free_percent":43.16,
      "used_percent":56.84
   },
   "cache_total":{
      "mount":"total",
      "total":906,
      "used":515,
      "free":391,
      "free_percent":43.16,
      "used_percent":56.84
   },
   "network_total":{
      "interface":"total",
      "rx_MiBs":0.81,
      "tx_MiBs":0.03,
      "rx_Mbps":6.84,
      "tx_Mbps":0.22
   },
   "cpu": {
      "load_percent":3.79
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
