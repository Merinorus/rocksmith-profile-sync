# Rocksmith Profile Sync
Synchronize multiple Rocksmith 2014 user profiles across multiple machines.

/!\ This was an experimental app quickly written in March 2016. I decided to publish it in case someone might find it useful. It may help to understand how are written Rocksmith user profiles with some reverse engineering. 

However, I wrote this documentation in 2021 thanks to old .txt notes without testing this app.
It is not guaranteed to work (actually I couldn't make it work at the time I'm writing this), so you will need to do some research.
Feel free to use my work or to contact me if you want to make this work again. If I have time I might have a try... but you know what they say: it's far easier to write code than to read code.

The goal was to be able to run this tool out of the box, without installing any dependency on my friends' PC (Python...). At the same time, I wanted to learn a new language, so I decided to write it in Go to have only one `.exe` file.
Thus, this was my first application written in Go, so don't expect any good programmer practice there. :>

## Ok, but what does it do exactly?

On Rocksmith, you learn to play songs with a real guitar that is connected to your PC.
Your progress is synchronized on a profile, which is saved locally.
If you bought the game on Steam, your profile should also be synchronized in the cloud.

Several profiles can be created on the same PC so several people can save their progression. They can also play in a "screen share" mode (ie: both on the same computer, playing a song at the same time), and the progression of each player will be saved in their profile.

In my case, we were two friends playing individually, each one on his PC. But sometimes we wanted to play together, so he would join me and play on my PC, but he had to create a new profile and couldn't save his progression when he would go back on his PC.
Rocksmith did not allow that... so I made a tool to synchronize both profiles on both computers.

### Technical details
When creating multiple profiles on your machines, Rocksmith will create several files. Example with two profiles:
- 089827cfb2d9463586ccc7a2212198fb_prfldb: profile 1 data
- e9074fb34d1445e59be7880a496bd5d3_prfldb: profile 2 data
- localprofiles.json: file keeping a track of all local profiles. If I remember, this is changed only when profiles are created or deleted so it rarely changes (this will be important for the next step).

So, let assume I am profile 1, playing on PC 1. My friend is profile 2, playing on PC 2.

PC1 Gamedata <-> PC1 Sync Folder <- (cloud sync) -> PC2 Sync Folder <--> PC2 Gamedata

When finishing a song on my PC 1, "profile 1 data" will be updated. While running in the background, the "Rocksmith Profile Sync" tool will copy the profile 1 data to the sync folder. It will then be synchronized with PC 2 thanks to cloud sync (in my case Syncthing, but you could use Dropbox, GDrive, Onedrive...) and, eventually, my progress will be available on PC 2. And vice versa.

When playing both on PC 1 on the same song, both profiles will be sent to PC 2.

Enjoy multiplayer synchronization!!

### Limitations
- Supports more than two PC! It's limited to the number of in-game profiles.

- If two people create or delete different profiles at the same time, the synchronization will be messed up as localprofiles.json will be modified in both machines and won't be kept in sync. So be synchronized "in real life" with your friends. There is very basic security in the tool (instead of removing a file, it adds a ".old" extension) but you should use a "recycle bin" feature to avoid losing game data permanently.

- If someone forgets to launch the sync tool while running the game, profiles won't be kept in sync and an undefined situation may occur. :>


## Installation steps

### 1- Install a synchronization tool between machines

I remember I used Syncthing. You may try using other sync tools, it should work as long as they are running in the background. Syncthing can work in local mode in case your internet connection is down.

### 2- Make a backup of your Rocksmith profile

Did I tell you to make a backup of your profile? Do it. Just do it.


### 3- Locate the "Storage" folder
I don't remember the location, but it should be the folder that contains the profiles files and the "localprofiles.json" file.


### 4- Change the following variables in main.go

```Go
// Location of the game data storage directory.
// "." by default, it means that the tool executable is put directly in this directory.
var storageDirectory = "."
// Name of the log file. Will be written in the storage directory.
var logFileName = "rocksmithprofilesync.log"
// Location of the "cloud sync" folder.
// Located next to the storage directory by default if you use Syncthing.
// You may have to change it if it's OneDrive, Dropbox etc.
var syncDirectory = "..\\Sync"
```

### 5- Build the executable
If you want to execute the synchronization without any window (headless):

`go build -o rocksmithprofilesync.exe -ldflags="-H windowsgui" main.go`

For debugging purposes, you can build it to be run as a console application:

`go build -o rocksmithprofilesync.exe main.go`

### 6- First install only: create profiles

Create profiles on PC 1. PC 2 should not have any profile.
This ensures there is no collision.
Then, copy only the localprofiles.json on PC 2

### 7- Launch the executable and the cloud sync before running the game!

Be sure to run both the tool and the cloud synchronization. You can launch cloud sync at startup and:
- launch the tool at startup,
- OR create a .bat or .ps1 file to run the tool before you run the game.

I don't have the .bat file anymore, sorry.

### 8- Verify profiles are synchronized

If all PC are running both cloud sync and this tool, you should see all profiles kept in sync on all machines. Congrats!