#!/bin/bash

rm -f guacamole-keyboard.js

curl -o guacamole-keyboard.js \
https://raw.githubusercontent.com/apache/guacamole-client/refs/heads/main/guacamole-common-js/src/main/webapp/modules/Keyboard.js

gzip guacamole-keyboard.js
