# Ninja Sphere - ZigBee Driver

A golang Ninja Sphere driver for interacting with (ZigBee)[http://www.zigbee.org/] devices.

It uses https://github.com/ninjasphere/go-zigbee, which relies on TI Z-Stack Linux Gateway. The version of the gateway
that runs on the Sphere is not able to be released in source form due to a NDA, but it should work just fine with the TI
binary release from http://www.ti.com/tool/z-stack using a CC2531 USB dongle on other platforms.

## License

Copyright 2014 Ninja Blocks, Inc. All rights reserved.

Licensed under the MIT License
