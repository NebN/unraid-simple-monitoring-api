![GitHub Release](https://img.shields.io/github/v/release/nebn/unraid-simple-monitoring-api?display_name=tag&style=for-the-badge)
![GitHub commits since latest release](https://img.shields.io/github/commits-since/nebn/unraid-simple-monitoring-api/latest?style=for-the-badge)
![GitHub last commit](https://img.shields.io/github/last-commit/nebn/unraid-simple-monitoring-api?style=for-the-badge)


> [!NOTE]  
> From version 0.4 disk and memory measurements include decimals. You can decide which units to use.
> If you're using `number` in Homepage's `services.yaml` values will still show as whole numbers, if you wish to display decimals you can change `number` to `float`.
> Percentages will also include more decimal places, all rounding logic has been removed from the API, and the raw floats will be returned. Homepage's `format: percent` will format percentages appropriately.

# Unraid Simple Monitoring API
Simple REST API to monitor basic metrics, currently supports:
- Disk utilization and status
- Network traffic
- CPU load and temperature
- Memory utilization

Originally created for [Unraid](https://unraid.net/) for use with [Homepage](https://gethomepage.dev/widgets/services/customapi/).

## Table of Contents
- [Utilization with Unraid](#unraid)
   - [Installation](#unraid-install)
   - [Configuration](#unraid-conf)
      - [Additional pools](#pools)
      - [Custom units](#units)
      - [CPU Temperature](#cpu-temp)
      - [Logging](#logging-level)
      - [CORS](#cors)  
   - [ZFS](#unraid-zfs)
   - [Calling the API](#unraid-use)
- [Integration with Homepage](#homepage)
    - [Configuration](#homepage-conf)
      - [Available Fields](#available-fields)
- [How reliable are the measurements?](#caveat)
- [Installing a QA build](#qa)

## Utilization with Unraid <a id="unraid"></a> 
### Installation <a id="unraid-install"></a>
#### Unraid Community App
Install from the Unraid community apps.

<img width="380" height="259" alt="image" src="https://github.com/user-attachments/assets/6670690f-25d9-420a-9a81-f516e91cf54b" />

#### Manually
Install manually using docker compose.

```yaml
services:
  unraid-simple-monitoring-api:
    image: ghcr.io/nebn/unraid-simple-monitoring-api:latest
    container_name: unraid-simple-monitoring-api
    privileged: false
    restart: unless-stopped
    ports:
      - '24940:24940'
    volumes:
      - /mnt/user/appdata/unraid-simple-monitoring-api:/app
      - /:/hostfs
    environment:
      - CONF_PATH=/app/conf.yml
      - HOSTFS_PREFIX=/hostfs
```

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
#### Additional pools <a id="pools"></a>
You can add any number of custom disk pools.
```yaml
disks:
  poolname:
    - /mnt/pooldisk1
    - /mnt/pooldisk2
  anotherpool:
    - /mnt/anotherdisk
```

#### Custom units <a id="units"></a>
You can choose which units the measuresments should use.
Accepted values are `B` (bytes),`Ki` (KibiBytes),`K` (KiloBytes) up to [`Qi` and `Q`](https://en.wikipedia.org/wiki/Metric_prefix).
If no unit is specified, a default value will be used.
```yaml
units:
  array: Ti # Default Gi
  cache: Gi # Default Gi
  pools: Ti # Default Gi
  memory: M # Default Mi
```
> [!TIP]  
> Use `float` in your homepage configuration if you wish to see decimals for bigger units, `number` will round down to the nearest integer.


#### CPU Temperature file <a id="cpu-temp"></a>
You can specify which file to read to obtain the correct CPU temperature.
```yaml
cpuTemp: /path/to/temp/file
```
To see where this information might be, you can try running the following command:
```bash
for dir in /sys/class/hwmon/hwmon*; do
  echo "Directory: $dir"
  for file in $dir/temp*_input; do
    echo "Reading from: $file"
    cat $file
  done
done
```
If no file is specified in the configuration, **the software will attempt to figure it out by running a very quick stress test** (a few seconds) while monitoring plausible files. You can find the result of this search in the application's logs. This method is of questionable reliability, specifying which file should be read is the preferred option. 

#### Logging level <a id="logging-level"></a>
```yaml
loggingLevel: DEBUG
```
Accepted values are `DEBUG` `INFO` `WARN` and `ERROR`, it defaults to `INFO`. 

#### CORS <a id="cors"></a>
You can specify these CORS headers:
- Access-Control-Allow-Origin
- Access-Control-Allow-Methods
- Access-Control-Allow-Headers

```yaml
cors:
  origin: "*"
  methods: "method, method"
  headers: "header-name, header-name"
```


### ZFS <a id="unraid-zfs"></a>
If any of the mount points listed in the configuration are using ZFS, the application needs to be run as privileged in order to obtain the correct utilization of ZFS datasets. The command `zfs list` is being used to obtain the correct information, as conventional disk reading methods do not seem to work.

If you are comfortable with running the container as privileged, follow these steps:
- Unraid Docker Tab
- `unraid-simple-monitoring-api` > Edit
- Change `Privileged:` to `ON`
- Apply

You can always decide to turn `Privileged:` back to `OFF`.
> [!TIP]
> If you are not using ZFS, there is no reason to run the container as privileged.

### Calling the API <a id="unraid-use"></a>
Make a request to 
```
http://your-unraid-ip:24940
```
<details>
  <summary>Click to view an example JSON response</summary>

```json
{
  "array": [
    {
      "mount": "/mnt/disk1",
      "total": 3724.20388412476,
      "used": 3608.56298828125,
      "free": 115.640895843506,
      "used_percent": 96.8948827872596,
      "free_percent": 3.10511721274044,
      "temp": 0,
      "disk_id": "WDC_WD40EFPX-1234",
      "is_spinning": false
    },
    {
      "mount": "/mnt/disk2",
      "total": 3724.20388412476,
      "used": 3598.42082977295,
      "free": 125.783054351807,
      "used_percent": 96.6225518724153,
      "free_percent": 3.37744812758466,
      "temp": 0,
      "disk_id": "WDC_WD40EFPX-4321",
      "is_spinning": false
    },
    {
      "mount": "/mnt/disk5",
      "total": 2793.15542221069,
      "used": 2692.77926635742,
      "free": 100.376155853271,
      "used_percent": 96.4063526485101,
      "free_percent": 3.59364735148991,
      "temp": 0,
      "disk_id": "WDC_WD30EFRX-1243",
      "is_spinning": false
    },
    {
      "mount": "/mnt/disk6",
      "total": 1862.10697937012,
      "used": 1749.14831924438,
      "free": 112.958660125732,
      "used_percent": 93.9338254258656,
      "free_percent": 6.06617457413442,
      "temp": 0,
      "disk_id": "WDC_WD2003FZEX-3421",
      "is_spinning": false
    },
    {
      "mount": "/mnt/disk7",
      "total": 931.057510375977,
      "used": 702.875312805176,
      "free": 228.182197570801,
      "used_percent": 75.4921479040906,
      "free_percent": 24.5078520959094,
      "temp": 0,
      "disk_id": "Hitachi_4312",
      "is_spinning": false
    }
  ],
  "cache": [
    {
      "mount": "/mnt/cache",
      "total": 931.512413024902,
      "used": 204.731128692627,
      "free": 726.781284332275,
      "used_percent": 21.978357543063,
      "free_percent": 78.021642456937,
      "temp": 0,
      "disk_id": "Samsung_SSD_870_EVO_1TB_2341",
      "is_spinning": false
    }
  ],
  "pools": [],
  "parity": [
    {
      "name": "parity",
      "temp": 0,
      "disk_id": "WDC_WD80EDBZ-3241",
      "is_spinning": false
    },
    {
      "name": "parity2",
      "temp": 0,
      "disk_id": "",
      "is_spinning": true
    }
  ],
  "network": [
    {
      "interface": "docker0",
      "rx_MiBs": 0,
      "tx_MiBs": 0,
      "rx_Mbps": 0,
      "tx_Mbps": 0
    },
    {
      "interface": "eth0",
      "rx_MiBs": 0.240369570881588,
      "tx_MiBs": 0.00792626055173589,
      "rx_Mbps": 2.01636610525386,
      "tx_Mbps": 0.0664902926743761
    }
  ],
  "array_total": {
    "mount": "/mnt/disk*",
    "total": 13034.7276802063,
    "used": 12351.7867164612,
    "free": 682.940963745117,
    "used_percent": 94.760604283416,
    "free_percent": 5.239395716584,
    "temp": 0,
    "disk_id": "WDC_WD40EFPX-1234 WDC_WD40EFPX-4321 WDC_WD30EFRX-1243 WDC_WD2003FZEX-3421 Hitachi_4312",
    "is_spinning": false
  },
  "cache_total": {
    "mount": "/mnt/cache*",
    "total": 931.512413024902,
    "used": 204.731128692627,
    "free": 726.781284332275,
    "used_percent": 21.978357543063,
    "free_percent": 78.021642456937,
    "temp": 0,
    "disk_id": "Samsung_SSD_870_EVO_1TB_2341",
    "is_spinning": false
  },
  "network_total": {
    "interface": "docker0 eth0",
    "rx_MiBs": 0.240369570881588,
    "tx_MiBs": 0.00792626055173589,
    "rx_Mbps": 2.01636610525386,
    "tx_Mbps": 0.0664902926743761
  },
  "cpu": {
    "load_percent": 9.95962314939435,
    "temp": 33
  },
  "cores": [
    {
      "name": "cpu0",
      "load_percent": 9.15208613728129
    },
    {
      "name": "cpu1",
      "load_percent": 13.1081081081081
    },
    {
      "name": "cpu2",
      "load_percent": 7.93010752688172
    },
    {
      "name": "cpu3",
      "load_percent": 9.79865771812081
    }
  ],
  "memory": {
    "total": 15785.82421875,
    "used": 1387.29296875,
    "free": 14398.53125,
    "used_percent": 8.78822004809992,
    "free_percent": 91.2117799519001
  },
  "error": null
}
```

</details>

## Integration with Homepage <a id="homepage"></a> 
![image](https://github.com/NebN/unraid-simple-monitoring-api/assets/57036949/0175ffbd-fe84-494c-a29f-264f09aae6f3)
### Homepage configuration <a id="homepage-conf"></a>
Check out [Hompage's official custom API widget documentation](https://gethomepage.dev/widgets/services/customapi/).  
Your homepage `services.yaml` should look like this if you want it to look like the above example, showing cache and network data. 

```yaml
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
              suffix: GiB
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

#### Available fields<a id="available-fields"></a>
##### Array Total
```yaml
- field:
    array_total: free # or used, total, used_percent, free_percent, temp, mount, disk_id, is_spinning
  label: your label
  format: number # or percentage
  suffix: GiB # or nothing in case of percentages, or whatver you prefer
```
<br>

##### Cache Total
```yaml
- field:
    cache_total: free # or used, total, used_percent, free_percent, temp, mount, disk_id, is_spinning
  label: your label
  format: number # or percentage
  suffix: GiB # or nothing in case of percentages, or whatver you prefer
```
<br>

##### Specific Disk
```yaml
- field:
    array: # or cache
      0: free 
      # '0' is the index of the disk, 0 = the first 
      # 'free' is the field you wish to read
  label: your label
  format: number
  suffix: GiB
```
<br>

##### Custom pool
```yaml
- field:
    pools:
      0:
        total: free
      # '0' is the index of the pool, 0 = the first 
      # 'free' is the field you wish to read
  label: your label
  format: number
  suffix: GiB
```
<br>

##### Specific disk in custom pool
```yaml
- field:
    pools:
      0: # '0' is the index of the pool, 0 = the first 
        disks: # reading 'disks' list
          0: free # '0' is the index of the disk in the list
  label: your label
  format: number
  suffix: GiB
```
<br>

##### Parity
```yaml
- field:
    parity: 
      0: temp 
      # '0' is the index of the parity disk, 0 = the first 
      # 'temp' is the field you wish to read
  label: your label
  format: number
  suffix: °
```
<br>

##### Network Total
```yaml
- field:
    network_total: rx_MiBs # or tx_MiBs, rx_Mbps, tx_Mbps
  label: your label
  format: float # or 'number' to round to the nearest integer
  suffix: MiB/s # or Mbps, or whatever you prefer
```
<br>

##### Specific Network
```yaml
- field:
    network:
      0: rx_MiBs 
      # '0' is the index of the network, 0 = the first 
      # 'rx_MiBs' is the field you wish to read
  label: your label
  format: float
  suffix: MiB/s 
```
<br>

##### CPU
```yaml
- field:
    cpu: load_percent # or temp
  label: your label
  format: percent # or number
```

<br>

##### Cores
```yaml
- field:
    cores:
      0: load_percent
  label: cpu0
  format: percent
```

<br>

##### Memory
```yaml
- field:
    memory: used_percent # or free_percent, total, used, free
  label: your label
  format: percent
```
<br>

> [!TIP]
> If you wish to show more than the usual 4 allowed fields, there are two solutions:
> - you can set the widget property `display: list` to have the fields displayed in a vertical list that can be arbitrarily long
> ```yaml
> widget:
>   type: customapi
>   url: http://<unraid-ip>:24940
>   display: list
>   mappings:
>      ...
> ```
> ![image](https://github.com/NebN/unraid-simple-monitoring-api/assets/57036949/ed4b694c-ac76-4516-a722-573510e0271c)
>
> <br>
>
> - instead of `widget` you can use `widgets` and specify a list of widgets, each one is able to display up to 4 fields
> ```yaml
> widgets:
>   - type: customapi
>     url: http://<unraid-ip>:24940
>     method: GET
>       mappings:   
>         ...
>
>   - type: customapi
>     url: http://<unraid-ip>:24940
>     method: GET
>       mappings:   
>         ...
> ```
> ![image](https://github.com/user-attachments/assets/f0ae80ab-2884-4bca-90cb-e52c94baa891)
>
> <br>
>   
> You can also combine the two:
> ![image](https://github.com/user-attachments/assets/209db47f-9d96-47f2-9e1d-89fd74f0d93a)






## How reliable are the measurements? <a id="caveat"></a>
The goal of this API is to be simple, fast, and lightweight.  
For these reasons, the measurements provided are not as accurate as they could be.

### Network and CPU
Both Network and CPU usage need to be measured for some time interval. Typically, to get an accurate measurement, you would monitor these for a few seconds before providing a response.  
To avoid having to either:
- wait for the measurement to be completed before responding
- continuosly measure them to have a recent measurement ready to respond with

A different approach has been taken: a snapshot of Network and CPU usage is taken every time the API is called, and the response is the average Network and CPU usage between the current and last API call.
This ensures that the response is quick and reasonably accurate, without having the process continuously read Network and CPU data even when not required.

## Installing a QA build <a id="qa"></a>  
Everyone's Unraid setup is different, therefore, when implementing a new feature or fixing a bug specific to a certain setup, it might be necessary that the end user (you) install a testing deployment to verify that everything works as expected.  

To do so follow these steps:
- Unraid Docker Tab
- `unraid-simple-monitoring-api` > Stop
- Add container
- Template > `unraid-simple-monitoring-api`
- Change the name to something else, e.g.: `unraid-simple-monitoring-api-QA`
- Change `Repository:` to `ghcr.io/nebn/unraid-simple-monitoring-api:qa` (The actual tag might change, currently using `qa`)
- Apply

You should now have 2 installations on your Docker Tab, and can switch between them by stopping/starting them. 

> [!NOTE]  
> Avoid having both active at the same time, as they share the same port and would therefore be unable to start the HTTP service.
 
> [!WARNING]  
> It is a good idea to switch back to the official build as soon as whatever fix you were testing is deployed to it. QA builds are unstable and are likely to not work correctly if you update them further.
