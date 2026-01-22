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

	"bitbucket.org/rj/goey"
	"bitbucket.org/rj/goey/base"
	"bitbucket.org/rj/goey/loop"
	"bitbucket.org/rj/goey/windows"
	"github.com/sqweek/dialog"
)

type launcherCommune struct {
	Patchline string `json:"last_patchline"`
	Username string `json:"last_username"`
	SelectedVersion int `json:"last_version"`
	LatestVersions map[string]int `json:"last_version_scan_result"`
	GameFolder string `json:"install_directory"`
	Mode string `json:"mode"`
	AuthTokens *accessTokens `json:"token"`
	Profiles *[]accountInfo `json:"profiles"`
	SelectedProfile int `json:"selected_profile"`
}


var(
	wMainWin *windows.Window
	wCommune = launcherCommune {
		Patchline: "release",
		Username: "TransRights",
		LatestVersions: map[string]int{
			"release": 4,
			"pre-release": 8,
		},
		SelectedVersion: 4,
		GameFolder: DefaultGameFolder(),
		Mode: "fakeonline",
		AuthTokens: nil,
		Profiles: nil,
		SelectedProfile: 0,
	};
	wProgress = 0
	wDisabled = false
)

func doAuthentication() {
	aTokens, err := getAuthTokens(wCommune.AuthTokens);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Failed to get auth tokens: %s", err), "Auth failed.");
		wCommune.AuthTokens = nil;
		wCommune.Mode = "fakeonline";
		loop.Do(func() error {
			updateWindow();
			writeSettings();
			return nil;
		});
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
			loop.Do(func() error {
				updateWindow();
				writeSettings();
				return nil;
			});
		}
	}
}

func authenticatedCheckForUpdatesAndGetProfileList() {
	if wCommune.AuthTokens == nil {
		return;
	}

	lData, err := getLauncherData(*wCommune.AuthTokens, runtime.GOARCH, runtime.GOOS);

	if err != nil {
		showErrorDialog(fmt.Sprintf("Failed to get launcher data: %s", err), "Auth failed.");
		wCommune.AuthTokens = nil;
		wCommune.Mode = "fakeonline";
		loop.Do(func() error {
			updateWindow();
			writeSettings();
			return nil;
		});
	}

	lastReleaseVersion := wCommune.LatestVersions["release"];
	latestReleaseVersion := lData.Patchlines.Release.Newest;

	lastPreReleaseVersion := wCommune.LatestVersions["pre-release"];
	latestPreReleaseVersion := lData.Patchlines.PreRelease.Newest;

	if lastReleaseVersion > latestReleaseVersion {
		fmt.Printf("found new release: %d\n", lastReleaseVersion)
		wCommune.LatestVersions["release"] = latestReleaseVersion;
	}
	if lastPreReleaseVersion > latestPreReleaseVersion {
		fmt.Printf("found new release: %d\n", lastPreReleaseVersion)
		wCommune.LatestVersions["pre-release"] = latestPreReleaseVersion;
	}

	wCommune.Profiles = &lData.Profiles;

	if wMainWin != nil {
		loop.Do(func() error {
			updateWindow();
			writeSettings();
			return nil;
		});
	}
}

func reAuthenticate() {
	if wCommune.AuthTokens != nil {
		aTokens, err:= getAuthTokens(*wCommune.AuthTokens);

		if err != nil {
			fmt.Printf("Failed to get auth token: %s\n",err);
			return;
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
			updateWindow()
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
									fmt.Printf("failed to remove: %s", err);
									wMainWin.Message(fmt.Sprintf("failed to remove: %s", err)).WithError().WithTitle("Failed to remove").Show();
								}
								wDisabled = false;
								updateWindow();
							}();
						},
					},
				},
			},
		},
	};
}

func installLocation() base.Widget {
	return &goey.VBox {
			AlignMain: goey.SpaceBetween,
			Children: []base.Widget {
				&goey.HR{},
				&goey.Label{ Text: "Install Location:" },
				&goey.HBox{
					AlignCross: goey.CrossCenter,
					Children: []base.Widget {
						&goey.Expand{
							Child: &goey.TextInput{
								Placeholder: "Install Location",
								Disabled: wDisabled,
								Value: wCommune.GameFolder,
								OnChange: func(v string) {
									wCommune.GameFolder = v;
									updateWindow();
								},
							},
						},
						&goey.Button{
							Text: "Browse",
							Disabled: wDisabled,
							OnClick: func() {
								dir, err := dialog.Directory().Title("Select install location").Browse();
								if err != nil {
									if err != dialog.ErrCancelled {
										errorMsg := fmt.Sprintf("Failed: %s", err);
										wMainWin.Message(errorMsg).WithError().WithTitle("Error reading directory.").Show();
									}
								}

								wCommune.GameFolder = dir;
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

func usernameBox() base.Widget {
	usernameDisabled := wDisabled || wCommune.Mode == "authenticated";

	return &goey.VBox{
		AlignMain: goey.SpaceBetween,
		Children: []base.Widget{
			&goey.Label{Text: "Username:"},
			&goey.TextInput{
					Value: wCommune.Username,
					Placeholder: "Username",
					Disabled: usernameDisabled,
					OnChange: func(v string) {
						wCommune.Username = v;
					},
			},
		},
	};
}

func loginBox() base.Widget {

	logoutDisabled := wDisabled || wCommune.AuthTokens == nil;
	loginDisabled := wDisabled || wCommune.AuthTokens != nil;
	profileList := []string{};

	if wCommune.Mode != "authenticated" {
		return &goey.Empty{};
	}

	if wCommune.Profiles != nil {
		for _, profile := range *wCommune.Profiles {
			profileList = append(profileList, profile.Username);
		}
	}

	profilesDisabled := wDisabled || wCommune.Profiles == nil;


	return &goey.VBox {
		AlignMain: goey.MainStart,
		Children: []base.Widget {
			&goey.HBox{
				AlignCross: goey.CrossCenter,
				AlignMain: goey.Homogeneous,
				Children: []base.Widget {
					&goey.Button{
						Text: "Login (OAuth 2.0)",
						Disabled: loginDisabled,
						OnClick: func() {
							atokens, err := getAuthTokens(nil);
							if err != nil {
								showErrorDialog(fmt.Sprintf("Failed to login: %s", err), "Login error");
								return;
							}

							wCommune.AuthTokens = &atokens;
							writeSettings();
							updateWindow();
						},
					},
					&goey.Button{
						Text: "Logout",
						Disabled: logoutDisabled,
						OnClick: func() {
							wCommune.AuthTokens = nil;
							writeSettings();
							updateWindow();
						},
					},
				},
			},
			&goey.SelectInput{
				Items: profileList,
				OnChange: func(v int) {
					wCommune.SelectedProfile = v;
					wCommune.Username = profileList[v];
					loop.Do(func() error {
						updateWindow();
						return nil;
					});
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
		wProgress = newProgress
		loop.Do(func() error {
			updateWindow();
			return nil;
		});
	}
}

func createWindow() error {
	w, err := windows.NewWindow("HytaleSP", renderWindow())
	if err != nil {
		return err
	}

	w.SetScroll(false, false);

	w.SetOnClosing(func() bool {
			writeSettings();
			return false;
	});

	wMainWin = w;


	f, err := embeddedImages.Open(path.Join("Resources", "icon.png"));
	if err != nil {
		return nil;
	}
	defer f.Close()

	image, _, err := image.Decode(f)
	w.SetIcon(image);

	return nil
}


func updateWindow() {
	err := wMainWin.SetChild(renderWindow())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
	}
}

func showErrorDialog(msg string, title string) {
		loop.Do(func() error {
			wMainWin.Message(msg).WithError().WithTitle(title).Show();
			return nil;
		})

}

func renderWindow() base.Widget {
	return &goey.Padding{
		Insets: goey.DefaultInsets(),
		Child: &goey.Align{
			Child: &goey.VBox{
				AlignMain: goey.MainStart,
				Children: []base.Widget{

					usernameBox(),
					patchLineMenu(),
					versionMenu(),

					&goey.Progress{
						Value: wProgress,
						Min: 0,
						Max: 100,
					},
					&goey.Button{
						Text: "Start Game",
						Disabled: wDisabled,
						OnClick: func() {
							go func() {

								// disable the current window
								loop.Do(func() error {
									wDisabled = true;
									updateWindow();
									return nil;
								});


								// enable the window again once done
								defer func() {
									wDisabled = false;
									loop.Do(func() error {
										updateWindow();
										return nil;
									});
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

								err = launchGame(wCommune.SelectedVersion, wCommune.Patchline, wCommune.Username, usernameToUuid(wCommune.Username));

								if err != nil {
									showErrorDialog(fmt.Sprintf("Error running the game: %s", err), "Run game failed.");
									return;
								};

							}();


						},
					},
					installLocation(),
					modeSelector(),
					loginBox(),
				},
			},
		},
	}
}



func main() {

	os.MkdirAll(MainFolder(), 0775);
	os.MkdirAll(LauncherFolder(), 0775);
	os.MkdirAll(UserDataFolder(), 0775);
	os.MkdirAll(JreFolder(), 0775);
	os.MkdirAll(ServerDataFolder(), 0775);
	readSettings();
	os.MkdirAll(GameFolder(), 0775);

	go reAuthenticate();
	go checkForUpdates();

	err := loop.Run(createWindow)
	if err != nil {
		fmt.Println("Error: ", err)
	}

}
