# Table of Contents
1. [What is this](#what)
2. [Utilization with Unraid](#unraid)
   1. [Installation](#unraid-install)
   2. [Configuration](#unraid-conf)
3. [Integration with Homepage](#homepage)
   1. [Configuration](#homepage-conf)

## What is this? <a id="what"></a> 
Simple rest API to monitor basic metrics: Disk usage and Network traffic.  
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
