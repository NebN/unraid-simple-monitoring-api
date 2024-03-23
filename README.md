# Utilization with Unraid
### Installation
Install from the Unraid community apps

### Configuration
By default the application expects a configuration file in 
```
/mnt/user/appdata/unraid-simple-monitoring-api/conf.yml
```

You can find an example file [here](https://github.com/NebN/unraid-simple-monitoring-api/blob/master/conf/conf.yml)

### Utilization
Make a request to 
```
http://your-unraid-ip:24940
```

The response will be formatted this way

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

