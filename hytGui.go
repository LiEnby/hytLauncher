//go:generate goversioninfo

package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"

	"git.silica.codes/Li/goey"
	"git.silica.codes/Li/goey/base"
	"git.silica.codes/Li/goey/loop"
	"git.silica.codes/Li/goey/windows"

	"github.com/sqweek/dialog"
)

type launcherCommune struct {
	Patchline string `json:"last_patchline"`
	Username string `json:"last_username"`
	SelectedVersion int `json:"last_version"`
	LatestVersions map[string]int `json:"last_version_scan_result"`
	Mode string `json:"mode"`

	// authentication
	AuthTokens *accessTokens `json:"token"`
	Profiles *[]accountInfo `json:"profiles"`
	SelectedProfile int `json:"selected_profile"`

	// settings
	GameFolder string `json:"install_directory"`
	UserDataFolder string `json:"userdata_directory"`
	JreFolder string `json:"jre_directory"`
	UUID string `json:"uuid_override"`
}


const DEFAULT_USERNAME = "TransRights";
const DEFAULT_PATCHLINE = "release";

var (
	wMainWin *windows.Window
	wCommune = launcherCommune {
		Patchline: DEFAULT_PATCHLINE,
		Username: DEFAULT_USERNAME,
		LatestVersions: map[string]int{
			"release": 5,
			"pre-release": 12,
		},
		SelectedVersion: 4,
		Mode: "fakeonline",
		AuthTokens: nil,
		Profiles: nil,
		SelectedProfile: 0,

		GameFolder: DefaultGameFolder(),
		UserDataFolder: DefaultUserDataFolder(),
		JreFolder: DefaultJreFolder(),
		UUID: "",
	};
	wProgress = 0
	wDisabled = false
	wSelectedTab = 0

	wAdvanced = false
	w = false
)




func doAuthentication() {
	aTokens, err := getAuthTokens(wCommune.AuthTokens);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Failed to get auth tokens: %s", err), "Auth failed.");
		wCommune.AuthTokens = nil;
		wCommune.Mode = "fakeonline";
		writeSettings();
		loop.Do(updateWindow);
	}

	wCommune.AuthTokens = &aTokens;

	// get profile list ..
	authenticatedCheckForUpdatesAndGetProfileList();

}


func checkForUpdates() {
	if wCommune.Mode != "authenticated" {
		lastRelease := wCommune.LatestVersions["release"]
		lastPreRelease := wCommune.LatestVersions["pre-release"]

		latestRelease := findLatestVersionNoAuth(lastRelease, runtime.GOARCH, runtime.GOOS, "release");
		latestPreRelease := findLatestVersionNoAuth(lastPreRelease, runtime.GOARCH, runtime.GOOS, "pre-release");

		fmt.Printf("latestRelease: %d\n", latestRelease);
		fmt.Printf("latestPreRelease: %d\n", latestPreRelease);

		if latestRelease > lastRelease {
			fmt.Printf("Found new release version: %d\n", latestRelease);
			wCommune.LatestVersions["release"] = latestRelease;
		}

		if latestPreRelease > lastPreRelease {
			fmt.Printf("Found new pre-release version: %d\n", latestPreRelease);
			wCommune.LatestVersions["pre-release"] = latestPreRelease;
		}

		if wMainWin != nil {
			loop.Run(updateWindow);
			writeSettings();
		}
	}
}

func authenticatedCheckForUpdatesAndGetProfileList() {
	if wCommune.AuthTokens == nil {
		return;
	}
	if(wCommune.Mode != "authenticated") {
		return;
	}

	lData, err := getLauncherData(*wCommune.AuthTokens, runtime.GOARCH, runtime.GOOS);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Failed to get launcher data: %s", err), "Auth failed.");
		wCommune.AuthTokens = nil;
		wCommune.Mode = "fakeonline";
		go func() {
			loop.Do(updateWindow);
		}();
		writeSettings();
	}

	lastReleaseVersion := wCommune.LatestVersions["release"];
	latestReleaseVersion := lData.Patchlines.Release.Newest;

	lastPreReleaseVersion := wCommune.LatestVersions["pre-release"];
	latestPreReleaseVersion := lData.Patchlines.PreRelease.Newest;

	if latestReleaseVersion > lastReleaseVersion {
		fmt.Printf("found new release: %d\n", lastReleaseVersion)
		wCommune.LatestVersions["release"] = latestReleaseVersion;
	}
	if latestPreReleaseVersion > lastPreReleaseVersion {
		fmt.Printf("found new release: %d\n", lastPreReleaseVersion)
		wCommune.LatestVersions["pre-release"] = latestPreReleaseVersion;
	}

	wCommune.Profiles = &lData.Profiles;

	if wMainWin != nil {
		loop.Do(updateWindow);
		writeSettings();
	}
}

func reAuthenticate() {
	if wCommune.AuthTokens != nil && wCommune.Mode == "authenticated" {
		aTokens, err:= getAuthTokens(*wCommune.AuthTokens);

		if err != nil {
			showErrorDialog(fmt.Sprintf("Failed to authenticate: %s", err), "Auth failed.");
			wCommune.AuthTokens = nil;
			wCommune.Mode = "fakeonline";
			loop.Do(updateWindow);
			writeSettings();
		}

		wCommune.AuthTokens = &aTokens;
		authenticatedCheckForUpdatesAndGetProfileList();
	}
}

func writeSettings() {
	fmt.Printf("Saving settings ...\n");
	jlauncher, _ := json.Marshal(wCommune);

	err := os.MkdirAll(filepath.Dir(getLauncherJson()), 0666);
	if err != nil {
		fmt.Printf("error writing settings: %s\n", err);
		return;
	}

	err = os.WriteFile(getLauncherJson(), jlauncher, 0666);
	if err != nil {
		fmt.Printf("error writing settings: %s\n", err);
		return;
	}
}

func getDefaultSettings() {
	writeSettings();
	go checkForUpdates();

}

func getLauncherJson() string {
	return filepath.Join(LauncherFolder(), "launcher.json");
}

func readSettings() {
	_, err := os.Stat(getLauncherJson())
	if err != nil {
		getDefaultSettings();
	} else {
		data, err := os.ReadFile(getLauncherJson());
		if err != nil{
			getDefaultSettings();
			return;
		}
		json.Unmarshal(data, &wCommune);

		if wCommune.GameFolder != GameFolder() {
			wCommune.GameFolder = GameFolder();
		}

		fmt.Printf("Reading last settings: \n");
		fmt.Printf("username: %s\n", wCommune.Username);
		fmt.Printf("patchline: %s\n", wCommune.Patchline);
		fmt.Printf("last used version: %d\n", wCommune.SelectedVersion);
		fmt.Printf("newest known release: %d\n", wCommune.LatestVersions["release"])
		fmt.Printf("newest known pre-release: %d\n", wCommune.LatestVersions["pre-release"])

	}
}


func valToChannel(vchl int) string {
	switch vchl {
		case 0:
			return "release";
		case 1:
			return "pre-release";
		default:
			return "release";
	}
}

func channelToVal(channel string) int {
	switch channel {
		case "release":
			return 0;
		case "pre-release":
			return 1;
		default:
			return 0;
	}
}

func startGame() {
	// disable the current window
	wDisabled = true;
	loop.Do(updateWindow);

	// enable the window again once done
	defer func() {
		wDisabled = false;
		loop.Do(updateWindow);
	}();

	err := installJre(updateProgress);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Error getting the JRE: %s", err), "Install JRE failed.");
		return;
	};

	err = installGame(wCommune.SelectedVersion, wCommune.Patchline, updateProgress);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Error getting the game: %s", err), "Install game failed.");
		return;
	};

	err = launchGame(wCommune.SelectedVersion, wCommune.Patchline, wCommune.Username, getUUID());

	if err != nil {
		showErrorDialog(fmt.Sprintf("Error running the game: %s", err), "Run game failed.");
		return;
	};
}

func patchLineMenu() base.Widget {
	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Patchline:"},
			&goey.SelectInput{
				Items: []string {
					"release",
					"pre-release",
				},
				Value: channelToVal(wCommune.Patchline),
				Disabled: wDisabled,
				OnChange: func(v int) {
					if wCommune.Patchline != valToChannel(v) {
						wCommune.Patchline = valToChannel(v);
						wCommune.SelectedVersion = wCommune.LatestVersions[wCommune.Patchline];
						updateWindow();
					}
				},
			},
		},
	};
}


func versionMenu() base.Widget {
	versions := goey.SelectInput {
		OnChange: func(v int) {
			wCommune.SelectedVersion = v+1;
			updateWindow();
		},
		Disabled: wDisabled,
	};
	latest := wCommune.LatestVersions[wCommune.Patchline];

	for i := range latest {
		txt := "Version "+strconv.Itoa(i+1);
		if isGameVersionInstalled(i+1, wCommune.Patchline) {
			txt += " - installed";
		} else {
			txt += " - not installed";
		}

		versions.Items = append(versions.Items, txt);
	}

	selectedVersion := wCommune.SelectedVersion;
	selectedChannel := wCommune.Patchline;

	versions.Value = (selectedVersion-1);
	disabled := !isGameVersionInstalled(selectedVersion, selectedChannel) || wDisabled;

	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Version:"},

			&goey.HBox{
				AlignCross: goey.CrossCenter,
				Children: []base.Widget{
					&goey.Expand{
						Child: &versions,
					},
					&goey.Button {
						Text: "Delete",
						Disabled: disabled,
						OnClick: func() {
							wDisabled = true;
							updateWindow();

							go func() {
								installDir := getVersionInstallPath(selectedVersion, wCommune.Patchline);
								err := os.RemoveAll(installDir);
								if err != nil {
									showErrorDialog(fmt.Sprintf("failed to remove: %s", err), "failed to remove");
								}

								wDisabled = false;
								loop.Do(updateWindow);
							}();
						},
					},
				},
			},
		},
	};
}


func labeledTextInput(label string, value *string, disabled bool) base.Widget {
	if value == nil {
		panic("failed to initalize browse button");
	}

	isDisabled := wDisabled || disabled;

	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget {
			&goey.Label{ Text: label + ": " },
			&goey.TextInput{
				Placeholder: label,
				Disabled: isDisabled,
				Value: *value,
				OnChange: func(v string) {
					*value = v;
					updateWindow();
				},
			},
		},
	};
}

func browseButton(label string, value *string) base.Widget {
	if value == nil {
		panic("failed to initalize browse button");
	}

	return &goey.VBox {
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget {
			&goey.Label{ Text: label +": " },
			&goey.HBox{
				AlignCross: goey.CrossCenter,
				Children: []base.Widget {
					&goey.Expand{
						Child: &goey.TextInput{
							Placeholder: label,
							Disabled: wDisabled,
							Value: *value,
							OnChange: func(v string) {
								*value = v;
								updateWindow();
							},
						},
					},
					&goey.Button{
						Text: "Browse",
						Disabled: wDisabled,
						OnClick: func() {
							dir, err := dialog.Directory().Title("Select "+label).Browse();
							if err != nil {
								if err != dialog.ErrCancelled {
									showErrorDialog(fmt.Sprintf("Failed: %s", err), "Error reading directory");
								}
							}

							*value = dir;
							updateWindow();
						},
					},
				},
			},
		},
	};
}



func modeSelector () base.Widget {
	var v int;

	switch(wCommune.Mode) {
		case "offline":
			v = 0;
		case "fakeonline":
			v = 1;
		case "authenticated":
			v = 2;
	}

	return &goey.VBox {
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Launch Mode:"},
			&goey.SelectInput{
				Items: []string {
					"Offline Mode",
					"Fake Online Mode",
					"Authenticated",
				},
				Value: v,
				Disabled: wDisabled,
				OnChange: func(v int) {
					switch(v) {
						case 0:
							wCommune.Mode = "offline";
						case 1:
							wCommune.Mode = "fakeonline";
						case 2:
							wCommune.Mode = "authenticated";
					}
					updateWindow();
				},
			},
		},
	}
}



func drawAuthenticatedSettings() base.Widget {

	if wCommune.Mode != "authenticated" {
		return &goey.Empty{}
	}

	logoutDisabled := wDisabled || (wCommune.AuthTokens == nil);
	loginDisabled := wDisabled || (wCommune.AuthTokens != nil);
	profileList := []string{};

	if wCommune.Profiles != nil {
		for _, profile := range *wCommune.Profiles {
			profileList = append(profileList, profile.Username);
		}
	}
	profilesDisabled := wDisabled || wCommune.Profiles == nil;
	return &goey.VBox {
		AlignMain: goey.MainStart,
		Children: []base.Widget {
			drawDivider("Authentication"),
			&goey.HBox{
				AlignCross: goey.CrossCenter,
				AlignMain: goey.Homogeneous,
				Children: []base.Widget {
					&goey.Button{
						Text: "Login (OAuth 2.0)",
						Disabled: loginDisabled,
						OnClick: func() {
							go doAuthentication();
						},
					},
					&goey.Button{
						Text: "Logout",
						Disabled: logoutDisabled,
						OnClick: func() {
							wCommune.AuthTokens = nil;
							wCommune.Profiles = nil;
							writeSettings();
							updateWindow();
						},
					},
				},
			},
			&goey.Label{
				Text: "Select profile",
			},
			&goey.SelectInput{
				Items: profileList,
				OnChange: func(v int) {
					wCommune.SelectedProfile = v;
					wCommune.Username = profileList[v];
					updateWindow();
				},
				Value: wCommune.SelectedProfile,
				Disabled: profilesDisabled,
			},
		},
	};
}


func updateProgress(done int64, total int64) {
	lastProgress := wProgress;
	newProgress := int((float64(done) / float64(total)) * 100.0);

	if newProgress != lastProgress {
		wProgress = newProgress;
		loop.Do(updateWindow);
	}
}

func createDownloadProgress () base.Widget {
	return &goey.Progress{
		Value: wProgress,
		Min: 0,
		Max: 100,
	};
}

func drawStartGame() base.Widget{
	return &goey.VBox {
		AlignMain: goey.MainStart,
		Children: []base.Widget {
			labeledTextInput("Username", &wCommune.Username, wCommune.Mode == "authenticated"),
			modeSelector(),
			drawDivider("Version"),
			patchLineMenu(),
			versionMenu(),
			createDownloadProgress(),
			&goey.Button{
				Text: "Start Game",
				Disabled: wDisabled,
				OnClick: func() {
					go startGame();
				},
			},
			drawAuthenticatedSettings(),
		},
	}
}

func drawDivider(label string) base.Widget {
	return &goey.HBox{
		AlignCross: goey.CrossCenter,
		Children: []base.Widget{
			&goey.Label{
				Text: label,
			},
			&goey.Expand{
				Child: &goey.HR{},
			},
		},
	};
}



func drawSettings() base.Widget{
	return &goey.VBox{
		AlignMain: goey.MainStart,
		Children: []base.Widget {
			drawDivider("Directories"),
			browseButton("Game Location", &wCommune.GameFolder),
			browseButton("JRE Location", &wCommune.JreFolder),
			browseButton("Game UserData Location", &wCommune.UserDataFolder),
			drawDivider("Advanced"),
			labeledTextInput("★UUID Override", &wCommune.UUID, wCommune.Mode == "authenticated"),
		},
	};
}


func drawWidgets() base.Widget {
	return &goey.Tabs {
		Value: wSelectedTab,
		OnChange: func( v int ) {wSelectedTab = v;},
		Children: []goey.TabItem {
			{
				Caption: "Game",
				Child: drawStartGame(),
			},
			{
				Caption: "Settings",
				Child: drawSettings(),
			},
		},
	};

}

func createWindow() error {

	win, err := windows.NewWindow("HytaleSP", drawWidgets())
	if err != nil {
		return err
	}

	win.SetScroll(false, false);

	win.SetOnClosing(func() bool {
		writeSettings();
		return false;
	});


	f, err := embeddedImages.Open(path.Join("Resources", "icon.png"));
	if err != nil {
		return nil;
	}
	defer f.Close()

	image, _, err := image.Decode(f)
	win.SetIcon(image);

	wMainWin = win;

	return nil
}

func updateWindow() error {
	if wMainWin == nil {
		return fmt.Errorf("Failed to update window because the window ptr is nil.");
	}

	err := wMainWin.SetChild(drawWidgets());

	if err != nil {
		showErrorDialog(fmt.Sprintf("error updating window: %s", err), "error updating window");
		return err;
	}
	return nil;
}

func showErrorDialog(msg string, title string) {
		loop.Do(func() error {
			wMainWin.Message(msg).WithError().WithTitle(title).Show();
			return nil;
		});
}




func main() {

	os.MkdirAll(MainFolder(), 0775);
	os.MkdirAll(LauncherFolder(), 0775);
	os.MkdirAll(ServerDataFolder(), 0775);
	readSettings();

	os.MkdirAll(UserDataFolder(), 0775);
	os.MkdirAll(JreFolder(), 0775);
	os.MkdirAll(GameFolder(), 0775);

	go reAuthenticate();
	go checkForUpdates();

	err := loop.Run(createWindow)
	if err != nil {
		fmt.Println("Error: ", err)
	}

}
