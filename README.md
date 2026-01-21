An alternative launcher for "Hytale" 
 features:
 
 - Multiple version management
 - Run offline without needing a hytale account.
 - Universal "fake online" or  "online fix" for every version;
 
Currently for: Windows and Linux

(i need access to an ARM mac to make a mac os version) 

## Project layout

hytServer.go - hytale authentication server emulator "online fix"
hytAuth.go - implementation of hytale OAuth2.0, currently not used
hytFormats.go - most of the JSON structures used by hytale are here;
hytClient.go - downloading versions, etc
patch.go - itch.io's 'apply patch' wharf code
jwt.go - JSON structures used in hytale auth tokens
locations.go - default / location resolvers for many folders


Aurora/ - c code for the dll or shared object, loaded with the game, 
that replaces account-data.hytale.com with a custom server .. 

# Building

on windows, you first have to build the "Aurora.dll" using MSVC, 
and then you can use ``go build .``
or you can ``build-windows.bat`` within the VS2026 developer command prompt to do this;

on linux, you need ``build-essential`` and then you can build "Aurora.so" using its Makefile;
after that you can use ``go build .`` 
or you can use ``build-linux.sh``

# Alternative names

'hytLauncher may not be the final name for this project ..'
other names i considered using :

- hyTLauncher
- HytaleSP
- AnjoCaido's Hytale Launcher
- HytaleForFree.com

in all seriousness i kind of would want something a bit original xS
also this is not nessecarily planned to be a purely 'offline mode' launcher; 
i also want to add "premium" support as well-
mm the code for authentication flow is actually already here ..