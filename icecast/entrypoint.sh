#!/bin/sh
sed \
  -e "s|\${ICECAST_SOURCE_PASSWORD}|$ICECAST_SOURCE_PASSWORD|g" \
  -e "s|\${ICECAST_RELAY_PASSWORD}|$ICECAST_RELAY_PASSWORD|g" \
  -e "s|\${ICECAST_ADMIN_USER}|$ICECAST_ADMIN_USER|g" \
  -e "s|\${ICECAST_ADMIN_PASSWORD}|$ICECAST_ADMIN_PASSWORD|g" \
  -e "s|\${ICECAST_HOSTNAME}|$ICECAST_HOSTNAME|g" \
  -e "s|\${ICECAST_SERVER_PORT}|$ICECAST_SERVER_PORT|g" \
  /etc/icecast2/icecast.xml > /tmp/icecast.xml
exec icecast2 -c /tmp/icecast.xml
