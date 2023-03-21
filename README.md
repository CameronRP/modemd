# modemd

`modemd` together with `modemd.connrequester` control how the Raspberry Pi on the Thermal Camera accesses the internet.  If the wifi is working then that will be used.   If not then the USB modem plugged into the Raspberry Pi will be turned on.

Project | modemd
---|--- |
Platform | Thermal camera (Raspbian) |
Requires | <nothing> |
Build Status | [![Build Status](https://api.travis-ci.com/TheCacophonyProject/modemd.svg?branch=master)](https://travis-ci.com/TheCacophonyProject/modemd) |
Licence | GNU General Public License v3.0 |

## Instructions

Download and install the latest release from [Github](https://github.com/TheCacophonyProject/modemd/releases).  Then restart the device.

## Development Instructions

Follow our [go instructions](https://docs.cacophony.org.nz/home/developing-in-go) to download and build this project.

### Using modemd in an application

1. Make sure that modemd is running on your thermal-camera
2. Import "github.com/TheCacophonyProject/modemd/connrequester" into your go file
3. Before _each_ time your application needs to access the internet do the following:
```
	cr := connrequester.NewConnectionRequester()
	log.Println("requesting internet connection")
	cr.Start()
	cr.WaitUntilUpLoop(connectionTimeout, connectionRetryInterval, -1)
	log.Println("internet connection made")
	defer cr.Stop()
```

On some devices the USB modem only remains on for a short period after each connrequester call, so it is important to do this everytime.


### Releases
Releases are created using travis and git and saved [on Github](https://github.com/TheCacophonyProject/modemd/releases).   Follow our [release instructions](https://docs.cacophony.org.nz/home/creating-releases) to create a new release.

