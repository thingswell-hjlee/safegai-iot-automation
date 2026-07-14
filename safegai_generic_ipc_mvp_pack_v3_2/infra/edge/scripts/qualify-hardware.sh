#!/bin/sh
set -eu
out="${1:-qualification.json}"
arch="$(dpkg --print-architecture 2>/dev/null || uname -m)"
cores="$(getconf _NPROCESSORS_ONLN)"
mem_kib="$(awk '/MemTotal/ {print $2}' /proc/meminfo)"
mem_mib="$((mem_kib / 1024))"
kernel="$(uname -r)"
os_id="$(. /etc/os-release && printf '%s-%s' "$ID" "$VERSION_ID")"
root_source="$(findmnt -n -o SOURCE /)"
root_size_bytes="$(lsblk -bndo SIZE "$root_source" 2>/dev/null | head -n 1 || printf '0')"
[ -n "$root_size_bytes" ] || root_size_bytes=0
root_size_gib="$((root_size_bytes / 1024 / 1024 / 1024))"
uefi=false
[ -d /sys/firmware/efi ] && uefi=true
wired_nics=""
wired_count=0
for path in /sys/class/net/*; do
  iface="$(basename "$path")"
  [ "$iface" = "lo" ] && continue
  [ -e "$path/device" ] || continue
  [ -d "$path/wireless" ] && continue
  type="$(cat "$path/type" 2>/dev/null || printf '0')"
  [ "$type" = "1" ] || continue
  driver="$(basename "$(readlink -f "$path/device/driver" 2>/dev/null || printf 'unknown')")"
  [ -n "$driver" ] || driver=unknown
  speed="$(cat "$path/speed" 2>/dev/null || printf 'unknown')"
  wired_count=$((wired_count + 1))
  entry="{\"name\":\"$iface\",\"driver\":\"$driver\",\"speedMbps\":\"$speed\"}"
  if [ -n "$wired_nics" ]; then wired_nics="$wired_nics,$entry"; else wired_nics="$entry"; fi
done
cat > "$out" <<JSON
{
  "hardwareProfileId": "ipc-lite-amd64-v1",
  "hardwareGrade": "SG-EPC-L1",
  "architecture": "$arch",
  "cpuCores": $cores,
  "memoryMiB": $mem_mib,
  "rootStorageGiB": $root_size_gib,
  "wiredEthernetPorts": $wired_count,
  "wiredInterfaces": [$wired_nics],
  "uefi": $uefi,
  "kernel": "$kernel",
  "os": "$os_id",
  "rootSource": "$root_source"
}
JSON
printf 'Wrote %s\n' "$out"
