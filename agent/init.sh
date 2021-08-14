#!/busybox

START=$(date -u +%s)

while true
do
  CURRENT=$(date -u +%s)
  echo $(($CURRENT - $START)) seconds since the container was started
  sleep 10
done