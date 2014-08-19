#!/bin/bash

count=1
while [[ count -lt 2000000 ]]
do
  echo "Counter $count " >> count.txt
  count=$(( $count + 1 ))
done


