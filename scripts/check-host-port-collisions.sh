#!/usr/bin/env bash
set -euo pipefail

# Prints host ports published by running docker containers and detects duplicates.
# Exits with 0 if no collisions, 2 if collisions found.

echo "Scanning published host ports for collisions..."

# Collect host ports published by containers (0.0.0.0:PORT)
ports=$(docker ps --format '{{.Ports}}' | grep -oE '0.0.0.0:[0-9]+' | sed 's/0.0.0.0://' || true)

if [ -z "$ports" ]; then
  echo "No containers publish host ports."
  exit 0
fi

# Print counts per host port
printf "%s\n" $ports | sort | uniq -c | sort -nr | awk '{printf("%5s %s\n", $1, $2)}' > /tmp/_docker_host_ports.txt

echo "Host port -> count"
cat /tmp/_docker_host_ports.txt

# Check for collisions
collisions=$(awk '$1>1{print $0}' /tmp/_docker_host_ports.txt || true)
if [ -n "$collisions" ]; then
  echo "\nCOLLISIONS DETECTED (host ports mapped by multiple containers):"
  echo "$collisions"
  exit 2
fi

echo "\nNo collisions detected."
exit 0
