#!/bin/sh

# zfs list will not work if the container is being run as unprivileged 
if zfs list > /dev/null 2>&1; then
  export ZFS_OK="true"
else
  export ZFS_OK="false"
fi

echo "ZFS_OK=${ZFS_OK}" >> /etc/environment

# Source the /etc/environment 
. /etc/environment

exec "$@"
