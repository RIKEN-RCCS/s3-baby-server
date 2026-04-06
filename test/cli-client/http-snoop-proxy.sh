#!/bin/ksh

# This runs a proxy that dumps http traffic: port=9001 (client side)
# to port=9000 (server side).

# Crazy quotations are by https://stackoverflow.com/questions/1250079.

# ncat -lkv 127.0.0.1 9001 -c 'tee /dev/stderr | ncat -v 127.0.0.1 9000 | tee /dev/stderr'

ncat -lkv 127.0.0.1 9001 -c 'tee >(awk '"'"'{print "> "$0}'"'"' > /dev/stderr) | ncat -v 127.0.0.1 9000 | tee >(awk '"'"'{print "< "$0}'"'"' > /dev/stderr)'
